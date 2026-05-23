package router

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetErrorMessage(t *testing.T) {
	r := New()

	baseErr := errors.New("base")

	tests := []struct {
		name string
		in   any
		want string
	}{
		{
			name: "error",
			in:   baseErr,
			want: "base",
		},
		{
			name: "string",
			in:   "boom",
			want: "panic occurred: boom",
		},
		{
			name: "other",
			in:   123,
			want: "panic occurred with unknown type: 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.getErrorMessage(tt.in)
			if err == nil {
				t.Fatal("getErrorMessage returned nil")
			}
			if err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
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

func TestHandlerReturnsServeHTTPHandler(t *testing.T) {
	r := New()

	r.GET("/handler", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("ok"))
	})

	h := r.handler()
	if h == nil {
		t.Fatal("handler returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/handler", nil)
	w := httptest.NewRecorder()

	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "ok")
	}
}
