// client_more_test.go - Additional coverage for gateway client helpers.
//
// Purpose:
//   - Exercise lightweight gateway client branches not covered by the core suites.
//
// Responsibilities:
//   - Cover injected HTTP clients and alias fallback behavior.
//   - Verify key-list prerequisite and parse-failure shaping.
//
// Scope:
//   - Small helper and error-path coverage only.
//
// Usage:
//   - Run with `go test ./internal/gateway`.
//
// Invariants/Assumptions:
//   - Tests use httptest servers instead of a live gateway.
package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithHTTPClientAndAliasFallback(t *testing.T) {
	injected := &http.Client{Timeout: 2 * time.Second}
	client := NewClient(WithHTTPClient(injected))
	if client.httpClient != injected {
		t.Fatalf("expected injected HTTP client to be preserved")
	}
	if alias := (KeyInfo{AliasValue: "fallback-alias"}).Alias(); alias != "fallback-alias" {
		t.Fatalf("Alias() = %q", alias)
	}
}

func TestListKeysRequiresMasterKeyAndSurfacesParseFailures(t *testing.T) {
	client := NewClient(WithBaseURL("http://127.0.0.1:4000"))
	if _, err := client.ListKeys(context.Background()); err == nil || !strings.Contains(err.Error(), "master key is required") {
		t.Fatalf("expected missing master key error, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"keys":`))
	}))
	defer server.Close()

	client = NewClient(WithBaseURL(server.URL), WithMasterKey("sk-master"))
	if _, err := client.ListKeys(context.Background()); err == nil || !strings.Contains(err.Error(), "failed to parse key list response") {
		t.Fatalf("expected parse failure, got %v", err)
	}
}
