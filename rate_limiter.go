package router

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimitOption func(*RateLimitConfig)

type cooldownLimiter struct {
	mu   sync.RWMutex
	last map[string]time.Time
	ttl  time.Duration
}

type RateLimitConfig struct {
	KeyFn           func(r *http.Request, c *Context) string
	OnLimit         func(w http.ResponseWriter, r *http.Request, c *Context, retryAfter time.Duration)
	TTL             time.Duration
	CleanupInterval time.Duration
}

func newCooldownLimiter(ttl, cleanupInterval time.Duration) *cooldownLimiter {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Second
	}
	lim := &cooldownLimiter{
		last: make(map[string]time.Time, 1024),
		ttl:  ttl,
	}

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		for range ticker.C {
			lim.cleanup()
		}
	}()

	return lim
}

func (l *cooldownLimiter) cleanup() {
	now := time.Now()
	cutoff := now.Add(-l.ttl)

	l.mu.Lock()
	defer l.mu.Unlock()
	for k, t := range l.last {
		if t.Before(cutoff) {
			delete(l.last, k)
		}
	}
}

func (l *cooldownLimiter) allow(now time.Time, key string, threshold time.Duration) (bool, time.Duration) {
	if threshold <= 0 {
		return true, 0
	}

	l.mu.RLock()
	t, ok := l.last[key]
	l.mu.RUnlock()

	if ok {
		next := t.Add(threshold)
		if now.Before(next) {
			return false, next.Sub(now)
		}
	}

	l.mu.Lock()
	l.last[key] = now
	l.mu.Unlock()

	return true, 0
}

func RateLimit(threshold time.Duration, opts ...RateLimitOption) Middleware {
	cfg := RateLimitConfig{
		KeyFn: defaultRateLimitKey,
		OnLimit: func(w http.ResponseWriter, r *http.Request, c *Context, retryAfter time.Duration) {
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)

			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"title":  "too_many_requests",
				"status": http.StatusTooManyRequests,
				"detail": "Please slow down. Retry after " + strconv.Itoa(secs) + "s.",
			})
		},
		TTL:             maxDur(2*threshold, 5*time.Second),
		CleanupInterval: minDur(maxDur(threshold, 1*time.Second), 30*time.Second),
	}

	for _, o := range opts {
		o(&cfg)
	}

	lim := newCooldownLimiter(cfg.TTL, cfg.CleanupInterval)

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			key := cfg.KeyFn(r, c)

			ok, retryAfter := lim.allow(time.Now(), key, threshold)
			if !ok {
				cfg.OnLimit(w, r, c, retryAfter)
				return
			}

			next(w, r, c)
		}
	}
}

func defaultRateLimitKey(r *http.Request, c *Context) string {
	ip := ClientIP(r)

	route := r.URL.Path
	if c != nil && c.handler.Route != "" {
		route = c.handler.Route
	}

	var b strings.Builder
	b.Grow(len(r.Method) + 1 + len(ip) + 1 + len(route))
	b.WriteString(r.Method)
	b.WriteByte('|')
	b.WriteString(ip)
	b.WriteByte('|')
	b.WriteString(route)
	return b.String()
}

func maxDur(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
