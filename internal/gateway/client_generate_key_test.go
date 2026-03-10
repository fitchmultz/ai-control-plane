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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestDeleteKey_SuccessAndFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/delete" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-master" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		var payload DeleteKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode delete payload: %v", err)
		}
		if payload.KeyAlias != "demo" {
			t.Fatalf("unexpected delete alias: %q", payload.KeyAlias)
		}
		if r.Header.Get("X-Case") == "failure" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"deleted"}`))
	}))
	defer server.Close()

	host, port := testutil.HostPortFromURL(t, server.URL)
	client := NewClient(WithHost(host), WithPort(port), WithMasterKey("sk-master"))

	if err := client.DeleteKey(context.Background(), "demo"); err != nil {
		t.Fatalf("DeleteKey success returned error: %v", err)
	}

	ctxFailure := context.WithValue(context.Background(), headerContextKey("X-Case"), "failure")
	err := clientWithHeaderContext(client).DeleteKey(ctxFailure, "demo")
	if err == nil || !strings.Contains(err.Error(), "HTTP 400 - bad request") {
		t.Fatalf("expected HTTP failure, got %v", err)
	}
}

func TestDeleteKey_RequiresMasterKeyAndAlias(t *testing.T) {
	t.Parallel()

	client := NewClient(WithHost("127.0.0.1"), WithPort(4000))
	if err := client.DeleteKey(context.Background(), "demo"); err == nil || !strings.Contains(err.Error(), "master key is required") {
		t.Fatalf("expected missing master key error, got %v", err)
	}

	client = NewClient(WithHost("127.0.0.1"), WithPort(4000), WithMasterKey("sk-master"))
	if err := client.DeleteKey(context.Background(), "   "); err == nil || !strings.Contains(err.Error(), "key alias is required") {
		t.Fatalf("expected missing alias error, got %v", err)
	}
}

func TestNewClientDefaultsIgnoreRepoEnvButOptionsCanInjectRepoAwareSettings(t *testing.T) {
	repoRoot := t.TempDir()
	envPath := filepath.Join(repoRoot, "demo", ".env")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(envPath, []byte("LITELLM_MASTER_KEY=repo-key\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("ACP_REPO_ROOT", repoRoot)
	t.Setenv("LITELLM_MASTER_KEY", "")

	defaultClient := NewClient()
	if defaultClient.HasMasterKey() {
		t.Fatalf("expected default client to ignore repo env fallback")
	}

	repoAwareClient := NewClient(WithMasterKey("repo-key"))
	if !repoAwareClient.HasMasterKey() {
		t.Fatalf("expected options to inject repo-aware master key")
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
