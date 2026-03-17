// client_keys_test.go - Tests for key inventory gateway helpers.
//
// Purpose:
//   - Verify virtual-key listing stays compatible with common LiteLLM payloads.
//
// Responsibilities:
//   - Cover envelope and direct-array responses.
//   - Keep returned key lists sorted by canonical alias.
//
// Scope:
//   - Unit tests for ListKeys only.
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
	"testing"
)

func TestClientListKeysSupportsEnvelopeAndDirectArray(t *testing.T) {
	t.Run("envelope", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/key/list" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"keys":[{"key_alias":"b"},{"key_alias":"a"}]}`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithMasterKey("sk-master"))
		keys, err := client.ListKeys(context.Background())
		if err != nil {
			t.Fatalf("ListKeys() error = %v", err)
		}
		if len(keys) != 2 || keys[0].Alias() != "a" || keys[1].Alias() != "b" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})

	t.Run("direct array", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`[{"key_alias":"demo"}]`))
		}))
		defer server.Close()

		client := NewClient(WithBaseURL(server.URL), WithMasterKey("sk-master"))
		keys, err := client.ListKeys(context.Background())
		if err != nil {
			t.Fatalf("ListKeys() error = %v", err)
		}
		if len(keys) != 1 || keys[0].Alias() != "demo" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})
}
