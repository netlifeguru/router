package router

import (
	"fmt"
	"net/http"
	"regexp"
	"regexp/syntax"
	"strings"
	"sync/atomic"
)

const serverName = `NetLifeGuru`
const serverVersion = `v0.0.1`

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

type RouteGroup struct {
	r      *Router
	prefix string
}

type middlewareGroup struct {
	r           *Router
	middlewares []Middleware
}

type Listener struct {
	Addr string
}

type Listeners []Listener

type activeParts []string

type pattern struct {
	Slug          string
	Type          int
	RegexCompiled *regexp.Regexp
	Fn            matchFunc
}

var notFound = []byte("404 page not found")

type routeEntry struct {
	Route      string
	Parts      activeParts
	Patterns   []pattern
	Handler    HandlerFunc
	Bitmask    int
	Validation bool
}

type staticRoutes map[string][]routeEntry

type groupMiddleware struct {
	Route string
	Group string
}

type groupMiddlewares map[string]groupMiddleware

type Router struct {
	radixRoot        *radixNode
	staticRoutes     staticRoutes
	groupMiddlewares groupMiddlewares
	recovery         HandlerFunc
	notFound         HandlerFunc
	ready            atomic.Bool
	middlewares      map[string][]Middleware
}

func New() *Router {

	r := &Router{
		radixRoot:        &radixNode{},
		staticRoutes:     make(staticRoutes),
		groupMiddlewares: make(groupMiddlewares),
		middlewares:      make(map[string][]Middleware),
		recovery:         nil,
		notFound:         nil,
	}

	r.ready.Store(true)

	return r
}

const (
	isString = iota + 1
	isPattern
	isMatch
	isSubmatch
)

func (r *Router) removeWrapper(s string, start string, end string) string {
	var str string

	if len(s) >= 2 && s[0] == start[0] && s[len(s)-1] == end[0] {
		str = s[1 : len(s)-1]
	} else {
		str = s
	}

	return str
}

func (r *Router) findPatterns(str string) matchFunc {
	possibleRegExpPattern := r.removeWrapper(str, "(", ")")

	if pattern, ok := patternMatchers[possibleRegExpPattern]; ok {
		// function
		return pattern
	} else if pattern, ok := functionMatchers[possibleRegExpPattern]; ok {
		// function
		return pattern
	}

	return nil
}

func (r *Router) countCaptureGroups(s string) int {
	left := strings.Count(s, "(")
	right := strings.Count(s, ")")

	if left != right {
		return -1
	}

	return left
}

func (r *Router) parseSlug(isStatic, reqValidation bool, s, url string) (pattern, bool, bool) {
	var slugPattern pattern

	if s == "*" {
		s = "{wildcard}"
	}

	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		isStatic = false
		str := s[1 : len(s)-1]

		name := str
		pt := "any"

		if i := strings.IndexByte(str, ':'); i != -1 {
			name = str[:i]
			if i+1 < len(str) {
				pt = str[i+1:]
			} else {
				pt = ""
			}
		}

		slugPattern.Slug = name

		if pt == "" {
			panic(fmt.Sprintf("Error: Empty pattern in URL segment %q (route %s)\n", s, url))
		}

		if pt != "any" {
			reqValidation = true
		}

		if c := r.countCaptureGroups(pt); c > 0 {
			//FindAllStringSubmatch
			if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
				panic(fmt.Sprintf("Error: Wrong regular expression %q in URL pattern %s\n", pt, url))
			}

			slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
			slugPattern.Type = isSubmatch
		} else {
			//Match
			if fn := r.findPatterns(pt); fn != nil {
				slugPattern.Fn = fn
				slugPattern.Type = isPattern
			} else {
				if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
					panic(fmt.Sprintf("Error: Wrong regular expression %q in URL pattern %s\n", pt, url))
				}

				slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
				slugPattern.Type = isMatch
			}
		}

		return slugPattern, isStatic, reqValidation
	}

	slugPattern.Slug = s
	slugPattern.Type = isString

	return slugPattern, isStatic, reqValidation
}

func splitPath(path string) []string {
	var segments []string
	start := -1

	for i := 0; i < len(path); i++ {
		if path[i] != '/' {
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			segments = append(segments, path[start:i])
			start = -1
		}
	}

	if start != -1 {
		segments = append(segments, path[start:])
	}

	return segments
}

func (r *Router) preparePattern(url string) (activeParts, []pattern, bool, bool, string) {
	var (
		first         string
		patterns      []pattern
		isStatic      = true
		reqValidation = false
	)

	segments := splitPath(url)
	parts := make([]string, 0, len(segments))
	var ap activeParts

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		if first == "" {
			first = seg
		}

		p, st, rv := r.parseSlug(isStatic, reqValidation, seg, url)
		slugPart := p.Slug

		if p.Type != isString {
			ap = append(ap, p.Slug)
			slugPart = "*"

			if p.Slug == "wildcard" {
				slugPart = "**"
			}

			patterns = append(patterns, p)
		}

		parts = append(parts, slugPart)

		isStatic = st
		reqValidation = rv
	}

	radixURL := "/" + strings.Join(parts, "/")

	return ap, patterns, isStatic, reqValidation, radixURL
}

var valid = regexp.MustCompile(`^[/A-Za-z0-9._~-]+$`)

func (r *Router) validatePath(path string) {
	if !valid.MatchString(path) {
		panic(fmt.Sprintf("router: invalid route path %q (allowed characters: / A-Z a-z 0-9 - _ . ~)", path))
	}
}
