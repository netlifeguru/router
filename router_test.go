package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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

func TestRoutesSplitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty",
			path: "",
			want: nil,
		},
		{
			name: "root",
			path: "/",
			want: nil,
		},
		{
			name: "single",
			path: "/users",
			want: []string{"users"},
		},
		{
			name: "multiple",
			path: "/api/v1/users",
			want: []string{"api", "v1", "users"},
		},
		{
			name: "multiple slashes",
			path: "//api///v1/users//",
			want: []string{"api", "v1", "users"},
		},
		{
			name: "without leading slash",
			path: "api/v1/users",
			want: []string{"api", "v1", "users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPath(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitPath(%q) = %#v, want %#v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRoutesRemoveWrapper(t *testing.T) {
	r := New()

	tests := []struct {
		in    string
		start string
		end   string
		want  string
	}{
		{in: "(abc)", start: "(", end: ")", want: "abc"},
		{in: "[abc]", start: "[", end: "]", want: "abc"},
		{in: "abc", start: "(", end: ")", want: "abc"},
		{in: "(abc", start: "(", end: ")", want: "(abc"},
		{in: "abc)", start: "(", end: ")", want: "abc)"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := r.removeWrapper(tt.in, tt.start, tt.end); got != tt.want {
				t.Fatalf("removeWrapper(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRoutesFindPatterns(t *testing.T) {
	r := New()

	tests := []struct {
		name  string
		input string
		ok    string
		fail  string
	}{
		{
			name:  "function matcher",
			input: "isDigits",
			ok:    "123",
			fail:  "abc",
		},
		{
			name:  "pattern matcher",
			input: "[0-9]+",
			ok:    "123",
			fail:  "abc",
		},
		{
			name:  "wrapped pattern matcher",
			input: "([0-9]+)",
			ok:    "123",
			fail:  "abc",
		},
		{
			name:  "any",
			input: "any",
			ok:    "",
			fail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := r.findPatterns(tt.input)
			if fn == nil {
				t.Fatalf("findPatterns(%q) returned nil", tt.input)
			}

			if !fn(tt.ok) {
				t.Fatalf("findPatterns(%q)(%q) = false, want true", tt.input, tt.ok)
			}

			if tt.input != "any" && fn(tt.fail) {
				t.Fatalf("findPatterns(%q)(%q) = true, want false", tt.input, tt.fail)
			}
		})
	}
}

func TestRoutesFindPatternsUnknownReturnsNil(t *testing.T) {
	r := New()

	if fn := r.findPatterns("doesNotExist"); fn != nil {
		t.Fatalf("findPatterns returned non-nil for unknown matcher")
	}
}

func TestRoutesCountCaptureGroups(t *testing.T) {
	r := New()

	tests := []struct {
		in   string
		want int
	}{
		{in: "", want: 0},
		{in: "[0-9]+", want: 0},
		{in: "(foo)", want: 1},
		{in: "(foo)-(bar)", want: 2},
		{in: "(foo", want: -1},
		{in: "foo)", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := r.countCaptureGroups(tt.in); got != tt.want {
				t.Fatalf("countCaptureGroups(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestRoutesParseSlugStaticSegment(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "users", "/users")

	if !isStatic {
		t.Fatalf("isStatic = false, want true")
	}
	if reqValidation {
		t.Fatalf("reqValidation = true, want false")
	}
	if p.Slug != "users" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "users")
	}
	if p.Type != isString {
		t.Fatalf("Type = %d, want %d", p.Type, isString)
	}
}

func TestRoutesParseSlugWildcardSegment(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "*", "/assets/*")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if reqValidation {
		t.Fatalf("reqValidation = true, want false")
	}
	if p.Slug != "wildcard" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "wildcard")
	}
	if p.Type != isPattern {
		t.Fatalf("Type = %d, want %d", p.Type, isPattern)
	}
	if p.Fn == nil {
		t.Fatalf("Fn is nil")
	}
	if !p.Fn("anything/goes") {
		t.Fatalf("wildcard matcher returned false")
	}
}

func TestRoutesParseSlugAnyPattern(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "{wildcard:any}", "/files/{wildcard:any}")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if reqValidation {
		t.Fatalf("reqValidation = true, want false")
	}
	if p.Slug != "wildcard" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "wildcard")
	}
	if p.Type != isPattern {
		t.Fatalf("Type = %d, want %d", p.Type, isPattern)
	}
	if p.Fn == nil {
		t.Fatalf("Fn is nil")
	}
}

