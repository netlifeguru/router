package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewInitializesRouter(t *testing.T) {
	r := New()

	if r == nil {
		t.Fatal("New returned nil")
	}
	if r.radixRoot == nil {
		t.Fatal("radixRoot is nil")
	}
	if r.staticRoutes == nil {
		t.Fatal("staticRoutes is nil")
	}
	if r.groupMiddlewares == nil {
		t.Fatal("groupMiddlewares is nil")
	}
	if r.middlewares == nil {
		t.Fatal("middlewares is nil")
	}
	if !r.IsReady() {
		t.Fatal("new router should be ready")
	}
}

func TestSetReadyAndIsReady(t *testing.T) {
	r := New()

	r.SetReady(false)
	if r.IsReady() {
		t.Fatal("IsReady() = true, want false")
	}

	r.SetReady(true)
	if !r.IsReady() {
		t.Fatal("IsReady() = false, want true")
	}
}

func TestHandleFuncRegistersStaticRoute(t *testing.T) {
	r := New()

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("world"))
	})

	entries := r.staticRoutes["/hello"]
	if len(entries) != 1 {
		t.Fatalf("len(staticRoutes[/hello]) = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Route != "/hello" {
		t.Fatalf("Route = %q, want %q", entry.Route, "/hello")
	}
	if entry.Bitmask != GET {
		t.Fatalf("Bitmask = %d, want %d", entry.Bitmask, GET)
	}
	if entry.Handler == nil {
		t.Fatal("Handler is nil")
	}
}

func TestHandleFuncPanicsOnInvalidPath(t *testing.T) {
	tests := []string{
		"",
		"missing-leading-slash",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			r := New()

			defer func() {
				if recover() == nil {
					t.Fatalf("HandleFunc(%q) did not panic", path)
				}
			}()

			r.HandleFunc(path, "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {})
		})
	}
}

func TestHandleFuncPanicsOnNilHandler(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatal("HandleFunc with nil handler did not panic")
		}
	}()

	r.HandleFunc("/nil", "GET", nil)
}

func TestHandleFuncPanicsOnInvalidMethod(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatal("HandleFunc with invalid method did not panic")
		}
	}()

	r.HandleFunc("/test", "INVALID", func(w http.ResponseWriter, req *http.Request, ctx *Context) {})
}

func TestHTTPMethodHelpersRegisterRoutes(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		bitmask int
		add     func(*Router, string, HandlerFunc)
	}{
		{name: "GET", method: http.MethodGet, bitmask: GET, add: (*Router).GET},
		{name: "POST", method: http.MethodPost, bitmask: POST, add: (*Router).POST},
		{name: "PUT", method: http.MethodPut, bitmask: PUT, add: (*Router).PUT},
		{name: "DELETE", method: http.MethodDelete, bitmask: DELETE, add: (*Router).DELETE},
		{name: "PATCH", method: http.MethodPatch, bitmask: PATCH, add: (*Router).PATCH},
		{name: "HEAD", method: http.MethodHead, bitmask: HEAD, add: (*Router).HEAD},
		{name: "OPTIONS", method: http.MethodOptions, bitmask: OPTIONS, add: (*Router).OPTIONS},
		{name: "CONNECT", method: http.MethodConnect, bitmask: CONNECT, add: (*Router).CONNECT},
		{name: "TRACE", method: http.MethodTrace, bitmask: TRACE, add: (*Router).TRACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			tt.add(r, "/resource", func(w http.ResponseWriter, req *http.Request, ctx *Context) {})

			entries := r.staticRoutes["/resource"]
			if len(entries) != 1 {
				t.Fatalf("len(staticRoutes[/resource]) = %d, want 1", len(entries))
			}

			if entries[0].Bitmask != tt.bitmask {
				t.Fatalf("Bitmask = %d, want %d", entries[0].Bitmask, tt.bitmask)
			}
		})
	}
}

func TestLivenessDefaultPath(t *testing.T) {
	r := New()

	r.Liveness("", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("alive"))
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "alive" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "alive")
	}
}

func TestReadinessDefaultPath(t *testing.T) {
	r := New()

	r.Readiness("", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ready"))
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ready" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "ready")
	}
}

func TestLivenessCustomPath(t *testing.T) {
	r := New()

	r.Liveness("/live", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("live"))
	})

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "live" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "live")
	}
}

func TestReadinessCustomPath(t *testing.T) {
	r := New()

	r.Readiness("/ready", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ready-custom"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ready-custom" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "ready-custom")
	}
}

func TestMountFunc(t *testing.T) {
	r := New()

	r.MountFunc("/mounted", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("mounted ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/mounted", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "mounted ok" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "mounted ok")
	}
}

func TestMountPanicsOnEmptyOrRootPrefix(t *testing.T) {
	tests := []string{"", "/"}

	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			r := New()

			defer func() {
				if recover() == nil {
					t.Fatalf("Mount(%q) did not panic", prefix)
				}
			}()

			r.Mount(prefix, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
		})
	}
}

func TestMountedHandlerReceivesPathParams(t *testing.T) {
	r := New()

	r.Mount("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(req.PathValue("id")))
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "42" {
		t.Fatalf("body = %q, want %q", w.Body.String(), "42")
	}
}

func TestMountedHandlerWorksForAnyMethod(t *testing.T) {
	r := New()

	r.MountFunc("/any", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(req.Method))
	})

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/any", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%q", w.Code, http.StatusOK, w.Body.String())
			}

			if method != http.MethodHead && w.Body.String() != method {
				t.Fatalf("body = %q, want %q", w.Body.String(), method)
			}
		})
	}
}

func TestStaticFileServing(t *testing.T) {
	root := t.TempDir()

	filePath := filepath.Join(root, "test.txt")
	expectedContent := "Hello from static"

	if err := os.WriteFile(filePath, []byte(expectedContent), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	r := New()
	r.Static("/assets", root)

	req := httptest.NewRequest(http.MethodGet, "/assets/test.txt", nil)
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

	if string(body) != expectedContent {
		t.Fatalf("body = %q, want %q", string(body), expectedContent)
	}
}

func TestStaticServesFaviconWhenPresent(t *testing.T) {
	root := t.TempDir()

	expected := "ico"
	if err := os.WriteFile(filepath.Join(root, "favicon.ico"), []byte(expected), 0o644); err != nil {
		t.Fatalf("failed to write favicon: %v", err)
	}

	r := New()
	r.Static("/assets", root)

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != expected {
		t.Fatalf("body = %q, want %q", w.Body.String(), expected)
	}
}

func TestRecoverySetter(t *testing.T) {
	r := New()

	fn := func(w http.ResponseWriter, req *http.Request, ctx *Context) {}

	r.Recovery(fn)

	if r.recovery == nil {
		t.Fatal("recovery handler was not set")
	}
}

func TestNotFoundSetter(t *testing.T) {
	r := New()

	fn := func(w http.ResponseWriter, req *http.Request, ctx *Context) {}

	r.NotFound(fn)

	if r.notFound == nil {
		t.Fatal("notFound handler was not set")
	}
}
