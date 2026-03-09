// Package gateway provides LiteLLM gateway client functionality.
//
// Purpose:
//   - Verify gateway status aggregation and base URL parsing branches directly.
//
// Responsibilities:
//   - Cover scheme/host/port option handling.
//   - Verify Status probes surface reachability and authorization correctly.
//   - Lock down unreachable and unauthorized gateway probe behavior.
//
// Scope:
//   - Unit tests for gateway client status and URL helpers only.
//
// Usage:
//   - Run with `go test ./internal/gateway`.
//
// Invariants/Assumptions:
//   - Tests use httptest servers or closed listeners for deterministic behavior.
//   - No live gateway is required.
package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestClientBaseURLAndParsingOptions(t *testing.T) {
	t.Parallel()

	client := NewClient(WithHost("gateway.local"), WithPort(8443), WithScheme("https"))
	if got := client.BaseURL(); got != "https://gateway.local:8443" {
		t.Fatalf("BaseURL() = %q", got)
	}

	baseURLClient := NewClient(WithBaseURL("https://gw.example.com"))
	if baseURLClient.scheme != "https" || baseURLClient.host != "gw.example.com" || baseURLClient.port != 443 {
		t.Fatalf("unexpected parsed base URL client: %+v", baseURLClient)
	}
	if got := baseURLClient.BaseURL(); got != "https://gw.example.com" {
		t.Fatalf("BaseURL() = %q", got)
	}

	fallback := NewClient(WithBaseURL("http://gw.example.com"), WithHost("override.local"), WithPort(9000))
	if got := fallback.BaseURL(); got != "http://override.local:9000" {
		t.Fatalf("override BaseURL() = %q", got)
	}

	unchanged := NewClient(WithHost("kept.local"), WithPort(4000), WithBaseURL("http://broken:url"))
	if got := unchanged.BaseURL(); got != "http://kept.local:4000" {
		t.Fatalf("invalid base URL should not overwrite host/port, got %q", got)
	}

	timeoutClient := NewClient(WithTimeout(1500 * time.Millisecond))
	if timeoutClient.httpClient.Timeout != 1500*time.Millisecond {
		t.Fatalf("expected timeout override, got %v", timeoutClient.httpClient.Timeout)
	}
}

func TestStatus_ReportsHealthyAuthorizedGateway(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-master" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		switch r.URL.Path {
		case "/health", "/v1/models":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	host, port := testutil.HostPortFromURL(t, server.URL)
	status := NewClient(WithHost(host), WithPort(port), WithMasterKey("sk-master")).Status(context.Background())
	if !status.MasterKeyConfigured || !status.TLSEnabled && status.Scheme != "http" {
		t.Fatalf("unexpected status metadata: %+v", status)
	}
	if !status.Health.Healthy || !status.Models.Healthy {
		t.Fatalf("expected healthy probes, got %+v", status)
	}
	if !status.Health.Authorized || !status.Models.Authorized {
		t.Fatalf("expected authorized probes, got %+v", status)
	}
}

func TestStatus_ReportsUnauthorizedModelsAndUnreachableGateway(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	host, port := testutil.HostPortFromURL(t, server.URL)
	client := NewClient(WithHost(host), WithPort(port), WithMasterKey("sk-master"))
	status := client.Status(context.Background())
	server.Close()

	if status.Models.Healthy {
		t.Fatal("expected models probe to be unhealthy")
	}
	if status.Models.Authorized {
		t.Fatal("expected models probe to be unauthorized")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	unusedPort := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	unreachable := NewClient(
		WithHost("127.0.0.1"),
		WithPort(unusedPort),
		WithMasterKey("sk-master"),
		WithTimeout(250*time.Millisecond),
	)
	unreachable.connectTimeout = 250 * time.Millisecond
	result := unreachable.Status(context.Background())
	if result.Health.Error == "" {
		t.Fatalf("expected unreachable health error, got %+v", result)
	}
	if result.Health.Reachable || result.Models.Reachable {
		t.Fatalf("expected unreachable probes, got %+v", result)
	}
}
