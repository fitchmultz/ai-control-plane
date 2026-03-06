// client_test.go - Tests for gateway client authentication behavior.
//
// Purpose:
//
//	Verify that gateway status checks use master-key authorization and that
//	health/model checks require authorized HTTP 200 responses.
//
// Responsibilities:
//   - Assert Authorization header propagation for status requests
//   - Assert behavior when no master key is configured
//   - Assert Health/Models endpoint status interpretation
//
// Non-scope:
//   - Does not test real network infrastructure
//   - Does not test key-generation response payloads
//
// Invariants/Assumptions:
//   - Tests run against httptest servers for deterministic responses
//   - Status endpoint checks are read-only GET requests
package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func mustClientForServer(t *testing.T, server *httptest.Server, opts ...Option) *Client {
	t.Helper()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("failed to parse server port: %v", err)
	}

	baseOpts := []Option{
		WithHost(parsed.Hostname()),
		WithPort(port),
	}
	baseOpts = append(baseOpts, opts...)
	return NewClient(baseOpts...)
}

func TestDoStatusRequest_SendsAuthorizationHeaderWhenMasterKeyIsSet(t *testing.T) {
	t.Parallel()
	const expectedAuthHeader = "Bearer sk-test-master"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != expectedAuthHeader {
			t.Fatalf("expected Authorization header %q, got %q", expectedAuthHeader, got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := mustClientForServer(t, server, WithMasterKey("sk-test-master"))
	_, _, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDoStatusRequest_DoesNotSendAuthorizationHeaderWithoutMasterKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("expected no Authorization header, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := mustClientForServer(t, server, WithMasterKey(""))
	_, _, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestHealth_RequiresHTTP200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := mustClientForServer(t, server, WithMasterKey("sk-test-master"))

	healthy, code, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if healthy {
		t.Fatal("expected health check to fail on HTTP 401")
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, code)
	}
}

func TestModels_RequiresHTTP200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := mustClientForServer(t, server, WithMasterKey("sk-test-master"))

	accessible, code, err := client.Models(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if accessible {
		t.Fatal("expected models check to fail on HTTP 401")
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, code)
	}
}
