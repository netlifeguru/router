# Changelog

All notable changes to this project will be documented in this file.

This project follows:

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)

---

## [0.1.0] - 2026-05-23

### Added

- Initial public release
- Fast and idiomatic HTTP router for Go applications
- Radix-tree based route matching
- Static, wildcard, mounted, and parameterized route support
- Route parameters such as `/users/{id}`
- Prepared pattern matchers for UUIDs, digits, slugs, dates, hex values, base64 values, and safe path segments
- Custom regex route parameter support
- Route groups with inherited middleware
- Global, group-level, and route-level middleware pipeline
- Mount support for existing `http.Handler` and `http.HandlerFunc` implementations
- Per-request context with route parameter access
- Request-scoped key/value storage
- Built-in health check route helpers
- Built-in rate limiting guard middleware
- Built-in profiling server support using Go’s `net/http/pprof`
- Static file serving support with automatic `favicon.ico` handling
- Custom `NotFound` handler support
- Custom panic recovery handler support
- Panic logging through Go’s standard `log/slog`
- Access logging middleware
- Multi-server support
- Graceful shutdown for interrupt and termination signals
- Compatibility with Go’s standard `net/http` interfaces
- Optional integration with `github.com/netlifeguru/logger`
- README with installation, quick start, examples, documentation links, versioning, contributing, and license information
- Documentation links for NetLife Guru docs and pkg.go.dev

### Notes

- This is the first public `v0` release.
- The API may still change before `v1.0.0`.