func TestRoutesParseSlugFunctionMatcher(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "{id:isDigits}", "/users/{id:isDigits}")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if !reqValidation {
		t.Fatalf("reqValidation = false, want true")
	}
	if p.Slug != "id" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "id")
	}
	if p.Type != isPattern {
		t.Fatalf("Type = %d, want %d", p.Type, isPattern)
	}
	if p.Fn == nil {
		t.Fatalf("Fn is nil")
	}
	if !p.Fn("123") {
		t.Fatalf("Fn(%q) = false, want true", "123")
	}
	if p.Fn("abc") {
		t.Fatalf("Fn(%q) = true, want false", "abc")
	}
}

func TestRoutesParseSlugRegexMatcher(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "{id:[0-9]+}", "/users/{id:[0-9]+}")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if !reqValidation {
		t.Fatalf("reqValidation = false, want true")
	}
	if p.Slug != "id" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "id")
	}
	if p.Type != isPattern {
		t.Fatalf("Type = %d, want %d", p.Type, isPattern)
	}
	if p.Fn == nil {
		t.Fatalf("Fn is nil")
	}
	if !p.Fn("123") {
		t.Fatalf("Fn(%q) = false, want true", "123")
	}
	if p.Fn("abc") {
		t.Fatalf("Fn(%q) = true, want false", "abc")
	}
}

func TestRoutesParseSlugRegexSubmatch(t *testing.T) {
	r := New()

	p, isStatic, reqValidation := r.parseSlug(true, false, "{slug:(foo|bar)}", "/items/{slug:(foo|bar)}")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if !reqValidation {
		t.Fatalf("reqValidation = false, want true")
	}
	if p.Slug != "slug" {
		t.Fatalf("Slug = %q, want %q", p.Slug, "slug")
	}
	if p.Type != isSubmatch {
		t.Fatalf("Type = %d, want %d", p.Type, isSubmatch)
	}
	if p.RegexCompiled == nil {
		t.Fatalf("RegexCompiled is nil")
	}
	if p.RegexCompiled.FindStringSubmatch("foo") == nil {
		t.Fatalf("RegexCompiled should match %q", "foo")
	}
	if p.RegexCompiled.FindStringSubmatch("baz") != nil {
		t.Fatalf("RegexCompiled should not match %q", "baz")
	}
}

func TestRoutesParseSlugEmptyPatternPanics(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatalf("parseSlug with empty pattern did not panic")
		}
	}()

	r.parseSlug(true, false, "{id:}", "/users/{id:}")
}

func TestRoutesParseSlugInvalidRegexPanics(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatalf("parseSlug with invalid regex did not panic")
		}
	}()

	r.parseSlug(true, false, "{id:[0-9+}", "/users/{id:[0-9+}")
}

func TestRoutesPreparePatternStatic(t *testing.T) {
	r := New()

	parts, patterns, isStatic, reqValidation, radixURL := r.preparePattern("/api/v1/users")

	if !isStatic {
		t.Fatalf("isStatic = false, want true")
	}
	if reqValidation {
		t.Fatalf("reqValidation = true, want false")
	}
	if radixURL != "/api/v1/users" {
		t.Fatalf("radixURL = %q, want %q", radixURL, "/api/v1/users")
	}
	if len(parts) != 0 {
		t.Fatalf("parts = %#v, want empty", parts)
	}
	if len(patterns) != 0 {
		t.Fatalf("patterns = %#v, want empty", patterns)
	}
}

func TestRoutesPreparePatternDynamic(t *testing.T) {
	r := New()

	parts, patterns, isStatic, reqValidation, radixURL := r.preparePattern("/users/{id:[0-9]+}")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if !reqValidation {
		t.Fatalf("reqValidation = false, want true")
	}
	if radixURL != "/users/*" {
		t.Fatalf("radixURL = %q, want %q", radixURL, "/users/*")
	}
	if !reflect.DeepEqual(parts, activeParts{"id"}) {
		t.Fatalf("parts = %#v, want %#v", parts, activeParts{"id"})
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want 1", len(patterns))
	}
	if patterns[0].Slug != "id" {
		t.Fatalf("patterns[0].Slug = %q, want %q", patterns[0].Slug, "id")
	}
}

