//go:build arm || mips || mipsle || mips64 || mips64le

package router

// Go uses a 32-byte cache-line padding assumption on these architectures.
// Context is already a multiple of that size, so no extra bytes are needed.
type contextCachePad struct{}

const contextCacheLineBytes uintptr = 32
