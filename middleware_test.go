package router

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestContext() *Context {
	return &Context{}
}

func TestGetHeadRewritesMethodToGet(t *testing.T) {
	m := GetHead()

	var gotMethod string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		gotMethod = r.Method
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
}

func TestGetHeadLeavesGetUntouched(t *testing.T) {
	m := GetHead()

	var gotMethod string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		gotMethod = r.Method
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
}

func TestContentCharsetAllowed(t *testing.T) {
	m := ContentCharset("utf-8", "iso-8859-1")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestContentCharsetMissingHeaderPassesThrough(t *testing.T) {
	m := ContentCharset("utf-8")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
}

func TestContentCharsetMissingCharsetPassesThrough(t *testing.T) {
	m := ContentCharset("utf-8")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestContentCharsetInvalidContentTypePassesThrough(t *testing.T) {
	m := ContentCharset("utf-8")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "%%%")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestContentCharsetDisallowed(t *testing.T) {
	m := ContentCharset("utf-8")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=iso-8859-1")
	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should not be called")
	}
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
}

func TestAllowContentTypeAllowed(t *testing.T) {
	m := AllowContentType("application/json")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAllowContentTypeDisallowed(t *testing.T) {
	m := AllowContentType("application/json")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should not be called")
	}
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
}

func TestAllowContentTypeMissingHeaderPassesThrough(t *testing.T) {
	m := AllowContentType("application/json")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestCleanPathNormalizesURLPath(t *testing.T) {
	m := CleanPath()

	var gotPath string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		gotPath = r.URL.Path
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/foo//bar/../baz/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if gotPath != "/foo/baz" {
		t.Fatalf("path = %q, want %q", gotPath, "/foo/baz")
	}
}

func TestCleanPathLeavesValidPathUntouched(t *testing.T) {
	m := CleanPath()

	var gotPath string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		gotPath = r.URL.Path
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if gotPath != "/api/v1/resource" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/resource")
	}
}

func TestCompressGzipEnabledForAllowedType(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}

	if got := resp.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want %q", got, "Accept-Encoding")
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}

	if string(decompressed) != "hello world" {
		t.Fatalf("body = %q, want %q", string(decompressed), "hello world")
	}
}

func TestCompressDetectsContentTypeWhenMissing(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		_, _ = io.WriteString(w, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}
}

func TestCompressNotEnabledWithoutAcceptEncoding(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "no gzip")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}
	if string(body) != "no gzip" {
		t.Fatalf("body = %q, want %q", string(body), "no gzip")
	}
}

