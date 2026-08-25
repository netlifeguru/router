package router

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (r *Router) write405(w http.ResponseWriter, mask int) {
	if allow := r.maskToAllowHeader(mask); allow != "" {
		w.Header().Set("Allow", allow)
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = io.WriteString(w, "405 method not allowed")
}

func (r *Router) validatePathEntry(ctx *Context, entry *routeEntry) bool {
	patterns := entry.Patterns
	segments := ctx.segments

	for i := 0; i < len(patterns); i++ {
		p := patterns[i]
		segment := segments[i].Value

		switch p.Type {
		case isString:
		case isMatch:
			if !p.RegexCompiled.MatchString(segment) {
				return false
			}

		case isPattern:
			if !p.Fn(segment) {
				return false
			}

		case isSubmatch:
			if !p.RegexCompiled.MatchString(segment) {
				return false
			}
		}
	}

	return true
}

func (r *Router) finishRequest(w http.ResponseWriter, req *http.Request, ctx *Context) {
	if message := recover(); message != nil {
		r.logError(req, message)

		if r.recovery != nil {
			func() {
				defer func() {
					if recoveryPanic := recover(); recoveryPanic != nil {
						r.logError(req, recoveryPanic)

						http.Error(w, "Recovery middleware failed: an error occurred while executing the recovery handler.", http.StatusInternalServerError)
					}
				}()

				r.recovery(w, req, ctx)
			}()
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}

	if ctx != nil {
		putContext(ctx)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := getContext()

	defer r.finishRequest(w, req, ctx)

	var foundPath bool
	var allowedMask int

	t := r.staticRoutes[req.URL.Path]

	bitmask := r.getBitmaskIndex(req.Method)

	if len(t) > 0 {
		for i := 0; i < len(t); i++ {
			entry := &t[i]
			if entry.Bitmask&bitmask != 0 {
				ctx.handler = entry
				entry.Handler(w, req, ctx)
				return
			}
			allowedMask |= entry.Bitmask
		}

		r.write405(w, allowedMask)
		return
	} else if ok := r.search(req.URL.Path, ctx, bitmask); ok >= 1 {
		foundPath = true

		if ok == 1 {
			entry := ctx.handler
			if entry.Validation {
				if !r.validatePathEntry(ctx, entry) {
					foundPath = false
				}
			}

			if foundPath {
				entry.Handler(w, req, ctx)
				return
			}
		} else if ok == 2 {
			allowedMask = ctx.allowedMask
		}
	}

	if foundPath {
		r.write405(w, allowedMask)
		return
	}

	if r.notFound != nil {
		r.notFound(w, req, ctx)
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(notFound)
	}
}

func (r *Router) isConsoleLoggingEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.Level(-10))
}

func (r *Router) newHTTPServer() *http.Server {
	return &http.Server{
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}
}

func (r *Router) startServer(listener net.Listener, addr string, servers *[]*http.Server, wg *sync.WaitGroup, errCh chan<- error) {
	server := r.newHTTPServer()
	*servers = append(*servers, server)

	wg.Add(1)

	go func() {
		defer wg.Done()

		err := server.Serve(listener)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}

		select {
		case errCh <- fmt.Errorf("%w: %q: %w", ErrServeFailed, addr, err):
		default:
		}
	}()
}

func validateListenAddress(listenAddr string) error {
	_, portStr, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrInvalidListenAddress, listenAddr, err)
	}

	if _, err = strconv.Atoi(portStr); err != nil {
		return fmt.Errorf("%w: %q: %w", ErrInvalidListenPort, listenAddr, err)
	}

	return nil
}

func reusePortListenConfig() net.ListenConfig {
	return net.ListenConfig{
		Control: func(network string, address string, c syscall.RawConn) error {
			var socketErr error

			if err := c.Control(func(fd uintptr) {
				if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
					socketErr = err
					return
				}

				if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
					socketErr = err
				}
			}); err != nil {
				return err
			}

			return socketErr
		},
	}
}

func (r *Router) shutdownServers(servers []*http.Server, consoleEnabled bool) {
	if len(servers) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(servers))

	for i := 0; i < len(servers); i++ {
		server := servers[i]

		go func() {
			defer wg.Done()

			if err := server.Shutdown(ctx); err != nil && consoleEnabled {
				slog.Error("Server shutdown error", "error", err)
			}
		}()
	}

	wg.Wait()
}

func (r *Router) MultiListenAndServe(listeners Listeners) error {
	consoleEnabled := r.isConsoleLoggingEnabled()

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}

	if consoleEnabled {
		slog.Info("Starting server")
		slog.Info("System resources", "workers", workers, "logical_cpus", runtime.NumCPU())
	}

	for i := 0; i < len(listeners); i++ {
		if err := validateListenAddress(listeners[i].Addr); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup

	servers := make([]*http.Server, 0, len(listeners)*workers)
	serveErrCh := make(chan error, len(listeners)*workers+1)

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	for i := 0; i < len(listeners); i++ {
		listenAddr := listeners[i].Addr

		useReusePort := runtime.GOOS != "windows"
		var reuseErr error

		if useReusePort {
			lc := reusePortListenConfig()

			firstListener, err := lc.Listen(context.Background(), "tcp", listenAddr)

			if err == nil {
				r.startServer(firstListener, listenAddr, &servers, &wg, serveErrCh)

				for worker := 1; worker < workers; worker++ {
					listener, err := lc.Listen(
						context.Background(),
						"tcp",
						listenAddr,
					)
					if err != nil {
						slog.Error("REUSEPORT listen failed", "worker", worker, "listen_addr", listenAddr, "err", err)
						continue
					}

					r.startServer(listener, listenAddr, &servers, &wg, serveErrCh)
				}
			} else {
				useReusePort = false
				reuseErr = err
			}
		}

		if !useReusePort {
			if reuseErr != nil {
				slog.Error("REUSEPORT unavailable; falling back to single listener", "listen_addr", listenAddr, "error", reuseErr)
			}

			listener, err := net.Listen("tcp", listenAddr)
			if err != nil {
				r.shutdownServers(servers, consoleEnabled)
				wg.Wait()

				return fmt.Errorf(
					"%w: %q: %w",
					ErrListenFailed,
					listenAddr,
					err,
				)
			}

			r.startServer(listener, listenAddr, &servers, &wg, serveErrCh)
		}

		if consoleEnabled {
			slog.Info("web server started", "server", serverName, "version", serverVersion, "listen_addr", listenAddr)
		}
	}

	select {
	case err := <-serveErrCh:
		r.shutdownServers(servers, consoleEnabled)
		wg.Wait()
		return err

	case <-signalCtx.Done():
	}

	r.SetReady(false)

	if consoleEnabled {
		slog.Info("Shutdown signal received. Shutting down servers...")
	}

	r.shutdownServers(servers, consoleEnabled)
	wg.Wait()

	if consoleEnabled {
		slog.Info("All servers shut down gracefully.")
	}

	return nil
}

func (r *Router) ListenAndServe(addr string) error {
	return r.MultiListenAndServe(Listeners{
		{Addr: addr},
	})
}
