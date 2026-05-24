# NLG HTTP Router

A fast and idiomatic HTTP router for modern Go applications.

> Use router to build APIs, web services, and backend applications with radix-tree routing, middleware, route groups, request context, static assets, rate limiting, profiling, and multi-server support.

[![Go Reference](https://pkg.go.dev/badge/github.com/netlifeguru/router.svg)](https://pkg.go.dev/github.com/netlifeguru/router)
[![Go Report Card](https://goreportcard.com/badge/github.com/netlifeguru/router)](https://goreportcard.com/report/github.com/netlifeguru/router)
[![Go Version](https://img.shields.io/badge/go-%3E=1.25-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-brightgreen)](LICENSE)

---

## About

`router` is a production-oriented HTTP routing package for Go focused on performance, composability, and clean application structure.

It combines radix-tree based route matching with middleware pipelines, hierarchical route grouping, request-scoped context utilities, recovery handling, rate limiting, profiling support, static asset serving, and multi-server orchestration.

The package is built on top of Go’s standard `net/http` interfaces and is designed for APIs, web services, microservices, backend platforms, and internal services.

## Features

- **Fast Radix Router**: Optimized radix-tree based route lookup for static, wildcard, mounted, and parameterized routes
- **Parameterized Routes**: Supports route parameters such as `/users/{id}`
- **Prepared Pattern Matching**: Built-in matchers for UUIDs, digits, slugs, dates, hex values, base64 values, and safe path segments
- **Regex Route Parameters**: Define custom route parameters with regular expressions
- **Route Groups**: Organize routes using hierarchical groups with inherited middleware
- **Middleware Pipeline**: Compose global, grouped, and route-level middleware
- **Mount Support**: Mount existing `http.Handler` and `http.HandlerFunc` implementations under route prefixes
- **Pooled Request Context**: Per-request context with parameter access and temporary key/value storage
- **Health Check Endpoints**: Built-in liveness and readiness route helpers
- **Rate Limiting Guard**: Built-in middleware for request throttling and cooldown-based protection
- **Built-in pprof Profiling**: Enable a dedicated profiling server with Go’s standard `net/http/pprof`
- **Static File Serving**: Serve static directories with automatic `favicon.ico` support
- **Custom NotFound Handler**: Override the default `404 page not found` response
- **Custom Recovery Handler**: Recover from panics and provide your own fallback response
- **Panic Logging**: Panic details are logged through Go’s standard `log/slog`
- **Access Logging**: Optional middleware for structured request logging
- **Multi-Server Support**: Run multiple listeners from a single router
- **Graceful Shutdown**: Handles interrupt and termination signals with safe HTTP server shutdown
- **Standard Library Compatible**: Works with `net/http`, `http.Handler`, `http.HandlerFunc`, and `log/slog`

## Requirements

This package requires Go 1.25 or newer.

- **Go:** `1.25` or newer
- **Dependencies:** Standard library + `github.com/netlifeguru/logger`

## Installation

Add the package to your project using `go get`:

```bash
go get github.com/netlifeguru/router@v0.1.0
```

## Quick Start

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/netlifeguru/logger"
	"github.com/netlifeguru/router"
)

func main() {
	r := router.New()

	closer, err := logger.Init(logger.Config{
		Dir:             "./logs",
		TerminalOutput:  true,
		DisableColors:   false,
		MinLevel:        slog.LevelInfo,
		ConsoleMinLevel: slog.LevelDebug,
		MaxFileSize:     100 * 1024 * 1024,
		MaxLogFiles:     10,
	})
	if err != nil {
		slog.Error("failed to initialize logger", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := closer.Close(); err != nil {
			slog.Error("failed to close logger", "error", err)
		}
	}()

	r.Use(router.Logger())

	r.HandleFunc("/", "GET POST", func(w http.ResponseWriter, req *http.Request, ctx *router.Context) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`<h1>Hello World</h1>`))
	})

	r.HandleFunc("/user/{id}", "GET", func(w http.ResponseWriter, req *http.Request, ctx *router.Context) {
		id := ctx.Param("id")
		_, _ = w.Write([]byte(id))
	})

	if err := r.ListenAndServe(":8000"); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
```

Run the application:

```bash
go run main.go
```

Example requests:

```bash
curl http://localhost:8000/
curl http://localhost:8000/user/42
```

Expected response:

```text
42
```

## Examples

Practical examples are available in the official examples repository:

[https://github.com/netlifeguru/examples/router](https://github.com/netlifeguru/examples/router)

Example categories include:

- default router setup
- handlers
- middleware
- route groups
- mounting handlers
- custom route patterns
- static file serving
- health check endpoints
- request logging
- recovery middleware
- custom error handling
- rate limiting
- built-in profiling
- multi-server setup
- observability integrations

---

## Documentation

Full package documentation, guides, and examples are available at:

[https://netlife.guru/docs/go/router](https://netlife.guru/docs/go/router)

API reference is also available on pkg.go.dev:

[https://pkg.go.dev/github.com/netlifeguru/router](https://pkg.go.dev/github.com/netlifeguru/router)

---

## Notes

- Review package-specific concurrency behavior before using it in highly parallel workloads.
- Check performance characteristics when using this package in latency-sensitive paths.
- Observability examples may require additional third-party packages and external tooling.
- See the package documentation and examples for limitations and recommended usage patterns.

---

## Versioning

This project follows Semantic Versioning.

See [`CHANGELOG.md`](./CHANGELOG.md) for release history and breaking changes.

---

## Contributing

Community contributions, feedback, and improvements are welcome.

Please read [`CONTRIBUTING.md`](./CONTRIBUTING.md) before submitting pull requests or opening issues.

---

## Code of Conduct

This project follows a Code of Conduct.

Please read [`CODE_OF_CONDUCT.md`](./CODE_OF_CONDUCT.md) before contributing or participating in discussions.

---

## Author

Created and maintained by [NetLife Guru s.r.o.](https://netlife.guru)

- Documentation: [https://netlife.guru/docs/go/router](https://netlife.guru/docs/go/router)
- GitHub: [https://github.com/netlifeguru](https://github.com/netlifeguru)
- Contact: [info@netlife.guru](mailto:info@netlife.guru)

---

## License

MIT License. See [`LICENSE`](./LICENSE).