func TestCompressNotEnabledForHEAD(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ignored")
	})

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressNotEnabledForDisallowedType(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "application/json")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "plain text")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressNotEnabledForNoContent(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestDefaultCompressUsesGzipForText(t *testing.T) {
	m := DefaultCompress()

	body := []byte("hello, compressed world")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(body)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	ctx := newTestContext()

	h(rr, req, ctx)

	resp := rr.Result()
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("io.ReadAll error: %v", err)
	}

	if !bytes.Equal(decompressed, body) {
		t.Fatalf("body = %q, want %q", string(decompressed), string(body))
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty", in: "", want: false},
		{name: "gzip", in: "gzip", want: true},
		{name: "gzip with q", in: "gzip;q=1", want: true},
		{name: "gzip disabled", in: "gzip;q=0", want: false},
		{name: "gzip disabled decimal", in: "gzip;q=0.000", want: false},
		{name: "wildcard", in: "*", want: true},
		{name: "wildcard disabled", in: "*;q=0", want: false},
		{name: "br and gzip", in: "br, gzip", want: true},
		{name: "gzip disabled beats wildcard", in: "gzip;q=0, *;q=1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := acceptsGzip(tt.in); got != tt.want {
				t.Fatalf("acceptsGzip(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	m := RequestID()

	var seenReqID string
	var seenCtxID string

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		seenReqID = r.Header.Get("X-Request-ID")
		seenCtxID = RequestIDFromContext(r.Context())
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if seenReqID == "" {
		t.Fatalf("request X-Request-ID was not generated")
	}
	if seenCtxID != seenReqID {
		t.Fatalf("context request id = %q, want %q", seenCtxID, seenReqID)
	}
	if respID := rr.Header().Get("X-Request-ID"); respID != seenReqID {
		t.Fatalf("response X-Request-ID = %q, want %q", respID, seenReqID)
	}
}

func TestRequestIDUsesExistingHeader(t *testing.T) {
	const existing = "existing-id"

	m := RequestID()

	var seenReqID string
	var seenCtxID string

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		seenReqID = r.Header.Get("X-Request-ID")
		seenCtxID = RequestIDFromContext(r.Context())
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", existing)
	ctx := newTestContext()

	h(rr, req, ctx)

	if seenReqID != existing {
		t.Fatalf("request X-Request-ID = %q, want %q", seenReqID, existing)
	}
	if seenCtxID != existing {
		t.Fatalf("context request id = %q, want %q", seenCtxID, existing)
	}
	if respID := rr.Header().Get("X-Request-ID"); respID != existing {
		t.Fatalf("response X-Request-ID = %q, want %q", respID, existing)
	}
}

func TestRequestIDWithGenerator(t *testing.T) {
	const generated = "generated-id"

	m := RequestIDWithGenerator(func(r *http.Request) string {
		return generated
	})

	var seenReqID string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		seenReqID = r.Header.Get("X-Request-ID")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if seenReqID != generated {
		t.Fatalf("request X-Request-ID = %q, want %q", seenReqID, generated)
	}
	if respID := rr.Header().Get("X-Request-ID"); respID != generated {
		t.Fatalf("response X-Request-ID = %q, want %q", respID, generated)
	}
}

func TestRequestIDWithEmptyGeneratorFallsBack(t *testing.T) {
	m := RequestIDWithGenerator(func(r *http.Request) string {
		return "   "
	})

	var seenReqID string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		seenReqID = r.Header.Get("X-Request-ID")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if seenReqID == "" {
		t.Fatalf("request X-Request-ID was not generated")
	}
}

func TestClientIPUsesRemoteAddrWhenNoTrustedProxies(t *testing.T) {
	if err := SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	req.Header.Set("X-Real-IP", "203.0.113.1")

	ip := ClientIP(req)
	if ip != "10.0.0.1" {
		t.Fatalf("ClientIP = %q, want %q", ip, "10.0.0.1")
	}
}

func TestClientIPWithTrustedProxyAndXForwardedFor(t *testing.T) {
	if err := SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}
	defer func() {
		_ = SetTrustedProxies(nil)
	}()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")

	ip := ClientIP(req)
	if ip != "203.0.113.1" {
		t.Fatalf("ClientIP = %q, want %q", ip, "203.0.113.1")
	}
}

func TestClientIPWithTrustedProxyAndXRealIP(t *testing.T) {
	if err := SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}
	defer func() {
		_ = SetTrustedProxies(nil)
	}()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "203.0.113.2")

	ip := ClientIP(req)
	if ip != "203.0.113.2" {
		t.Fatalf("ClientIP = %q, want %q", ip, "203.0.113.2")
	}
}

func TestSetTrustedProxiesInvalidCIDR(t *testing.T) {
	err := SetTrustedProxies([]string{"not-a-cidr"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrInvalidTrustedProxyCIDR) {
		t.Fatalf("error = %v, want ErrInvalidTrustedProxyCIDR", err)
	}
}

func TestClientIPFromRemoteAddrFallback(t *testing.T) {
	if err := SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:9999"

	ip := ClientIP(req)
	if ip != "192.0.2.10" {
		t.Fatalf("ClientIP = %q, want %q", ip, "192.0.2.10")
	}
}

func TestRealIPMiddlewareSetsXRealIPHeader(t *testing.T) {
	m := RealIP()

	called := false
	var seenHeader string
	var seenRemote string

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
		seenHeader = r.Header.Get("X-Real-IP")
		seenRemote = r.RemoteAddr
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:8080"
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if seenHeader != "203.0.113.10" {
		t.Fatalf("X-Real-IP = %q, want %q", seenHeader, "203.0.113.10")
	}
	if seenRemote != "203.0.113.10:8080" {
		t.Fatalf("RemoteAddr = %q, want original RemoteAddr", seenRemote)
	}
}

func TestNoCacheMiddleware(t *testing.T) {
	m := NoCache()

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}

	hdr := rr.Header()
	if got := hdr.Get("Cache-Control"); got != "no-store, no-cache, must-revalidate, max-age=0" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := hdr.Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want %q", got, "no-cache")
	}
	if got := hdr.Get("Expires"); got != "0" {
		t.Fatalf("Expires = %q, want %q", got, "0")
	}
}

