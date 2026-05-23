package router

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
)

func TestListenAndServeInvalidAddressReturnsError(t *testing.T) {
	r := New()

	err := r.ListenAndServe("-1")
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrInvalidListenAddress) {
		t.Fatalf("error = %v, want ErrInvalidListenAddress", err)
	}
}

func TestMultiListenAndServeInvalidListenAddress(t *testing.T) {
	r := New()

	err := r.MultiListenAndServe(Listeners{
		{Addr: "invalid-address"},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrInvalidListenAddress) {
		t.Fatalf("error = %v, want ErrInvalidListenAddress", err)
	}
}

func TestMultiListenAndServeInvalidListenPort(t *testing.T) {
	r := New()

	err := r.MultiListenAndServe(Listeners{
		{Addr: "localhost:not-a-port"},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrInvalidListenPort) {
		t.Fatalf("error = %v, want ErrInvalidListenPort", err)
	}
}

func TestMultiListenAndServeListenFailed(t *testing.T) {
	r := New()

	err := r.MultiListenAndServe(Listeners{
		{Addr: "localhost:-1"},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrListenFailed) {
		t.Fatalf("error = %v, want ErrListenFailed", err)
	}
}

func TestShutdownServersWithNilSlice(t *testing.T) {
	r := New()

	r.shutdownServers(nil, nilMutex(), false)
}

func TestIsConsoleLoggingEnabled(t *testing.T) {
	r := New()

	old := slog.Default()
	defer slog.SetDefault(old)

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if r.isConsoleLoggingEnabled() {
		t.Fatal("console logging should be disabled for level -10 with info handler")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	_ = r.isConsoleLoggingEnabled()
}

func TestServerNameAndVersionAreDefined(t *testing.T) {
	if serverName == "" {
		t.Fatal("serverName is empty")
	}
	if serverVersion == "" {
		t.Fatal("serverVersion is empty")
	}
}

func TestRouterHandlerSatisfiesHTTPHandler(t *testing.T) {
	r := New()

	var _ http.Handler = r
}

func nilMutex() *sync.Mutex {
	return &sync.Mutex{}
}
