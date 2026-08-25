//go:build 386 || wasm

package router

// Context's hot fields occupy 32 bytes on these 32-bit targets while Go uses
// a 64-byte cache-line assumption. Pad the object to one full assumed line.
type contextCachePad struct {
	_ [32]byte
}

const contextCacheLineBytes uintptr = 64
