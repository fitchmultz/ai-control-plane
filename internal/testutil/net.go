// Package testutil provides shared deterministic helpers for unit tests.
//
// Purpose:
//   - Centralize lightweight networking helpers for test-only setup.
//
// Responsibilities:
//   - Reserve ephemeral localhost ports without hard-coded collisions.
//   - Return cleanup hooks so callers can release ports deterministically.
//
// Scope:
//   - Generic networking helpers for unit tests only.
//
// Usage:
//   - Import from `_test.go` files and call `ReserveLocalPort`.
//
// Invariants/Assumptions:
//   - Helpers bind to `127.0.0.1` only.
//   - Returned cleanup functions are safe to call once.
package testutil

import (
	"net"
	"testing"
)

// ReserveLocalPort binds an ephemeral localhost TCP port until the caller releases it.
func ReserveLocalPort(t testing.TB) (int, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve local port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return port, func() {
		_ = listener.Close()
	}
}
