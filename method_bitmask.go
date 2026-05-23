package router

import "strings"

type httpMethod int

const (
	GET     = 1 << 0
	POST    = 1 << 1
	PUT     = 1 << 2
	DELETE  = 1 << 3
	PATCH   = 1 << 4
	HEAD    = 1 << 5
	OPTIONS = 1 << 6
	CONNECT = 1 << 7
	TRACE   = 1 << 8
	ANY     = 1 << 9
)

var methodMap = map[string]httpMethod{
	"GET":     GET,
	"POST":    POST,
	"PUT":     PUT,
	"DELETE":  DELETE,
	"PATCH":   PATCH,
	"HEAD":    HEAD,
	"OPTIONS": OPTIONS,
	"CONNECT": CONNECT,
	"TRACE":   TRACE,
	"ANY":     ANY,
}

func (r *Router) getMethodIndex(method string) int {
	if val, ok := methodMap[method]; ok {
		return int(val)
	}
	return -512
}

func (r *Router) indexToBit(i int) int {
	switch i {
	case GET:
		return 0
	case POST:
		return 1
	case PUT:
		return 2
	case DELETE:
		return 3
	case PATCH:
		return 4
	case HEAD:
		return 5
	case OPTIONS:
		return 6
	case CONNECT:
		return 7
	case TRACE:
		return 8
	case ANY:
		return 9
	}
	return 9
}

func (r *Router) getBitmaskIndex(m string) int {
	var method int

	switch m {
	case "GET":
		method = 1
	case "POST":
		method = 2
	case "PUT":
		method = 4
	case "DELETE":
		method = 8
	case "PATCH":
		method = 16
	case "HEAD":
		method = 32
	case "OPTIONS":
		method = 64
	case "CONNECT":
		method = 128
	case "TRACE":
		method = 256
	}

	return method
}

func (r *Router) methodsToBitmask(methods string) int {
	var bitmask int
	var seen [10]bool

	start := -1
	for i := 0; i < len(methods); i++ {
		if methods[i] != ' ' && start == -1 {
			start = i
		} else if methods[i] == ' ' && start != -1 {
			method := methods[start:i]
			start = -1
			index := r.getMethodIndex(method)
			if index == 512 {
				return 511
			}
			if index < 0 {
				return -1
			}
			if !seen[r.indexToBit(index)] {
				seen[r.indexToBit(index)] = true
				bitmask |= index
			}
		}
	}

	if start != -1 {
		method := methods[start:]
		index := r.getMethodIndex(method)
		if index == 512 {
			return 511
		}
		if index < 0 {
			return -1
		}
		if !seen[r.indexToBit(index)] {
			bitmask |= index
		}
	}

	return bitmask
}

func (r *Router) maskToAllowHeader(mask int) string {
	have := map[string]bool{}
	if mask&GET != 0 {
		have["GET"], have["HEAD"] = true, true
	}
	if mask&POST != 0 {
		have["POST"] = true
	}
	if mask&PUT != 0 {
		have["PUT"] = true
	}
	if mask&DELETE != 0 {
		have["DELETE"] = true
	}
	if mask&PATCH != 0 {
		have["PATCH"] = true
	}
	if mask&CONNECT != 0 {
		have["CONNECT"] = true
	}
	if mask&TRACE != 0 {
		have["TRACE"] = true
	}

	have["OPTIONS"] = true
	order := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "CONNECT", "TRACE"}
	out := make([]string, 0, len(order))

	for _, m := range order {
		if have[m] {
			out = append(out, m)
		}
	}

	return strings.Join(out, ", ")
}
