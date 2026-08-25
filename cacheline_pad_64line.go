//go:build amd64 || loong64 || riscv64

package router

// Go uses a 64-byte cache-line padding assumption on these architectures.
// Context's hot fields are already 64 bytes on 64-bit targets.
type contextCachePad struct{}

const contextCacheLineBytes uintptr = 64
