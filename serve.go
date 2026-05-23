package router

import (
	"fmt"
	"net/http"
)

func (r *Router) getErrorMessage(message any) error {
	var err error

	switch v := message.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf("panic occurred: %s", v)
	default:
		err = fmt.Errorf("panic occurred with unknown type: %v", v)
	}

	return err
}

func (r *Router) run(w http.ResponseWriter, req *http.Request, handler HandlerFunc, ctx *Context) {
	handler(w, req, ctx)
}

func (r *Router) write405(w http.ResponseWriter, mask int) {
	allow := r.maskToAllowHeader(mask)
	if allow != "" {
		w.Header().Set("Allow", allow)
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write([]byte("405 method not allowed"))
}

func (r *Router) validatePathEntry(ctx *Context, entry *routeEntry) bool {

	patterns := entry.Patterns
	segments := ctx.segments

	for i := 0; i < len(patterns); i++ {
		p := patterns[i]
		segment := segments[i].Value

		switch p.Type {
		case isString:
		case isMatch:
			if !p.RegexCompiled.MatchString(segment) {
				return false
			}
		case isPattern:
			if !p.Fn(segment) {
				return false
			}
		case isSubmatch:
			if p.RegexCompiled.FindStringSubmatch(segment) == nil {
				return false
			}
		}
	}

	return true
}

func (r *Router) finishRequest(w http.ResponseWriter, req *http.Request, ctx *Context) {
	if m := recover(); m != nil {
		err := r.getErrorMessage(m)
		if err != nil {
			r.logError(req, m)
			if r.recovery != nil {
				func() {
					defer func() {
						if message := recover(); message != nil {
							r.logError(req, message)
							http.Error(w, "Recovery middleware failed: an error occurred while executing the recovery handler.", http.StatusInternalServerError)
						}
					}()
					r.recovery(w, req, ctx)
				}()
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}
	}
	if ctx != nil {
		putContext(ctx)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := getContext()

	defer r.finishRequest(w, req, ctx)

	var foundPath bool
	var allowedMask int

	t := r.staticRoutes[req.URL.Path]

	bitmask := r.getBitmaskIndex(req.Method)

	if len(t) > 0 {
		for i := 0; i < len(t); i++ {
			if t[i].Bitmask&bitmask != 0 {
				ctx.handler = t[i]
				ctx.params = ctx.params[:0]
				ctx.entries = ctx.entries[:0]

				r.run(w, req, t[i].Handler, ctx)
				return
			}
			allowedMask |= t[i].Bitmask
		}

		r.write405(w, allowedMask)
		return
	} else if ok := r.search(req.URL.Path, ctx, bitmask); ok >= 1 {
		foundPath = true

		if ok == 1 {
			entry := &ctx.handler
			if entry.Validation {
				if !r.validatePathEntry(ctx, entry) {
					foundPath = false
				}
			}

			if foundPath {
				r.run(w, req, entry.Handler, ctx)
				return
			}
		} else if ok == 2 {
			allowedMask = ctx.handler.Bitmask
		}
	}

	if foundPath {
		r.write405(w, allowedMask)
		return
	}

	if r.notFound != nil {
		r.notFound(w, req, ctx)
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(notFound)
	}
}

func (r *Router) handler() http.HandlerFunc {

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.ServeHTTP(w, req)
	})

	return handler
}
