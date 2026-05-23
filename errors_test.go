package router

import (
	"errors"
	"fmt"
	"testing"
)

func TestRouterErrorsAreDefined(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrInvalidListenAddress",
			err:  ErrInvalidListenAddress,
			want: "router: invalid listen address",
		},
		{
			name: "ErrInvalidListenPort",
			err:  ErrInvalidListenPort,
			want: "router: invalid listen port",
		},
		{
			name: "ErrListenFailed",
			err:  ErrListenFailed,
			want: "router: listen failed",
		},
		{
			name: "ErrServeFailed",
			err:  ErrServeFailed,
			want: "router: serve failed",
		},
		{
			name: "ErrHijackerNotSupported",
			err:  ErrHijackerNotSupported,
			want: "router: hijacker not supported",
		},
		{
			name: "ErrInvalidTrustedProxyCIDR",
			err:  ErrInvalidTrustedProxyCIDR,
			want: "router: invalid trusted proxy CIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("%s is nil", tt.name)
			}

			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("%s.Error() = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRouterErrorsSupportErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		target error
	}{
		{
			name:   "ErrInvalidListenAddress",
			target: ErrInvalidListenAddress,
		},
		{
			name:   "ErrInvalidListenPort",
			target: ErrInvalidListenPort,
		},
		{
			name:   "ErrListenFailed",
			target: ErrListenFailed,
		},
		{
			name:   "ErrServeFailed",
			target: ErrServeFailed,
		},
		{
			name:   "ErrHijackerNotSupported",
			target: ErrHijackerNotSupported,
		},
		{
			name:   "ErrInvalidTrustedProxyCIDR",
			target: ErrInvalidTrustedProxyCIDR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("wrapped error: %w", tt.target)

			if !errors.Is(err, tt.target) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", err, tt.target)
			}
		})
	}
}
