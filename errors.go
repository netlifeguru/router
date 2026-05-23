package router

import "errors"

var (
	// ErrInvalidListenAddress indicates that a listen address is malformed
	// or cannot be split into host and port.
	ErrInvalidListenAddress = errors.New("router: invalid listen address")

	// ErrInvalidListenPort indicates that the port part of a listen address
	// is not a valid numeric port.
	ErrInvalidListenPort = errors.New("router: invalid listen port")

	// ErrListenFailed indicates that the router failed to bind a TCP listener.
	ErrListenFailed = errors.New("router: listen failed")

	// ErrServeFailed indicates that http.Server.Serve returned an unexpected error.
	ErrServeFailed = errors.New("router: serve failed")

	// ErrHijackerNotSupported indicates that the wrapped ResponseWriter
	// does not implement http.Hijacker.
	ErrHijackerNotSupported = errors.New("router: hijacker not supported")

	// ErrInvalidTrustedProxyCIDR indicates that a trusted proxy CIDR is invalid.
	ErrInvalidTrustedProxyCIDR = errors.New("router: invalid trusted proxy CIDR")
)
