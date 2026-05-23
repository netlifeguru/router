package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouterWithReturnsMiddlewareGroup(t *testing.T) {
	r := New()

	mw := func(next HandlerFunc) HandlerFunc {
		return next
	}

	g := r.With(mw)

	if g == nil {
		t.Fatalf("With returned nil")
	}
	if g.r != r {
		t.Fatalf("middleware group router mismatch")
	}
	if len(g.middlewares) != 1 {
		t.Fatalf("len(middlewares) = %d, want 1", len(g.middlewares))
	}
}

func TestMiddlewareGroupMiddlewareWrapperOrder(t *testing.T) {
	r := New()

	var order []string

	g := r.With(
		func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
				order = append(order, "first-before")
				next(w, req, ctx)
				order = append(order, "first-after")
			}
		},
		func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
				order = append(order, "second-before")
				next(w, req, ctx)
				order = append(order, "second-after")
			}
		},
	)

	h := g.middlewareWrapper(func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{}

	h(rec, req, ctx)

	want := []string{
		"first-before",
		"second-before",
		"handler",
		"second-after",
		"first-after",
	}

	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}
}

func TestMiddlewareGroupUseAppendsMiddleware(t *testing.T) {
	r := New()

	g := r.With()

	g.Use(func(next HandlerFunc) HandlerFunc {
		return next
	})
	g.Use(func(next HandlerFunc) HandlerFunc {
		return next
	})

	if len(g.middlewares) != 2 {
		t.Fatalf("len(middlewares) = %d, want 2", len(g.middlewares))
	}
}

func TestMiddlewareGroupGETRegistersWrappedRoute(t *testing.T) {
	r := New()

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Middleware-Group", "yes")
			ctx.Set("middleware_group", "ok")
			next(w, req, ctx)
		}
	})

	g.GET("/users", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		if got := ctx.Get("middleware_group"); got != "ok" {
			t.Fatalf("ctx.Get(%q) = %#v, want %q", "middleware_group", got, "ok")
		}

		_, _ = w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-Middleware-Group"); got != "yes" {
		t.Fatalf("X-Middleware-Group = %q, want %q", got, "yes")
	}

	if got := rec.Body.String(); got != "users" {
		t.Fatalf("body = %q, want %q", got, "users")
	}
}

func TestMiddlewareGroupHandleFuncRegistersWrappedRoute(t *testing.T) {
	r := New()

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-From-With", "true")
			next(w, req, ctx)
		}
	})

	g.HandleFunc("/custom", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("custom"))
	})

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-From-With"); got != "true" {
		t.Fatalf("X-From-With = %q, want %q", got, "true")
	}

	if got := rec.Body.String(); got != "custom" {
		t.Fatalf("body = %q, want %q", got, "custom")
	}
}

func TestMiddlewareGroupRegistersHTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		add    func(*middlewareGroup, string, HandlerFunc)
	}{
		{
			name:   "GET",
			method: http.MethodGet,
			add:    (*middlewareGroup).GET,
		},
		{
			name:   "POST",
			method: http.MethodPost,
			add:    (*middlewareGroup).POST,
		},
		{
			name:   "PUT",
			method: http.MethodPut,
			add:    (*middlewareGroup).PUT,
		},
		{
			name:   "DELETE",
			method: http.MethodDelete,
			add:    (*middlewareGroup).DELETE,
		},
		{
			name:   "PATCH",
			method: http.MethodPatch,
			add:    (*middlewareGroup).PATCH,
		},
		{
			name:   "HEAD",
			method: http.MethodHead,
			add:    (*middlewareGroup).HEAD,
		},
		{
			name:   "OPTIONS",
			method: http.MethodOptions,
			add:    (*middlewareGroup).OPTIONS,
		},
		{
			name:   "TRACE",
			method: http.MethodTrace,
			add:    (*middlewareGroup).TRACE,
		},
		{
			name:   "CONNECT",
			method: http.MethodConnect,
			add:    (*middlewareGroup).CONNECT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()

			g := r.With(func(next HandlerFunc) HandlerFunc {
				return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
					w.Header().Set("X-Method-Middleware", tt.method)
					next(w, req, ctx)
				}
			})

			tt.add(g, "/resource", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
				w.WriteHeader(http.StatusCreated)
			})

			req := httptest.NewRequest(tt.method, "/resource", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusCreated, rec.Body.String())
			}

			if got := rec.Header().Get("X-Method-Middleware"); got != tt.method {
				t.Fatalf("X-Method-Middleware = %q, want %q", got, tt.method)
			}
		})
	}
}

func TestMiddlewareGroupDoesNotAffectPlainRoutes(t *testing.T) {
	r := New()

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Middleware-Group", "yes")
			next(w, req, ctx)
		}
	})

	g.GET("/grouped", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("grouped"))
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

	if got := rec.Header().Get("X-Middleware-Group"); got != "" {
		t.Fatalf("X-Middleware-Group = %q, want empty", got)
	}

	if got := rec.Body.String(); got != "plain" {
		t.Fatalf("body = %q, want %q", got, "plain")
	}
}

func TestMiddlewareGroupWithGlobalMiddlewareOrder(t *testing.T) {
	r := New()

	var order []string

	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			order = append(order, "global-before")
			next(w, req, ctx)
			order = append(order, "global-after")
		}
	})

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			order = append(order, "with-before")
			next(w, req, ctx)
			order = append(order, "with-after")
		}
	})

	g.GET("/test", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	want := []string{
		"global-before",
		"with-before",
		"handler",
		"with-after",
		"global-after",
	}

	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}
}

func TestMiddlewareGroupMountFunc(t *testing.T) {
	r := New()

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Mounted-Middleware", "yes")
			next(w, req, ctx)
		}
	})

	g.MountFunc("/mounted", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("mounted"))
	})

	req := httptest.NewRequest(http.MethodGet, "/mounted", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-Mounted-Middleware"); got != "yes" {
		t.Fatalf("X-Mounted-Middleware = %q, want %q", got, "yes")
	}

	if got := rec.Body.String(); got != "mounted" {
		t.Fatalf("body = %q, want %q", got, "mounted")
	}
}

func TestMiddlewareGroupMountPassesPathValues(t *testing.T) {
	r := New()

	g := r.With(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			w.Header().Set("X-Middleware", "yes")
			next(w, req, ctx)
		}
	})

	g.MountFunc("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(req.PathValue("id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get("X-Middleware"); got != "yes" {
		t.Fatalf("X-Middleware = %q, want %q", got, "yes")
	}

	if got := rec.Body.String(); got != "42" {
		t.Fatalf("body = %q, want %q", got, "42")
	}
}

func TestMiddlewareGroupGroupRejectsInvalidPrefix(t *testing.T) {
	tests := []string{"", "/"}

	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			r := New()
			g := r.With()

			defer func() {
				if recover() == nil {
					t.Fatalf("Group(%q) did not panic", prefix)
				}
			}()

			g.Group(prefix)
		})
	}
}

func TestMiddlewareGroupGroupNormalizesPrefix(t *testing.T) {
	r := New()

	mg := r.With()
	g := mg.Group("api")

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
