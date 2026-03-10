// inspection_test.go - Runtime inspector configuration coverage.
//
// Purpose:
//   - Verify runtime inspection uses repo-aware gateway settings consistently.
//
// Responsibilities:
//   - Confirm repo-local demo/.env values flow into the shared gateway client.
//   - Guard against regressions where direct acpctl runtime commands ignore repo config.
//
// Scope:
//   - Runtime inspector construction behavior only.
//
// Usage:
//   - Run via `go test ./internal/runtimeinspect`.
//
// Invariants/Assumptions:
//   - Tests use temporary repo fixtures and do not require a live runtime.
package runtimeinspect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
)

func TestNewInspectorUsesRepoAwareGatewaySettings(t *testing.T) {
	repoRoot := t.TempDir()
	envPath := filepath.Join(repoRoot, "demo", ".env")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(envPath, []byte("LITELLM_MASTER_KEY=repo-key\nGATEWAY_HOST=repo-host\nLITELLM_PORT=4444\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	origMasterKey, hadMasterKey := os.LookupEnv("LITELLM_MASTER_KEY")
	origGatewayHost, hadGatewayHost := os.LookupEnv("GATEWAY_HOST")
	origGatewayPort, hadGatewayPort := os.LookupEnv("LITELLM_PORT")
	defer os.Setenv("ACP_REPO_ROOT", origRepoRoot)
	if hadMasterKey {
		defer os.Setenv("LITELLM_MASTER_KEY", origMasterKey)
	} else {
		defer os.Unsetenv("LITELLM_MASTER_KEY")
	}
	if hadGatewayHost {
		defer os.Setenv("GATEWAY_HOST", origGatewayHost)
	} else {
		defer os.Unsetenv("GATEWAY_HOST")
	}
	if hadGatewayPort {
		defer os.Setenv("LITELLM_PORT", origGatewayPort)
	} else {
		defer os.Unsetenv("LITELLM_PORT")
	}
	os.Setenv("ACP_REPO_ROOT", repoRoot)
	os.Unsetenv("LITELLM_MASTER_KEY")
	os.Unsetenv("GATEWAY_HOST")
	os.Unsetenv("LITELLM_PORT")

	inspector := NewInspector(repoRoot)
	client, ok := inspector.gateway.(*gateway.Client)
	if !ok {
		t.Fatalf("expected gateway client, got %T", inspector.gateway)
	}
	if !client.HasMasterKey() {
		t.Fatalf("expected repo-aware gateway client to include master key")
	}
	if got := client.BaseURL(); got != "http://repo-host:4444" {
		t.Fatalf("BaseURL() = %q, want %q", got, "http://repo-host:4444")
	}
}
