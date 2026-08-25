package router

import (
	"testing"
	"unsafe"
)

func TestContextSizeIsCacheLineMultiple(t *testing.T) {
	size := unsafe.Sizeof(Context{})

	if contextCacheLineBytes == 0 {
		t.Fatal("contextCacheLineBytes must not be zero")
	}

	if size%contextCacheLineBytes != 0 {
		t.Fatalf(
			"sizeof(Context) = %d, want a multiple of assumed cache line %d; update cacheline_pad_*.go after changing Context",
			size,
			contextCacheLineBytes,
		)
	}
}
