// workflow_test.go - Package onboard implementation.
//
// Purpose:
//   - Define this file's primary role within ACP.
//
// Responsibilities:
//   - Keep this file's behavior focused and deterministic.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package onboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

type fakeKeyGenerator struct {
	generated GeneratedKey
	err       error
}

func (f fakeKeyGenerator) Generate(context.Context, KeyRequest) (GeneratedKey, error) {
	return f.generated, f.err
}

func TestRun_InvalidToolReturnsUsage(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "invalid-tool",
	})

	if result.ExitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "unsupported tool") {
		t.Fatalf("expected explicit error, got %s", result.Stderr)
	}
}

func TestRun_RedactsGeneratedKeyByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "codex",
		Mode:     "api-key",
		Alias:    "codex-cli",
		Budget:   "10.00",
		Host:     "127.0.0.1",
		Port:     "4000",
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "codex-cli",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})

	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `export OPENAI_API_KEY="sk-test-...cdef"`) {
		t.Fatalf("expected redacted key, got %s", result.Stdout)
	}
	if strings.Contains(result.Stdout, "sk-test-full-key-1234567890-abcdef") {
		t.Fatalf("expected full key to stay hidden, got %s", result.Stdout)
	}
}

func TestRun_VerifyChecksGatewayEndpoints(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	var sawHealth bool
	var sawModels bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			sawHealth = true
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			sawModels = true
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test-full-key-1234567890-abcdef" {
				t.Fatalf("unexpected authorization header: %q", got)
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	host, port, ok := strings.Cut(hostPort, ":")
	if !ok {
		t.Fatalf("unexpected server URL: %s", server.URL)
	}

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "codex",
		Mode:     "api-key",
		Alias:    "verify-key",
		Budget:   "10.00",
		Host:     host,
		Port:     port,
		Verify:   true,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "verify-key",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
		HTTPClient: server.Client(),
	})

	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", result.ExitCode, result.Stderr)
	}
	if !sawHealth || !sawModels {
		t.Fatalf("expected both gateway checks, health=%t models=%t", sawHealth, sawModels)
	}
}

func writeEnvFixture(t *testing.T, repoRoot string) {
	t.Helper()
	testutil.WriteRepoFile(t, repoRoot, "demo/.env", "LITELLM_MASTER_KEY=sk-master-test-12345\n")
}
