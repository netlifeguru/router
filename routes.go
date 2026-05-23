package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
)

func (r *Router) EnableProfiling(profilingServer string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		log.Printf("[pprof] Profiling enabled at %s://%s/debug/pprof/", "http", profilingServer)
		if err := http.ListenAndServe(profilingServer, mux); err != nil {
			log.Printf("[pprof] Error: %v", err)
		}
	}()
}

func (r *Router) SetReady(ready bool) {
	r.ready.Store(ready)
}

func (r *Router) IsReady() bool {
	return r.ready.Load()
}

func (r *Router) Liveness(path string, fn func(w http.ResponseWriter, req *http.Request)) {
	if path == "" {
		path = "/healthz"
	}
	r.MountFunc(path, fn)
}

func (r *Router) Readiness(path string, fn func(w http.ResponseWriter, req *http.Request)) {
	if path == "" {
		path = "/readyz"
	}
	r.MountFunc(path, fn)
}

func (r *Router) Recovery(fn HandlerFunc) {
	r.recovery = fn
}

func (r *Router) NotFound(fn HandlerFunc) {
	r.notFound = fn
}

func (r *Router) MountFunc(prefix string, fn func(w http.ResponseWriter, req *http.Request)) {
	r.Mount(prefix, http.HandlerFunc(fn))
}

func (r *Router) Mount(path string, handler http.Handler) {
	if path == "" || path == "/" {
		panic(fmt.Sprintf("router: invalid mount prefix '%s' (cannot be empty or root)", path))
	}

	path = strings.TrimSuffix(path, "/")

	wrapper := func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		n := len(ctx.params)

		for i := 0; i < n; i++ {
			req.SetPathValue(ctx.params[i].Key, ctx.params[i].Value)
		}

		handler.ServeHTTP(w, req)
	}

	r.HandleFunc(path, "ANY", wrapper)
}

func (r *Router) Static(urlPrefix string, rootPath string) {
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix += "/"
	}

	fs := http.FileServer(http.Dir(rootPath))
	fileHandler := http.StripPrefix(urlPrefix, fs)

	faviconPath := filepath.Join(rootPath, "favicon.ico")

	if _, err := os.Stat(faviconPath); err == nil {
		r.GET("/favicon.ico", func(w http.ResponseWriter, req *http.Request, _ *Context) {
			http.ServeFile(w, req, faviconPath)
		})
	}

	r.GET(urlPrefix+"*", func(writer http.ResponseWriter, request *http.Request, context *Context) {
		fileHandler.ServeHTTP(writer, request)
	})
}

func (r *Router) HandleFunc(url string, methods string, fn HandlerFunc) {
	if url == "" || url[0] != '/' {
		panic(fmt.Sprintf("router: route path must start with '/': %q", url))
	}
	if fn == nil {
		panic(fmt.Sprintf("router: nil handler for route %q", url))
	}

	parts, patterns, isStatic, reqValidation, radixURL := r.preparePattern(url)

	finalHandler := r.wrap(url, fn)

	entry := routeEntry{
		Route:      url,
		Patterns:   patterns,
		Parts:      parts,
		Handler:    finalHandler,
		Bitmask:    r.methodsToBitmask(methods),
		Validation: reqValidation,
	}

	if entry.Bitmask < 0 {
		panic(fmt.Sprintf("invalid HTTP method in route %q methods %q", url, methods))
	}

	if isStatic {
		r.staticRoutes[url] = append(r.staticRoutes[url], entry)
	} else {
		r.insertNode(radixURL, entry)
	}
}

func (r *Router) GET(url string, fn HandlerFunc) {
	r.HandleFunc(url, "GET", fn)
}

func (r *Router) POST(url string, fn HandlerFunc) {
	r.HandleFunc(url, "POST", fn)
}

func (r *Router) PUT(url string, fn HandlerFunc) {
	r.HandleFunc(url, "PUT", fn)
}

func (r *Router) DELETE(url string, fn HandlerFunc) {
	r.HandleFunc(url, "DELETE", fn)
}

func (r *Router) PATCH(url string, fn HandlerFunc) {
	r.HandleFunc(url, "PATCH", fn)
}

func (r *Router) HEAD(url string, fn HandlerFunc) {
	r.HandleFunc(url, "HEAD", fn)
}

func (r *Router) OPTIONS(url string, fn HandlerFunc) {
	r.HandleFunc(url, "OPTIONS", fn)
}

func (r *Router) CONNECT(url string, fn HandlerFunc) {
	r.HandleFunc(url, "CONNECT", fn)
}

func (r *Router) TRACE(url string, fn HandlerFunc) {
	r.HandleFunc(url, "TRACE", fn)
}