func TestCompileAndMatchOriginPatterns(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		patterns []string
		want     bool
	}{
		{name: "any", origin: "https://example.com", patterns: []string{"*"}, want: true},
		{name: "exact", origin: "https://example.com", patterns: []string{"https://example.com"}, want: true},
		{name: "exact case insensitive", origin: "https://EXAMPLE.com", patterns: []string{"https://example.com"}, want: true},
		{name: "prefix", origin: "https://api.example.com", patterns: []string{"https://api.*"}, want: true},
		{name: "not exact", origin: "https://sub.example.com", patterns: []string{"https://example.com"}, want: false},
		{name: "not allowed", origin: "https://notallowed.com", patterns: []string{"https://example.com"}, want: false},
		{name: "empty origin", origin: "", patterns: []string{"*"}, want: false},
		{name: "empty pattern ignored", origin: "https://example.com", patterns: []string{" "}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchCompiledOrigin(tt.origin, compileOriginPatterns(tt.patterns))
			if got != tt.want {
				t.Fatalf("matchCompiledOrigin(%q, %v) = %v, want %v", tt.origin, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestCORSAllowsOriginOnSimpleRequest(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
		ExposedHeaders: []string{"X-Total-Count"},
	}

	m := CORS(opts)

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}

	hdr := rr.Header()
	if got := hdr.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if got := hdr.Get("Access-Control-Expose-Headers"); got != "X-Total-Count" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want %q", got, "X-Total-Count")
	}
	if got := hdr.Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want %q", got, "Origin")
	}
}

func TestCORSDisallowedOriginPassesThroughWithoutHeaders(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	}

	m := CORS(opts)

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler was not called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           600,
	}

	m := CORS(opts)

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should not be called for preflight")
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	hdr := rr.Header()
	if got := hdr.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if got := hdr.Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", got, "GET, POST")
	}
	if got := hdr.Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %q", got, "Authorization, Content-Type")
	}
	if got := hdr.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	if got := hdr.Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("Access-Control-Max-Age = %q, want %q", got, "600")
	}
}

func TestCORSOptionsWithoutRequestMethodReturnsNoContent(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	}

	m := CORS(opts)

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should not be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestAppendVaryDoesNotDuplicate(t *testing.T) {
	h := http.Header{}
	h.Add("Vary", "Origin")

	appendVary(h, "Origin")
	appendVary(h, "Accept-Encoding")
	appendVary(h, "accept-encoding")

	got := h.Values("Vary")
	if len(got) != 2 {
		t.Fatalf("Vary values = %#v, want 2 values", got)
	}
	if got[0] != "Origin" {
		t.Fatalf("first Vary = %q, want %q", got[0], "Origin")
	}
	if got[1] != "Accept-Encoding" {
		t.Fatalf("second Vary = %q, want %q", got[1], "Accept-Encoding")
	}
}

func TestUseDefaultsAppliesMiddlewares(t *testing.T) {
	r := New()
	r.UseDefaults()

	var seenMethod string
	var seenRealIP string
	var seenRemote string
	var seenRequestID string

	r.HandleFunc("/test", "GET", func(w http.ResponseWriter, req *http.Request, c *Context) {
		seenMethod = req.Method
		seenRealIP = req.Header.Get("X-Real-IP")
		seenRemote = req.RemoteAddr
		seenRequestID = RequestIDFromContext(req.Context())
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "203.0.113.5:5555"

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}

	if seenMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", seenMethod, http.MethodGet)
	}

	if seenRealIP != "203.0.113.5" {
		t.Fatalf("X-Real-IP = %q, want %q", seenRealIP, "203.0.113.5")
	}

	if seenRemote != "203.0.113.5:5555" {
		t.Fatalf("RemoteAddr = %q, want original RemoteAddr", seenRemote)
	}

	if seenRequestID == "" {
		t.Fatalf("request id was not stored in request context")
	}

	if got := rr.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("X-Request-ID response header was not set")
	}

	if rr.Header().Get("Cache-Control") == "" ||
		rr.Header().Get("Pragma") == "" ||
		rr.Header().Get("Expires") == "" {
		t.Fatalf("cache headers were not set")
	}
}
