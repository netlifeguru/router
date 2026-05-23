package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractReqInfo_NilRequest(t *testing.T) {
	path, method := extractReqInfo(nil)

	if path != "unknown" {
		t.Errorf("expected path 'unknown', got %q", path)
	}
	if method != "UNKNOWN" {
		t.Errorf("expected method 'UNKNOWN', got %q", method)
	}
}

func TestExtractReqInfo_NilURL(t *testing.T) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    nil,
	}

	path, method := extractReqInfo(req)

	if path != "unknown" {
		t.Errorf("expected path 'unknown', got %q", path)
	}
	if method != http.MethodGet {
		t.Errorf("expected method %q, got %q", http.MethodGet, method)
	}
}

func TestExtractReqInfo_Valid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/test/path", nil)

	path, method := extractReqInfo(req)

	if path != "/test/path" {
		t.Errorf("expected path '/test/path', got %q", path)
	}
	if method != http.MethodPost {
		t.Errorf("expected method %q, got %q", http.MethodPost, method)
	}
}

func TestPanicMessage_NotEmpty(t *testing.T) {
	msg := panicMessage()
	if len(msg) == 0 {
		t.Fatalf("expected some panic message output, got empty slice")
	}
}

func TestPanicMessage_HasContentOrFallback(t *testing.T) {
	msg := panicMessage()
	joined := strings.Join(msg, "")
	if joined == "" {
		t.Fatalf("expected panic message string to be non-empty")
	}
}
