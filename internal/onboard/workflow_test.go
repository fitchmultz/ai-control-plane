// workflow_test.go - Package onboard implementation tests.
//
// Purpose:
//   - Verify guided onboarding workflow behavior stays deterministic.
//
// Responsibilities:
//   - Cover wizard prompting, default resolution, output rendering, and verification.
//   - Assert secret redaction and one-time reveal behavior.
//   - Keep onboarding regression coverage focused on user-visible contract outcomes.
//
// Scope:
//   - Package-local workflow tests only.
//
// Usage:
//   - Run via `go test ./internal/onboard` or the project CI targets.
//
// Invariants/Assumptions:
//   - Tests must not require live gateway services.
//   - Equivalent inputs should produce deterministic outputs.
package onboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	result := Run(context.Background(), Options{RepoRoot: repoRoot, Tool: "invalid-tool"})
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
	for _, want := range []string{
		`export GATEWAY_URL="http://127.0.0.1:4000"`,
		`export OPENAI_BASE_URL="$GATEWAY_URL"`,
		`export OPENAI_API_KEY="sk-test-...cdef"`,
		`[OK] env/config contract`,
		`[SKIP] gateway reachability: network verification disabled by operator`,
		`Onboarding complete.`,
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected %q in output, got %s", want, result.Stdout)
		}
	}
	if strings.Contains(result.Stdout, "sk-test-full-key-1234567890-abcdef") {
		t.Fatalf("expected full key to stay hidden, got %s", result.Stdout)
	}
}

func TestRun_RevealPrintsFullKeyAfterRedactedSummary(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "codex",
		Mode:     "api-key",
		Host:     "127.0.0.1",
		Port:     "4000",
		ShowKey:  true,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "codex-cli",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})

	if !strings.Contains(result.Stdout, `export GATEWAY_URL="http://127.0.0.1:4000"`) {
		t.Fatalf("expected canonical gateway export, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export OPENAI_BASE_URL="$GATEWAY_URL"`) {
		t.Fatalf("expected OpenAI base URL to reference GATEWAY_URL, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export OPENAI_API_KEY="sk-test-...cdef"`) {
		t.Fatalf("expected redacted summary, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `Full key (shown once):`) || !strings.Contains(result.Stdout, `export OPENAI_API_KEY="sk-test-full-key-1234567890-abcdef"`) {
		t.Fatalf("expected one-time full key reveal, got %s", result.Stdout)
	}
}

func TestRun_WizardPromptsAndUsesDefaults(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	var promptOut strings.Builder
	input := strings.NewReader(strings.Join([]string{
		"2", // tool: claude
		"",  // mode: default api-key
		"",  // host
		"",  // port
		"n", // tls
		"",  // budget
		"",  // model
		"n", // verify
		"n", // reveal full key
	}, "\n"))

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Stdin:    input,
		Stdout:   &promptOut,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "claude-code",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})

	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(promptOut.String(), "ACP onboarding wizard") {
		t.Fatalf("expected wizard banner, got %s", promptOut.String())
	}
	if !strings.Contains(promptOut.String(), "Gateway contract: local default is http://127.0.0.1:4000") {
		t.Fatalf("expected gateway contract guidance, got %s", promptOut.String())
	}
	if !strings.Contains(result.Stdout, "Mode: api-key") {
		t.Fatalf("expected default api-key mode, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export GATEWAY_URL="http://127.0.0.1:4000"`) {
		t.Fatalf("expected canonical gateway export, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export ANTHROPIC_BASE_URL="$GATEWAY_URL"`) {
		t.Fatalf("expected Anthropic base URL to reference GATEWAY_URL, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export ANTHROPIC_API_KEY="sk-test-...cdef"`) {
		t.Fatalf("expected redacted key, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `[OK] env/config contract`) {
		t.Fatalf("expected local lint summary, got %s", result.Stdout)
	}
}

func TestRun_DirectModeSkipsPrereqEnv(t *testing.T) {
	repoRoot := t.TempDir()

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "codex",
		Mode:     "direct",
		Host:     "127.0.0.1",
		Port:     "4000",
	})
	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected direct mode success without demo env, got %d stderr=%s", result.ExitCode, result.Stderr)
	}
	for _, want := range []string{
		`export OTEL_EXPORTER_OTLP_ENDPOINT="http://127.0.0.1:4317"`,
		`[OK] env/config contract: generated OTEL exports are valid for direct mode`,
		`[SKIP] tool config writes: no ACP-managed tool config is required for this tool or mode`,
		`[SKIP] gateway reachability: network verification disabled by operator`,
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected %q in output, got %s", want, result.Stdout)
		}
	}
}

