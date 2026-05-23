package router

import (
	"fmt"
	"net/http"
)

func (m *middlewareGroup) middlewareWrapper(h HandlerFunc) HandlerFunc {
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		h = m.middlewares[i](h)
	}
	return h
}

func (m *middlewareGroup) Group(prefix string) *RouteGroup {
	if prefix == "" || prefix == "/" {
		panic(fmt.Sprintf("router: invalid group prefix %q (cannot be empty or '/')", prefix))
	}

	m.r.validatePath(prefix)

	if prefix[0] != '/' {
		prefix = "/" + prefix
	}

	return &RouteGroup{
		r:      m.r,
		prefix: prefix,
	}
}

func (m *middlewareGroup) HandleFunc(url string, methods string, fn HandlerFunc) {
	m.r.HandleFunc(url, methods, m.middlewareWrapper(fn))
}

func (m *middlewareGroup) GET(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "GET", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) POST(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "POST", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) PUT(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "PUT", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) DELETE(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "DELETE", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) PATCH(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "PATCH", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) HEAD(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "HEAD", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) OPTIONS(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "OPTIONS", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) TRACE(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "TRACE", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) CONNECT(url string, fn HandlerFunc) {
	m.r.HandleFunc(url, "CONNECT", m.middlewareWrapper(fn))
}

func (m *middlewareGroup) Use(mi Middleware) {
	m.middlewares = append(m.middlewares, mi)
}

func (m *middlewareGroup) Mount(prefix string, handler http.Handler) {
	wrappedAdapter := m.middlewareWrapper(func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		n := len(ctx.params)

		for i := 0; i < n; i++ {
			req.SetPathValue(ctx.params[i].Key, ctx.params[i].Value)
		}

		handler.ServeHTTP(w, req)
	})

	m.r.HandleFunc(prefix, "ANY", wrappedAdapter)
}

func (m *middlewareGroup) MountFunc(prefix string, fn func(w http.ResponseWriter, req *http.Request)) {
	m.Mount(prefix, http.HandlerFunc(fn))
}
