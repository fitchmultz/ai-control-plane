// Package gateway provides LiteLLM gateway client functionality.
//
// Purpose:
//   - Verify virtual-key generation success and failure branches directly.
//
// Responsibilities:
//   - Cover missing master-key failures.
//   - Verify successful key-generation response decoding.
//   - Verify HTTP and JSON failure shaping for callers.
//
// Scope:
//   - Unit tests for GenerateKey and response helpers only.
//
// Usage:
//   - Run with `go test ./internal/gateway`.
//
// Invariants/Assumptions:
//   - Tests use httptest servers instead of a live gateway.
//   - GenerateKey remains a pure HTTP client workflow.
package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestGenerateKey_RequiresMasterKey(t *testing.T) {
	t.Parallel()

	client := NewClient(WithHost("127.0.0.1"), WithPort(4000))
	_, err := client.GenerateKey(context.Background(), &GenerateKeyRequest{KeyAlias: "demo"})
	if err == nil || !strings.Contains(err.Error(), "master key is required") {
		t.Fatalf("expected missing master key error, got %v", err)
	}
}

func TestGenerateKey_SuccessFailureAndMalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/generate" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-master" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		switch r.Header.Get("X-Case") {
		case "success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":"sk-generated","key_alias":"demo","max_budget":10,"budget_duration":"30d"}`))
		case "malformed":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		}
	}))
	defer server.Close()

	host, port := testutil.HostPortFromURL(t, server.URL)
	client := NewClient(WithHost(host), WithPort(port), WithMasterKey("sk-master"))

	ctx := context.Background()
	req := &GenerateKeyRequest{KeyAlias: "demo", MaxBudget: 10, BudgetDuration: "30d"}

	ctxSuccess := context.WithValue(ctx, headerContextKey("X-Case"), "success")
	response, err := clientWithHeaderContext(client).GenerateKey(ctxSuccess, req)
	if err != nil {
		t.Fatalf("GenerateKey success returned error: %v", err)
	}
	if response.ExtractKey() != "sk-generated" {
		t.Fatalf("unexpected generated key: %+v", response)
	}

	ctxFailure := context.WithValue(ctx, headerContextKey("X-Case"), "failure")
	_, err = clientWithHeaderContext(client).GenerateKey(ctxFailure, req)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400 - bad request") {
		t.Fatalf("expected HTTP failure, got %v", err)
	}

	ctxMalformed := context.WithValue(ctx, headerContextKey("X-Case"), "malformed")
	_, err = clientWithHeaderContext(client).GenerateKey(ctxMalformed, req)
	if err == nil || !strings.Contains(err.Error(), "failed to parse response") {
		t.Fatalf("expected malformed JSON failure, got %v", err)
	}
}

type headerContextKey string

type headerInjectingRoundTripper struct {
	base http.RoundTripper
}

func (rt headerInjectingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if value, ok := req.Context().Value(headerContextKey("X-Case")).(string); ok {
		req.Header.Set("X-Case", value)
	}
	return rt.base.RoundTrip(req)
}

func clientWithHeaderContext(client *Client) *Client {
	clone := *client
	transport := http.DefaultTransport
	if client.httpClient.Transport != nil {
		transport = client.httpClient.Transport
	}
	clone.httpClient = &http.Client{
		Timeout:   client.httpClient.Timeout,
		Transport: headerInjectingRoundTripper{base: transport},
	}
	return &clone
}
