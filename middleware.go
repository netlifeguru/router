package router

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Middleware func(HandlerFunc) HandlerFunc

type contextKey string

const requestIDContextKey contextKey = "request_id"

var trustedCIDRs atomic.Value

func (r *Router) insertGroupMiddleware(group string, url string) {
	r.groupMiddlewares[url] = groupMiddleware{
		Route: url,
		Group: group,
	}
}

func (r *Router) Use(m Middleware) {
	r.useGroup(m, "")
}

func (r *Router) useGroup(m Middleware, n string) {
	r.middlewares[n] = append(r.middlewares[n], m)
}

func (r *Router) wrap(route string, h HandlerFunc) HandlerFunc {
	if gm, ok := r.groupMiddlewares[route]; ok && gm.Group != "" {
		if mws, ok := r.middlewares[gm.Group]; ok {
			for i := len(mws) - 1; i >= 0; i-- {
				h = mws[i](h)
			}
		}
	}

	if mws, ok := r.middlewares[""]; ok {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
	}

	return h
}

func contentTypeBase(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

func AllowContentType(types ...string) Middleware {
	allowed := make([]string, 0, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t != "" {
			allowed = append(allowed, t)
		}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			ct := r.Header.Get("Content-Type")
			if ct == "" || len(allowed) == 0 {
				next(w, r, ctx)
				return
			}

			ct = contentTypeBase(ct)
			for _, typ := range allowed {
				if strings.EqualFold(ct, typ) {
					next(w, r, ctx)
					return
				}
			}

			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		}
	}
}

// CleanPath normalizes r.URL.Path after the route has already matched.
// If you need cleaning before route matching, do it in Router.ServeHTTP before lookup.
func CleanPath() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			normalized := path.Clean(r.URL.Path)
			if normalized != r.URL.Path {
				r.URL.Path = normalized
			}
			next(w, r, ctx)
		}
	}
}

func GetHead() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			if r.Method == http.MethodHead {
				r.Method = http.MethodGet
			}
			next(w, r, c)
		}
	}
}

func ContentCharset(charsets ...string) Middleware {
	allowed := make([]string, 0, len(charsets))
	for _, ch := range charsets {
		ch = strings.TrimSpace(ch)
		if ch != "" {
			allowed = append(allowed, ch)
		}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			ct := r.Header.Get("Content-Type")
			if ct == "" || len(allowed) == 0 {
				next(w, r, ctx)
				return
			}

			_, params, err := mime.ParseMediaType(ct)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				return
			}

			cs := strings.TrimSpace(params["charset"])
			if cs == "" {
				next(w, r, ctx)
				return
			}

			for _, charset := range allowed {
				if strings.EqualFold(cs, charset) {
					next(w, r, ctx)
					return
				}
			}

			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		}
	}
}

type compressResponseWriter struct {
	http.ResponseWriter
	types    []string
	level    int
	gz       *gzip.Writer
	status   int
	wroteHdr bool
}

func (cw *compressResponseWriter) Header() http.Header {
	return cw.ResponseWriter.Header()
}

