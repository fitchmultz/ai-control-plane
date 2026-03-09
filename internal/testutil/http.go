// Package testutil provides shared deterministic helpers for unit tests.
//
// Purpose:
//   - Centralize small HTTP fixture helpers shared across package test suites.
//
// Responsibilities:
//   - Parse host/port details from server URLs for client construction.
//   - Fail fast on malformed test URLs.
//
// Scope:
//   - Generic HTTP parsing helpers for unit tests only.
//
// Usage:
//   - Import from `_test.go` files and call `HostPortFromURL`.
//
// Invariants/Assumptions:
//   - Inputs are expected to be valid HTTP(S) URLs produced by test servers.
//   - Returned ports are positive integers.
package testutil

import (
	"net/url"
	"strconv"
	"testing"
)

// HostPortFromURL extracts hostname and integer port from a server URL.
func HostPortFromURL(t testing.TB, raw string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse url port %q: %v", raw, err)
	}
	return parsed.Hostname(), port
}
