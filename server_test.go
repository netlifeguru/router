package router

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func suppressTestLogs(t *testing.T) {
	t.Helper()

	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Cleanup(func() {
		slog.SetDefault(old)
	})
}

func TestHandleFuncAndServeHTTP(t *testing.T) {
	r := New()

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("world"))
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if string(body) != "world" {
		t.Fatalf("body = %q, want %q", string(body), "world")
	}
}

func TestServeHTTPDynamicRouteParams(t *testing.T) {
	r := New()

	r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte(ctx.Param("id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", w.Code, http.StatusOK, w.Body.String())
	}

	if w.Body.String() != "42" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "42")
	}
}

func TestServeHTTPGlobRouteParams(t *testing.T) {
	r := New()

	r.GET("/files/{wildcard:any}", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte(ctx.Param("wildcard")))
	})

	req := httptest.NewRequest(http.MethodGet, "/files/images/logo.png", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", w.Code, http.StatusOK, w.Body.String())
	}

	if w.Body.String() != "images/logo.png" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "images/logo.png")
	}
}

func TestServeHTTPPatternValidationPasses(t *testing.T) {
	r := New()

	r.GET("/users/{id:[0-9]+}", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte(ctx.Param("id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() != "123" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "123")
	}
}

func TestDynamicRoutePatternValidationFallsBackTo404(t *testing.T) {
	r := New()

	r.HandleFunc("/users/{id:[0-9]+}", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := New()

	r.HandleFunc("/resource", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	allow := w.Header().Get("Allow")
	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		if !strings.Contains(allow, method) {
			t.Fatalf("Allow header = %q, want it to contain %q", allow, method)
		}
	}

	if w.Body.String() != "405 method not allowed" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "405 method not allowed")
	}
}

func TestDynamicRouteMethodNotAllowed(t *testing.T) {
	r := New()

	r.GET("/users/{id}", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte(ctx.Param("id")))
	})

	req := httptest.NewRequest(http.MethodPost, "/users/42", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	allow := w.Header().Get("Allow")
	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		if !strings.Contains(allow, method) {
			t.Fatalf("Allow header = %q, want it to contain %q", allow, method)
		}
	}
}

func TestCustomNotFoundHandler(t *testing.T) {
	r := New()

	r.NotFound(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("custom 404"))
	})

	req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	if w.Body.String() != "custom 404" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "custom 404")
	}
}

func TestDefaultNotFoundHandler(t *testing.T) {
	r := New()

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	if body := w.Body.String(); body != "404 page not found" {
		t.Fatalf("body = %q, want %q", body, "404 page not found")
	}
}

func TestRecoveryFromPanic(t *testing.T) {
	suppressTestLogs(t)

	r := New()

	r.Recovery(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("Recovered"))
	})

	r.HandleFunc("/panic", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		panic("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
	if !strings.Contains(w.Body.String(), "Recovered") {
		t.Fatalf("body = %q, want it to contain %q", w.Body.String(), "Recovered")
	}
}

func TestRecoveryWithoutCustomHandlerReturns500(t *testing.T) {
	suppressTestLogs(t)

	r := New()

	r.HandleFunc("/panic", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Fatalf("body = %q, want it to contain %q", w.Body.String(), "Internal Server Error")
	}
}

func TestRecoveryHandlerPanicReturns500(t *testing.T) {
	suppressTestLogs(t)

	r := New()

	r.Recovery(func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		panic("recovery failed")
	})

	r.GET("/panic", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		panic("handler failed")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	if !strings.Contains(w.Body.String(), "Recovery middleware failed") {
		t.Fatalf("body = %q, want recovery failure message", w.Body.String())
	}
}

func TestWrite405(t *testing.T) {
	r := New()
	w := httptest.NewRecorder()

	r.write405(w, GET|POST)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	allow := w.Header().Get("Allow")
	for _, method := range []string{"GET", "HEAD", "POST", "OPTIONS"} {
		if !strings.Contains(allow, method) {
			t.Fatalf("Allow header = %q, want it to contain %q", allow, method)
		}
	}

	if w.Body.String() != "405 method not allowed" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "405 method not allowed")
	}
}

func TestValidatePathEntry(t *testing.T) {
	r := New()

	entry := &routeEntry{
		Patterns: []pattern{
			{Slug: "id", Type: isPattern, Fn: isDigits},
		},
	}

	ctx := &Context{
		segments: []seg{{Value: "123"}},
	}

	if !r.validatePathEntry(ctx, entry) {
		t.Fatal("validatePathEntry returned false, want true")
	}

	ctx.segments[0].Value = "abc"

	if r.validatePathEntry(ctx, entry) {
		t.Fatal("validatePathEntry returned true, want false")
	}
}

func TestRouterCanBeUsedAsHTTPHandler(t *testing.T) {
	r := New()

	r.GET("/handler", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("ok"))
	})

	var h http.Handler = r
	req := httptest.NewRequest(http.MethodGet, "/handler", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "ok")
	}
}

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

	r.shutdownServers(nil, false)
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