func TestRoutesPreparePatternWildcardUsesGlob(t *testing.T) {
	r := New()

	parts, patterns, isStatic, reqValidation, radixURL := r.preparePattern("/files/*")

	if isStatic {
		t.Fatalf("isStatic = true, want false")
	}
	if reqValidation {
		t.Fatalf("reqValidation = true, want false")
	}
	if radixURL != "/files/**" {
		t.Fatalf("radixURL = %q, want %q", radixURL, "/files/**")
	}
	if !reflect.DeepEqual(parts, activeParts{"wildcard"}) {
		t.Fatalf("parts = %#v, want %#v", parts, activeParts{"wildcard"})
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want 1", len(patterns))
	}
	if patterns[0].Slug != "wildcard" {
		t.Fatalf("patterns[0].Slug = %q, want %q", patterns[0].Slug, "wildcard")
	}
}

func TestRoutesHandleFuncRegistersStaticRoute(t *testing.T) {
	r := New()

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte("hello"))
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
	if entry.Validation {
		t.Fatalf("Validation = true, want false")
	}
	if len(entry.Parts) != 0 {
		t.Fatalf("Parts = %#v, want empty", entry.Parts)
	}
	if len(entry.Patterns) != 0 {
		t.Fatalf("Patterns = %#v, want empty", entry.Patterns)
	}
	if entry.Handler == nil {
		t.Fatalf("Handler is nil")
	}
}

func TestRoutesHandleFuncRegistersDynamicRoute(t *testing.T) {
	r := New()

	r.HandleFunc("/users/{id:[0-9]+}", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		_, _ = w.Write([]byte(ctx.Param("id")))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "123" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "123")
	}

	req = httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	rec = httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d for invalid param", rec.Code, http.StatusNotFound)
	}
}

func TestRoutesHandleFuncPanicsOnInvalidPath(t *testing.T) {
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

func TestRoutesHandleFuncPanicsOnNilHandler(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatalf("HandleFunc with nil handler did not panic")
		}
	}()

	r.HandleFunc("/nil", "GET", nil)
}

func TestRoutesHandleFuncPanicsOnInvalidMethod(t *testing.T) {
	r := New()

	defer func() {
		if recover() == nil {
			t.Fatalf("HandleFunc with invalid method did not panic")
		}
	}()

	r.HandleFunc("/test", "INVALID", func(w http.ResponseWriter, req *http.Request, ctx *Context) {})
}

func TestRoutesHTTPMethodHelpersRegisterRoutes(t *testing.T) {
	tests := []struct {
		name    string
		bitmask int
		add     func(*Router, string, HandlerFunc)
	}{
		{name: "GET", bitmask: GET, add: (*Router).GET},
		{name: "POST", bitmask: POST, add: (*Router).POST},
		{name: "PUT", bitmask: PUT, add: (*Router).PUT},
		{name: "DELETE", bitmask: DELETE, add: (*Router).DELETE},
		{name: "PATCH", bitmask: PATCH, add: (*Router).PATCH},
		{name: "HEAD", bitmask: HEAD, add: (*Router).HEAD},
		{name: "OPTIONS", bitmask: OPTIONS, add: (*Router).OPTIONS},
		{name: "CONNECT", bitmask: CONNECT, add: (*Router).CONNECT},
		{name: "TRACE", bitmask: TRACE, add: (*Router).TRACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()

			tt.add(r, "/resource", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
				w.WriteHeader(http.StatusCreated)
			})

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

func TestRoutesMountRegistersAnyMethod(t *testing.T) {
	r := New()

	r.MountFunc("/mounted", func(w http.ResponseWriter, req *http.Request) {
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
			req := httptest.NewRequest(method, "/mounted", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
			}

			if method != http.MethodHead && rec.Body.String() != method {
				t.Fatalf("body = %q, want %q", rec.Body.String(), method)
			}
		})
	}
}

func TestRoutesMountPanicsOnEmptyOrRootPrefix(t *testing.T) {
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

func TestRoutesValidatePath(t *testing.T) {
	r := New()

	validPaths := []string{
		"/",
		"/users",
		"/api/v1/users",
		"/files/name.txt",
		"/dash-name",
		"/under_score",
		"/tilde~path",
	}

	for _, path := range validPaths {
		t.Run("valid "+path, func(t *testing.T) {
			r.validatePath(path)
		})
	}

	invalidPaths := []string{
		"",
		"/users/{id}",
		"/query?x=1",
		"/space here",
		"/hash#fragment",
	}

	for _, path := range invalidPaths {
		t.Run("invalid "+path, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("validatePath(%q) did not panic", path)
				}
			}()

			r.validatePath(path)
		})
	}
}
