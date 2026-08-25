# Changelog

All notable changes to this project will be documented in this file.

This project follows:

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)

---

## [1.1.0] - 2026-08-25

### Added

- Architecture-aware cache-line padding for request contexts to reduce false sharing under high concurrency
- Platform-specific cache-line definitions using Go build constraints
- Additional tests covering request context memory layout and cache-line alignment assumptions
- Performance diagnostics for parallel route matching and request context reuse

### Changed

- Optimized radix-tree dynamic route matching
- Replaced per-request route entry copies with direct route entry references
- Removed redundant route parameter materialization from the request hot path
- Route parameters are now resolved directly from matched route parts and captured segments
- Optimized radix child lookup for static path segments
- Reduced dynamic route traversal overhead
- Improved wildcard and parameter prefix matching
- Simplified request context state and reset logic
- Improved request context reuse through `sync.Pool`
- Updated mounted handler path parameter propagation for the optimized context representation
- Updated rate limiter route lookup for the optimized request context
- Improved parallel request scalability on multi-core systems
- Updated internal tests to match the optimized router and context implementation

### Performance

- Maintains zero allocations in the router request hot path
- Improved single-parameter route matching performance
- Improved deeply parameterized route matching performance
- Improved large dynamic route table lookup performance
- Reduced request routing overhead under parallel workloads
- Eliminated significant false-sharing overhead observed on Apple Silicon under high concurrency
- Improved scaling across multiple CPU cores while preserving `0 B/op` and `0 allocs/op`

### Fixed

- Fixed request context cache-line contention under high parallelism
- Fixed performance instability caused by false sharing between pooled request contexts
- Fixed tests relying on removed internal request context fields and previous radix-tree indexing structures

### Notes

- These changes are internal and do not intentionally change the public router API.
- Request context padding is architecture-aware because cache-line characteristics differ between CPU architectures.
- `GOMAXPROCS` is intentionally left under the control of the Go runtime or application.
- Router matching performance has been validated with serial and parallel benchmarks on Apple M2 Max.
- Server listener and `SO_REUSEPORT` worker configuration remain separate from these router hot-path optimizations.

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