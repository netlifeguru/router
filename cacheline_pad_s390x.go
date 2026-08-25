//go:build s390x

package router

// Go assumes a 256-byte cache line on s390x. Context's hot fields occupy
// 64 bytes, therefore 192 bytes of padding keep the total size on a cache-line
// multiple and reduce the risk of false sharing between pooled Context values.
type contextCachePad struct {
	_ [192]byte
}

const contextCacheLineBytes uintptr = 256
