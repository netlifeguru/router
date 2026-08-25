//go:build arm64 || ppc64 || ppc64le

package router

// Go assumes a 128-byte cache line on these architectures. Context's hot
// fields occupy 64 bytes on 64-bit targets, so another 64 bytes makes each
// pooled Context occupy a whole 128-byte size-class slot instead of allowing
// two independently-written Context values to share one assumed cache line.
type contextCachePad struct {
	_ [64]byte
}

const contextCacheLineBytes uintptr = 128