func (cw *compressResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

func (cw *compressResponseWriter) WriteHeader(status int) {
	if cw.wroteHdr {
		return
	}
	cw.status = status
}

func (cw *compressResponseWriter) commitHeader() {
	if cw.wroteHdr {
		return
	}
	if cw.status == 0 {
		cw.status = http.StatusOK
	}
	cw.wroteHdr = true
	cw.ResponseWriter.WriteHeader(cw.status)
}

func (cw *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := cw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, ErrHijackerNotSupported
}

func (cw *compressResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := cw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (cw *compressResponseWriter) enableGzip() error {
	if cw.gz != nil {
		return nil
	}

	appendVary(cw.Header(), "Accept-Encoding")
	cw.Header().Del("Content-Length")
	cw.Header().Set("Content-Encoding", "gzip")

	gz, err := gzip.NewWriterLevel(cw.ResponseWriter, cw.level)
	if err != nil {
		return err
	}

	cw.gz = gz
	return nil
}

func (cw *compressResponseWriter) shouldCompress(b []byte) bool {
	if cw.status < 200 || cw.status >= 300 {
		return false
	}
	if cw.status == http.StatusNoContent || cw.status == http.StatusResetContent {
		return false
	}

	ct := cw.Header().Get("Content-Type")
	if ct == "" && len(b) > 0 {
		ct = http.DetectContentType(b)
		cw.Header().Set("Content-Type", ct)
	}

	ct = contentTypeBase(ct)
	for _, allowed := range cw.types {
		if strings.EqualFold(ct, allowed) {
			return true
		}
	}

	return false
}

func (cw *compressResponseWriter) Write(b []byte) (int, error) {
	if !cw.wroteHdr {
		if cw.status == 0 {
			cw.status = http.StatusOK
		}

		if cw.shouldCompress(b) {
			if err := cw.enableGzip(); err != nil {
				return 0, err
			}
		}

		cw.commitHeader()
	}

	if cw.gz != nil {
		return cw.gz.Write(b)
	}

	return cw.ResponseWriter.Write(b)
}

func (cw *compressResponseWriter) Flush() {
	if !cw.wroteHdr {
		cw.commitHeader()
	}
	if cw.gz != nil {
		_ = cw.gz.Flush()
	}
	if fl, ok := cw.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func Compress(level int, types ...string) Middleware {
	allowed := make([]string, 0, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t != "" {
			allowed = append(allowed, t)
		}
	}

	if level < gzip.HuffmanOnly {
		level = gzip.DefaultCompression
	}
	if level > gzip.BestCompression {
		level = gzip.BestCompression
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			if r.Method == http.MethodHead || len(allowed) == 0 || !acceptsGzip(r.Header.Get("Accept-Encoding")) {
				next(w, r, c)
				return
			}

			cw := &compressResponseWriter{
				ResponseWriter: w,
				types:          allowed,
				level:          level,
			}

			defer func() {
				if !cw.wroteHdr {
					cw.commitHeader()
				}
				if cw.gz != nil {
					_ = cw.gz.Close()
				}
			}()

			next(cw, r, c)
		}
	}
}

func acceptsGzip(s string) bool {
	gzipAllowed := false
	starAllowed := false
	gzipExplicitlyDisabled := false

	for len(s) > 0 {
		part := s
		if i := strings.IndexByte(s, ','); i >= 0 {
			part = s[:i]
			s = s[i+1:]
		} else {
			s = ""
		}

		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		token := part
		q := ""
		if i := strings.IndexByte(part, ';'); i >= 0 {
			token = strings.TrimSpace(part[:i])
			params := part[i+1:]

			for len(params) > 0 {
				p := params
				if j := strings.IndexByte(params, ';'); j >= 0 {
					p = params[:j]
					params = params[j+1:]
				} else {
					params = ""
				}

				p = strings.TrimSpace(p)
				if len(p) >= 2 && (p[0] == 'q' || p[0] == 'Q') {
					if eq := strings.IndexByte(p, '='); eq >= 0 {
						q = strings.TrimSpace(p[eq+1:])
					}
				}
			}
		}

		if strings.EqualFold(token, "gzip") {
			if isZeroQ(q) {
				gzipExplicitlyDisabled = true
				gzipAllowed = false
			} else {
				gzipAllowed = true
			}
			continue
		}

		if token == "*" && !isZeroQ(q) {
			starAllowed = true
		}
	}

	if gzipExplicitlyDisabled {
		return false
	}

	return gzipAllowed || starAllowed
}

func isZeroQ(q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	if q == "0" {
		return true
	}
	if !strings.HasPrefix(q, "0.") {
		return false
	}
	if len(q) == 2 {
		return false
	}
	for _, c := range q[2:] {
		if c != '0' {
			return false
		}
	}
	return true
}

func SetTrustedProxies(cidrs []string) error {
	out := make([]*net.IPNet, 0, len(cidrs))

	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		_, ipn, err := net.ParseCIDR(c)
		if err != nil {
			return fmt.Errorf("%w: %q: %w", ErrInvalidTrustedProxyCIDR, c, err)
		}

		out = append(out, ipn)
	}

	trustedCIDRs.Store(out)
	return nil
}

func isTrustedRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	v := trustedCIDRs.Load()
	if v == nil {
		return false
	}

	nets := v.([]*net.IPNet)
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func ClientIP(r *http.Request) string {
	if isTrustedRemote(r.RemoteAddr) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i >= 0 {
				xff = xff[:i]
			}
			xff = strings.TrimSpace(xff)
			if net.ParseIP(xff) != nil {
				return xff
			}
		}

		if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
			if net.ParseIP(xrip) != nil {
				return xrip
			}
		}
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type originPattern struct {
	exact  string
	prefix string
	any    bool
}

