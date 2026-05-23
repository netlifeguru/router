package router

import (
	"fmt"
	"net/http"
	"strings"
)

func (r *Router) With(middlewares ...Middleware) *middlewareGroup {
	return &middlewareGroup{
		r:           r,
		middlewares: middlewares,
	}
}

func (r *Router) Group(prefix string) *RouteGroup {
	if prefix == "" || prefix == "/" {
		panic(fmt.Sprintf("router: invalid group prefix %q (cannot be empty or '/')", prefix))
	}

	r.validatePath(prefix)

	if prefix[0] != '/' {
		prefix = "/" + prefix
	}

	return &RouteGroup{
		r:      r,
		prefix: prefix,
	}
}

func (g *RouteGroup) HandleFunc(url string, methods string, fn HandlerFunc) {
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	full := g.prefix + url

	g.r.insertGroupMiddleware(g.prefix, full)
	g.r.HandleFunc(full, methods, fn)
}

func (g *RouteGroup) GET(url string, fn HandlerFunc) {
	g.HandleFunc(url, "GET", fn)
}

func (g *RouteGroup) POST(url string, fn HandlerFunc) {
	g.HandleFunc(url, "POST", fn)
}

func (g *RouteGroup) PUT(url string, fn HandlerFunc) {
	g.HandleFunc(url, "PUT", fn)
}

func (g *RouteGroup) DELETE(url string, fn HandlerFunc) {
	g.HandleFunc(url, "DELETE", fn)
}

func (g *RouteGroup) PATCH(url string, fn HandlerFunc) {
	g.HandleFunc(url, "PATCH", fn)
}

func (g *RouteGroup) HEAD(url string, fn HandlerFunc) {
	g.HandleFunc(url, "HEAD", fn)
}

func (g *RouteGroup) OPTIONS(url string, fn HandlerFunc) {
	g.HandleFunc(url, "OPTIONS", fn)
}

func (g *RouteGroup) TRACE(url string, fn HandlerFunc) {
	g.HandleFunc(url, "TRACE", fn)
}

func (g *RouteGroup) CONNECT(url string, fn HandlerFunc) {
	g.HandleFunc(url, "CONNECT", fn)
}

func (g *RouteGroup) Use(m Middleware) {
	g.r.useGroup(m, g.prefix)
}

func (g *RouteGroup) Mount(prefix string, handler http.Handler) {
	full := g.prefix + prefix
	g.r.Mount(full, handler)
}

func (g *RouteGroup) MountFunc(prefix string, fn func(w http.ResponseWriter, req *http.Request)) {
	full := g.prefix + prefix
	g.r.Mount(full, http.HandlerFunc(fn))
}
