package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterGroupRejectsInvalidPrefix(t *testing.T) {
	tests := []string{"", "/"}

	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			r := New()

			defer func() {
				if recover() == nil {
					t.Fatalf("Group(%q) did not panic", prefix)
				}
			}()

			r.Group(prefix)
		})
	}
}

func TestRouterGroupRejectsInvalidCharacters(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatalf("Group with invalid characters did not panic")
		}
	}()

	r.Group("/api/{id}")
}

func TestRouterGroupNormalizesPrefix(t *testing.T) {
	r := New()

	g := r.Group("api")

	if g == nil {
		t.Fatalf("Group returned nil")
	}

	if g.r != r {
		t.Fatalf("group router mismatch")
	}

	if g.prefix != "/api" {
		t.Fatalf("group prefix = %q, want %q", g.prefix, "/api")
	}
}

func TestRouteGroupGETRegistersRoute(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.GET("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Body.String(); got != "users" {
		t.Fatalf("body = %q, want %q", got, "users")
	}
}

func TestRouteGroupHandleFuncNormalizesChildPath(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.HandleFunc("users", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Body.String(); got != "ok" {
		t.Fatalf("body = %q, want %q", got, "ok")
	}
}

func TestRouteGroupRegistersHTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		add    func(*RouteGroup, string, HandlerFunc)
	}{
		{
			name:   "GET",
			method: http.MethodGet,
			add:    (*RouteGroup).GET,
		},
		{
			name:   "POST",
			method: http.MethodPost,
			add:    (*RouteGroup).POST,
		},
		{
			name:   "PUT",
			method: http.MethodPut,
			add:    (*RouteGroup).PUT,
		},
		{
			name:   "DELETE",
			method: http.MethodDelete,
			add:    (*RouteGroup).DELETE,
		},
		{
			name:   "PATCH",
			method: http.MethodPatch,
			add:    (*RouteGroup).PATCH,
		},
		{
			name:   "HEAD",
			method: http.MethodHead,
			add:    (*RouteGroup).HEAD,
		},
		{
			name:   "OPTIONS",
			method: http.MethodOptions,
			add:    (*RouteGroup).OPTIONS,
		},
		{
			name:   "TRACE",
			method: http.MethodTrace,
			add:    (*RouteGroup).TRACE,
		},
		{
			name:   "CONNECT",
			method: http.MethodConnect,
			add:    (*RouteGroup).CONNECT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			g := r.Group("/api")

			tt.add(g, "/resource", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
				w.WriteHeader(http.StatusCreated)
			})

			req := httptest.NewRequest(tt.method, "/api/resource", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusCreated, rec.Body.String())
			}
		})
	}
}

func TestRouteGroupReturns405ForWrongMethod(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.POST("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("created"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	if got := rec.Body.String(); got != "405 method not allowed" {
		t.Fatalf("body = %q, want %q", got, "405 method not allowed")
	}

	allow := rec.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodPost) {
		t.Fatalf("Allow header = %q, want it to contain %q", allow, http.MethodPost)
	}
}

func TestRouteGroupUseAppliesMiddlewareToGroupRoute(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Group-Middleware", "yes")
			ctx.Set("group", "middleware")
			next(w, req, ctx)
		}
	})

	g.GET("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		if got := ctx.Get("group"); got != "middleware" {
			t.Fatalf("ctx.Get(%q) = %#v, want %q", "group", got, "middleware")
		}

		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-Group-Middleware"); got != "yes" {
		t.Fatalf("X-Group-Middleware = %q, want %q", got, "yes")
	}

	if got := rec.Body.String(); got != "ok" {
		t.Fatalf("body = %q, want %q", got, "ok")
	}
}

func TestRouteGroupUseDoesNotApplyToOtherRoutes(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Group-Middleware", "yes")
			next(w, req, ctx)
		}
	})

	g.GET("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("group"))
	})

	r.GET("/plain", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("plain"))
	})

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-Group-Middleware"); got != "" {
		t.Fatalf("X-Group-Middleware = %q, want empty", got)
	}

	if got := rec.Body.String(); got != "plain" {
		t.Fatalf("body = %q, want %q", got, "plain")
	}
}

func TestRouteGroupMiddlewareOrder(t *testing.T) {
	r := New()

	var order []string

	g := r.Group("/api")
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			order = append(order, "first-before")
			next(w, req, ctx)
			order = append(order, "first-after")
		}
	})
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			order = append(order, "second-before")
			next(w, req, ctx)
			order = append(order, "second-after")
		}
	})

	g.GET("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	want := []string{
		"first-before",
		"second-before",
		"handler",
		"second-after",
		"first-after",
	}

	if len(order) != len(want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %#v, want %#v", order, want)
		}
	}
}

func TestRouteGroupMountFunc(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.MountFunc("/mounted", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("mounted"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/mounted", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Body.String(); got != "mounted" {
		t.Fatalf("body = %q, want %q", got, "mounted")
	}
}

func TestRouteGroupMountPassesPathValues(t *testing.T) {
	r := New()

	g := r.Group("/api")
	g.MountFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(req.PathValue("id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Body.String(); got != "42" {
		t.Fatalf("body = %q, want %q", got, "42")
	}
}