func compileOriginPatterns(patterns []string) []originPattern {
	out := make([]originPattern, 0, len(patterns))

	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}

		if p == "*" {
			out = append(out, originPattern{any: true})
			continue
		}

		if strings.HasSuffix(p, "*") {
			out = append(out, originPattern{prefix: strings.TrimSuffix(p, "*")})
			continue
		}

		out = append(out, originPattern{exact: p})
	}

	return out
}

func matchCompiledOrigin(origin string, patterns []originPattern) bool {
	if origin == "" {
		return false
	}

	origin = strings.ToLower(strings.TrimSpace(origin))

	for _, p := range patterns {
		if p.any {
			return true
		}

		if p.prefix != "" {
			if strings.HasPrefix(origin, p.prefix) {
				return true
			}
			continue
		}

		if origin == p.exact {
			return true
		}
	}

	return false
}

func CORS(opts CORSOptions) Middleware {
	origins := compileOriginPatterns(opts.AllowedOrigins)

	allowedMethods := strings.Join(opts.AllowedMethods, ", ")
	allowedHeaders := strings.Join(opts.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(opts.ExposedHeaders, ", ")

	maxAge := ""
	if opts.MaxAge > 0 {
		maxAge = strconv.Itoa(opts.MaxAge)
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			origin := r.Header.Get("Origin")
			if !matchCompiledOrigin(origin, origins) {
				next(w, r, c)
				return
			}

			h := w.Header()
			appendVary(h, "Origin")
			h.Set("Access-Control-Allow-Origin", origin)

			if opts.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				appendVary(h, "Access-Control-Request-Method")
				appendVary(h, "Access-Control-Request-Headers")

				if r.Header.Get("Access-Control-Request-Method") == "" {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				if allowedMethods != "" {
					h.Set("Access-Control-Allow-Methods", allowedMethods)
				}
				if allowedHeaders != "" {
					h.Set("Access-Control-Allow-Headers", allowedHeaders)
				}
				if maxAge != "" {
					h.Set("Access-Control-Max-Age", maxAge)
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			if exposedHeaders != "" {
				h.Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			next(w, r, c)
		}
	}
}

func appendVary(h http.Header, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	for _, line := range h.Values("Vary") {
		for len(line) > 0 {
			part := line
			if i := strings.IndexByte(line, ','); i >= 0 {
				part = line[:i]
				line = line[i+1:]
			} else {
				line = ""
			}

			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}

	h.Add("Vary", value)
}

var reqIDCounter uint64

func nextRequestID() string {
	id := atomic.AddUint64(&reqIDCounter, 1)
	return strconv.FormatUint(id, 10)
}

func RequestID() Middleware {
	return RequestIDWithGenerator(nil)
}

func RequestIDWithGenerator(generate func(*http.Request) string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if id == "" {
				if generate != nil {
					id = strings.TrimSpace(generate(r))
				}
				if id == "" {
					id = nextRequestID()
				}
			}

			r.Header.Set("X-Request-ID", id)
			w.Header().Set("X-Request-ID", id)

			ctx := context.WithValue(r.Context(), requestIDContextKey, id)
			r = r.WithContext(ctx)

			next(w, r, c)
		}
	}
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey).(string)
	return id
}

func RealIP() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			ip := ClientIP(r)
			if ip != "" {
				r.Header.Set("X-Real-IP", ip)
			}

			next(w, r, c)
		}
	}
}

func NoCache() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			h := w.Header()
			h.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			h.Set("Pragma", "no-cache")
			h.Set("Expires", "0")

			next(w, r, c)
		}
	}
}

func DefaultCompress() Middleware {
	return Compress(
		gzip.DefaultCompression,
		"text/html",
		"text/plain",
		"text/css",
		"application/javascript",
		"text/javascript",
		"application/json",
		"application/xml",
		"text/xml",
	)
}

func Logger() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			start := time.Now()
			reqID := RequestIDFromContext(r.Context())

			next(w, r, ctx)
			
			slog.Log(
				r.Context(),
				slog.LevelInfo,
				"request processed",
				"request_id", reqID,
				"method", r.Method,
				"host", r.Host,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"duration", time.Since(start),
			)
		}
	}
}

func (r *Router) UseDefaults() {
	r.Use(GetHead())
	r.Use(RequestID())
	r.Use(RealIP())
	r.Use(NoCache())
}
