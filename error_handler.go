package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime"
	"strings"
)

func extractReqInfo(req *http.Request) (path, method string) {
	if req == nil {
		return "unknown", "UNKNOWN"
	}
	if req.URL == nil {
		return "unknown", req.Method
	}
	return req.URL.Path, req.Method
}

func panicMessage() []string {
	stackTrace := make([]byte, 4096)
	stackSize := runtime.Stack(stackTrace, false)
	regex := regexp.MustCompile(`([[:print:]]+\(.+?\))\s+(/[^:]+:\d+)`)
	matches := regex.FindAllStringSubmatch(string(stackTrace[:stackSize]), -1)

	var panicError []string

	if len(matches) >= 1 {
		if len(matches) > 3 {
			matches = matches[3:]
		}

		for i, pn := range matches {
			if i == 0 {
				if len(pn) >= 3 {
					panicError = append(panicError, pn[2])
				}
			}
		}
		panicError = append(panicError, "\n")
	} else {
		panicError = append(panicError, "No panic information found.")
	}

	return panicError
}

func (r *Router) logError(req *http.Request, message any) {

	errors := strings.Join(panicMessage(), "\n")
	path, method := extractReqInfo(req)

	slog.Error(fmt.Sprintf(
		"Panic occurred on URL %s | method [%s]\nError message: %v\n%s",
		path,
		method,
		message,
		errors,
	))
}
