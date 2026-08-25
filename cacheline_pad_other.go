//go:build !386 && !amd64 && !arm && !arm64 && !loong64 && !mips && !mipsle && !mips64 && !mips64le && !ppc64 && !ppc64le && !riscv64 && !s390x && !wasm

package router

// Conservative fallback for future GOARCH values. If Go adds a new
// architecture, benchmark it and replace this fallback with an explicit
// cache-line assumption for that architecture.
type contextCachePad struct{}

const contextCacheLineBytes uintptr = 64