func TestRun_ClaudeSubscriptionRendersCustomHeaders(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "claude",
		Mode:     "subscription",
		Host:     "127.0.0.1",
		Port:     "4000",
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "claude-code-max",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})
	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `export GATEWAY_URL="http://127.0.0.1:4000"`) {
		t.Fatalf("expected canonical gateway export, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export ANTHROPIC_BASE_URL="$GATEWAY_URL"`) {
		t.Fatalf("expected Anthropic base URL to reference GATEWAY_URL, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `export ANTHROPIC_CUSTOM_HEADERS="x-litellm-api-key: Bearer sk-test-...cdef"`) {
		t.Fatalf("expected custom headers export, got %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "Key alias: claude-code-max") {
		t.Fatalf("expected subscription alias, got %s", result.Stdout)
	}
}

func TestRun_VerificationSummaryIncludesLocalLintAndNetworkChecks(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	home := t.TempDir()
	t.Setenv("HOME", home)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test-full-key-1234567890-abcdef" {
				t.Fatalf("authorization = %q", got)
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
		RepoRoot:    repoRoot,
		Tool:        "codex",
		Mode:        "api-key",
		Host:        host,
		Port:        port,
		Verify:      true,
		WriteConfig: true,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "codex-cli",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
		HTTPClient: server.Client(),
	})

	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("ExitCode = %d stderr=%s stdout=%s", result.ExitCode, result.Stderr, result.Stdout)
	}
	for _, want := range []string{
		"[OK] env/config contract",
		"[OK] tool config writes",
		"[OK] gateway reachability",
		"[OK] authorized model path",
		"Onboarding complete.",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, result.Stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); err != nil {
		t.Fatalf("expected managed Codex config to exist: %v", err)
	}
}

func TestRun_VerificationFailureReturnsDomainAndRemediation(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

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
		Host:     host,
		Port:     port,
		Verify:   true,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "codex-cli",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
		HTTPClient: server.Client(),
	})

	if result.ExitCode != exitcodes.ACPExitDomain {
		t.Fatalf("ExitCode = %d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "[FAIL] authorized model path") {
		t.Fatalf("expected failure summary, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "Rerun `acpctl onboard` to mint a fresh key and retry verification.") {
		t.Fatalf("expected remediation, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "Onboarding incomplete.") {
		t.Fatalf("expected incomplete banner, got:\n%s", result.Stdout)
	}
	if strings.Contains(result.Stdout, "Full key (shown once):") {
		t.Fatalf("full key should not be revealed on failed onboarding: %s", result.Stdout)
	}
}

func TestRun_UnmanagedCodexConfigReturnsDomainWithActionableRemediation(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	home := t.TempDir()
	t.Setenv("HOME", home)
	testutil.WriteFileMode(t, filepath.Join(home, ".codex", "config.toml"), "model = \"manual\"\n", 0o600)

	result := Run(context.Background(), Options{
		RepoRoot:    repoRoot,
		Tool:        "codex",
		Mode:        "api-key",
		Host:        "127.0.0.1",
		Port:        "4000",
		Verify:      false,
		WriteConfig: true,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "codex-cli",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})

	if result.ExitCode != exitcodes.ACPExitDomain {
		t.Fatalf("ExitCode = %d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "move it aside or merge the ACP settings manually") {
		t.Fatalf("expected unmanaged-config remediation, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "[FAIL] tool config writes") {
		t.Fatalf("expected tool config failure summary, got:\n%s", result.Stdout)
	}
}

func TestRun_VerifyFalseSkipsNetworkChecksButStillReportsLocalLint(t *testing.T) {
	repoRoot := t.TempDir()
	writeEnvFixture(t, repoRoot)

	result := Run(context.Background(), Options{
		RepoRoot: repoRoot,
		Tool:     "claude",
		Mode:     "api-key",
		Host:     "127.0.0.1",
		Port:     "4000",
		Verify:   false,
		KeyGenerator: fakeKeyGenerator{generated: GeneratedKey{
			Alias: "claude-code",
			Key:   "sk-test-full-key-1234567890-abcdef",
		}},
	})

	if result.ExitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("ExitCode = %d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "[OK] env/config contract") {
		t.Fatalf("expected local lint pass, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "[SKIP] gateway reachability: network verification disabled by operator") {
		t.Fatalf("expected network skip, got:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "Optional: rerun `acpctl onboard` and answer yes to verification if you want a live connectivity check.") {
		t.Fatalf("expected optional verify guidance, got:\n%s", result.Stdout)
	}
}

func writeEnvFixture(t *testing.T, repoRoot string) {
	t.Helper()
	testutil.WriteRepoFile(t, repoRoot, "demo/.env", "LITELLM_MASTER_KEY=sk-master-test-12345\n")
}
