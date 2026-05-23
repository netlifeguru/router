package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCooldownLimiterDefaults(t *testing.T) {
	lim := newCooldownLimiter(0, 0)

	if lim == nil {
		t.Fatal("expected newCooldownLimiter to return non-nil limiter")
	}
	if lim.last == nil {
		t.Fatal("expected lim.last map to be initialized")
	}
	if lim.ttl != 30*time.Second {
		t.Fatalf("expected default ttl 30s, got %v", lim.ttl)
	}
}

func TestCooldownLimiterAllowBasicFlow(t *testing.T) {
	lim := newCooldownLimiter(2*time.Second, 0)

	now := time.Now()
	key := "user1:/path"

	ok, retry := lim.allow(now, key, 500*time.Millisecond)
	if !ok {
		t.Fatalf("expected first request to be allowed")
	}
	if retry != 0 {
		t.Fatalf("expected retryAfter=0 for first request, got %v", retry)
	}

	ok2, retry2 := lim.allow(now, key, 500*time.Millisecond)
	if ok2 {
		t.Fatalf("expected second immediate request to be limited")
	}
	if retry2 <= 0 {
		t.Fatalf("expected positive retryAfter, got %v", retry2)
	}

	after := now.Add(500*time.Millisecond + time.Millisecond)
	ok3, retry3 := lim.allow(after, key, 500*time.Millisecond)
	if !ok3 {
		t.Fatalf("expected request after threshold to be allowed")
	}
	if retry3 != 0 {
		t.Fatalf("expected retryAfter=0 after threshold, got %v", retry3)
	}
}

func TestCooldownLimiterAllowNoThreshold(t *testing.T) {
	lim := newCooldownLimiter(2*time.Second, 0)

	now := time.Now()
	ok, retry := lim.allow(now, "any", 0)
	if !ok {
		t.Fatalf("expected allow= true when threshold <= 0")
	}
	if retry != 0 {
		t.Fatalf("expected retryAfter=0 when threshold <= 0, got %v", retry)
	}
}

func TestCooldownLimiterCleanupRemovesExpired(t *testing.T) {
	lim := newCooldownLimiter(1*time.Second, 0)

	now := time.Now()

	lim.last["old"] = now.Add(-2 * time.Second)
	lim.last["new"] = now
	lim.ttl = 1 * time.Second
	lim.cleanup()

	if _, ok := lim.last["old"]; ok {
		t.Fatalf("expected key 'old' to be removed by cleanup")
	}
	if _, ok := lim.last["new"]; !ok {
		t.Fatalf("expected key 'new' to remain after cleanup")
	}
}

func TestDefaultRateLimitKeyWithoutContext(t *testing.T) {
	if err := SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	req.RemoteAddr = "203.0.113.10:1234"

	key := defaultRateLimitKey(req, nil)

	expected := "GET|203.0.113.10|/test/path"
	if key != expected {
		t.Fatalf("defaultRateLimitKey = %q, want %q", key, expected)
	}
}

func TestDefaultRateLimitKeyWithContextRoute(t *testing.T) {
	if err := SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies error: %v", err)
	}
	
	req := httptest.NewRequest(http.MethodPost, "/ignored/path", nil)
	req.RemoteAddr = "198.51.100.5:9999"

	ctx := &Context{}
	ctx.handler.Route = "/api/v1/resource"

	key := defaultRateLimitKey(req, ctx)

	expected := "POST|198.51.100.5|/api/v1/resource"
	if key != expected {
		t.Fatalf("defaultRateLimitKey = %q, want %q", key, expected)
	}
}

func TestRateLimitMiddlewareBlocksFrequentRequests(t *testing.T) {
	threshold := 50 * time.Millisecond
	mw := RateLimit(threshold)

	handlerCalled := 0

	h := mw(func(w http.ResponseWriter, r *http.Request, c *Context) {
		handlerCalled++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/rate", nil)
	req.RemoteAddr = "203.0.113.1:4321"
	ctx := getContext()
	defer putContext(ctx)

	rr1 := httptest.NewRecorder()
	h(rr1, req, ctx)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	rr2 := httptest.NewRecorder()
	h(rr2, req, ctx)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rr2.Code)
	}

	retryAfter := rr2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatalf("expected Retry-After header to be set")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON body from default OnLimit, got error: %v", err)
	}
	if title, _ := payload["title"].(string); title != "too_many_requests" {
		t.Fatalf("expected title 'too_many_requests', got %q", payload["title"])
	}
	if status, _ := payload["status"].(float64); int(status) != http.StatusTooManyRequests {
		t.Fatalf("expected status %d in JSON, got %v", http.StatusTooManyRequests, payload["status"])
	}
	if detail, _ := payload["detail"].(string); !strings.Contains(detail, "Retry after") {
		t.Fatalf("expected detail to mention Retry after, got %q", detail)
	}

	time.Sleep(threshold + 10*time.Millisecond)

	rr3 := httptest.NewRecorder()
	h(rr3, req, ctx)
	if rr3.Code != http.StatusOK {
		t.Fatalf("third request after sleep: expected 200, got %d", rr3.Code)
	}

	if handlerCalled != 2 {
		t.Fatalf("expected handler to be called twice (1st a 3rd), got %d", handlerCalled)
	}
}

func TestRateLimitWithCustomOnLimit(t *testing.T) {
	threshold := 20 * time.Millisecond

	var onLimitCalled bool
	var seenRetryAfter time.Duration

	opt := func(cfg *RateLimitConfig) {
		cfg.OnLimit = func(w http.ResponseWriter, r *http.Request, c *Context, retryAfter time.Duration) {
			onLimitCalled = true
			seenRetryAfter = retryAfter
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("custom limit"))
		}
	}

	mw := RateLimit(threshold, opt)

	h := mw(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	req.RemoteAddr = "192.0.2.5:5555"
	ctx := getContext()
	defer putContext(ctx)

	rr1 := httptest.NewRecorder()
	h(rr1, req, ctx)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	rr2 := httptest.NewRecorder()
	h(rr2, req, ctx)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rr2.Code)
	}
	if !onLimitCalled {
		t.Fatalf("expected custom OnLimit to be called on rate limit")
	}
	if seenRetryAfter <= 0 {
		t.Fatalf("expected positive retryAfter in custom OnLimit, got %v", seenRetryAfter)
	}
	if body := rr2.Body.String(); body != "custom limit" {
		t.Fatalf("expected body 'custom limit', got %q", body)
	}
}

func TestMaxDur(t *testing.T) {
	a := 2 * time.Second
	b := 5 * time.Second

	if got := maxDur(a, b); got != b {
		t.Fatalf("maxDur(%v, %v) = %v, want %v", a, b, got, b)
	}
	if got := maxDur(b, a); got != b {
		t.Fatalf("maxDur(%v, %v) = %v, want %v", b, a, got, b)
	}
}

func TestMinDur(t *testing.T) {
	a := 2 * time.Second
	b := 5 * time.Second

	if got := minDur(a, b); got != a {
		t.Fatalf("minDur(%v, %v) = %v, want %v", a, b, got, a)
	}
	if got := minDur(b, a); got != a {
		t.Fatalf("minDur(%v, %v) = %v, want %v", b, a, got, a)
	}
}
