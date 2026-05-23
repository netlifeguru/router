package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func (r *Router) isConsoleLoggingEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.Level(-10))
}

func (r *Router) MultiListenAndServe(listeners Listeners) error {

	consoleEnabled := r.isConsoleLoggingEnabled()

	workers := runtime.NumCPU()
	runtime.GOMAXPROCS(workers)

	if consoleEnabled {
		slog.Info("Starting server")
		slog.Info("System resources", "cpu_cores", workers)
	}

	var (
		wg           sync.WaitGroup
		servers      []*http.Server
		mu           sync.Mutex
		startErrChan = make(chan error, len(listeners)*workers+1)
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for _, ln := range listeners {
		listenAddr := ln.Addr

		_, portStr, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return fmt.Errorf("%w: %q: %w", ErrInvalidListenAddress, listenAddr, err)
		}
		_, err = strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("%w: %q: %w", ErrInvalidListenPort, listenAddr, err)
		}

		if consoleEnabled {
			slog.Info("web server started",
				"server", serverName,
				"version", serverVersion,
				"listen_addr", listenAddr,
			)
		}

		useReusePort := runtime.GOOS != "windows"
		var reuseErr error

		if useReusePort {
			lc := net.ListenConfig{
				Control: func(network, address string, c syscall.RawConn) error {
					var sockErr error
					if err := c.Control(func(fd uintptr) {
						_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
						if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); e != nil {
							sockErr = e
						}
					}); err != nil {
						return err
					}
					return sockErr
				},
			}

			firstListener, err := lc.Listen(context.Background(), "tcp", listenAddr)

			if err == nil {
				for i := 0; i < workers; i++ {
					var listener net.Listener

					if i == 0 {
						listener = firstListener
					} else {
						var lErr error
						listener, lErr = lc.Listen(context.Background(), "tcp", listenAddr)
						if lErr != nil {
							slog.Error("REUSEPORT listen failed", "worker", i, "err", lErr)
							continue
						}
					}

					wg.Add(1)
					go func(l net.Listener, addr string) {
						defer wg.Done()

						server := &http.Server{
							Handler:           r.handler(),
							ReadTimeout:       5 * time.Second,
							WriteTimeout:      10 * time.Second,
							IdleTimeout:       120 * time.Second,
							ReadHeaderTimeout: 2 * time.Second,
						}

						mu.Lock()
						servers = append(servers, server)
						mu.Unlock()

						if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
							select {
							case startErrChan <- fmt.Errorf("%w: %q: %w", ErrServeFailed, addr, err):
							default:
							}
						}
					}(listener, listenAddr)
				}
			} else {
				useReusePort = false
				reuseErr = err
			}
		}

		if !useReusePort {
			slog.Error("REUSEPORT unavailable; falling back to single listener",
				"listen_addr", listenAddr,
				"error", reuseErr,
			)

			l, err := net.Listen("tcp", listenAddr)
			if err != nil {
				return fmt.Errorf("%w: %q: %w", ErrListenFailed, listenAddr, err)
			}

			wg.Add(1)
			go func(l net.Listener, addr string) {
				defer wg.Done()

				server := &http.Server{
					Handler:           r.handler(),
					ReadTimeout:       5 * time.Second,
					WriteTimeout:      10 * time.Second,
					IdleTimeout:       120 * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				mu.Lock()
				servers = append(servers, server)
				mu.Unlock()

				if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
					select {
					case startErrChan <- fmt.Errorf("%w: %q: %w", ErrServeFailed, addr, err):
					default:
					}
				}
			}(l, listenAddr)
		}
	}

	select {
	case err := <-startErrChan:
		r.shutdownServers(servers, &mu, consoleEnabled)
		return err
	case <-stop:
	}

	r.SetReady(false)

	if consoleEnabled {
		slog.Info("Shutdown signal received. Shutting down servers...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mu.Lock()
	for _, srv := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			if err := s.Shutdown(shutdownCtx); err != nil {
				if consoleEnabled {
					slog.Info("Server shutdown error", "error", err)
				}
			}
		}(srv)
	}
	mu.Unlock()

	wg.Wait()

	if consoleEnabled {
		slog.Info("All servers shut down gracefully.")
	}

	return nil
}

func (r *Router) shutdownServers(servers []*http.Server, mu *sync.Mutex, consoleEnabled bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mu.Lock()
	defer mu.Unlock()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			if consoleEnabled {
				slog.Error("Shutdown error during cleanup", "err", err)
			}
		}
	}
}

func (r *Router) ListenAndServe(addr string) error {
	return r.MultiListenAndServe(Listeners{
		{Addr: addr},
	})
}
