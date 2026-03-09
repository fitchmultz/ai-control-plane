// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Verify production config-profile helpers and parsing branches directly.
//
// Responsibilities:
//   - Cover secrets-file validation edge cases.
//   - Verify database URL and TLS exposure enforcement helpers.
//   - Lock down compose-port parsing and host normalization behavior.
//
// Scope:
//   - Unit tests for config profile helpers only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use isolated temp files instead of real host paths.
//   - Helper outputs remain deterministic for equivalent inputs.
package validation

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestValidateProductionSecretsEnvFile_RejectsSymlink(t *testing.T) {
	tempDir := t.TempDir()
	target := testutil.WriteFileMode(t, filepath.Join(tempDir, "secrets-target.env"), validProductionSecretsEnv(), 0o600)
	linkPath := filepath.Join(tempDir, "secrets.env")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("symlink secrets file: %v", err)
	}

	values, issues, err := validateProductionSecretsEnvFile(linkPath)
	if err != nil {
		t.Fatalf("validateProductionSecretsEnvFile returned error: %v", err)
	}
	if values != nil {
		t.Fatalf("expected nil values for symlink, got %v", values)
	}
	want := []string{linkPath + ": secrets file must not be a symlink"}
	if !reflect.DeepEqual(issues, want) {
		t.Fatalf("issues = %v, want %v", issues, want)
	}
}

func TestValidateProductionSecretsEnvFile_ReportsPermissionsAndMissingKeys(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Chmod(tempDir, 0o777); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}
	secretsPath := testutil.WriteFileMode(t, filepath.Join(tempDir, "secrets.env"), "LITELLM_MASTER_KEY=\n", 0o644)

	values, issues, err := validateProductionSecretsEnvFile(secretsPath)
	if err != nil {
		t.Fatalf("validateProductionSecretsEnvFile returned error: %v", err)
	}
	if values["LITELLM_MASTER_KEY"] != "" {
		t.Fatalf("expected empty master key, got %q", values["LITELLM_MASTER_KEY"])
	}
	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		"secrets file permissions must deny group/other access",
		"parent directory permissions must not be group/other writable",
		"required key LITELLM_MASTER_KEY is missing or empty",
		"required key LITELLM_SALT_KEY is missing or empty",
		"required key DATABASE_URL is missing or empty",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, issues)
		}
	}
}

func TestValidateDatabaseURL_CoversEmbeddedAndExternalBranches(t *testing.T) {
	embeddedURL, _ := url.Parse("postgresql://wrong:bad@db.example.com/acp")
	embeddedIssues := validateDatabaseURL("/tmp/secrets.env", map[string]string{
		"POSTGRES_USER":     "app",
		"POSTGRES_PASSWORD": "short",
		"POSTGRES_DB":       "acp",
	}, "embedded", embeddedURL)
	embeddedJoined := strings.Join(embeddedIssues, "\n")
	for _, expected := range []string{
		"embedded ACP_DATABASE_MODE requires DATABASE_URL host postgres",
		"POSTGRES_PASSWORD must be at least 16 characters",
		"DATABASE_URL username must match POSTGRES_USER",
		"DATABASE_URL password must match POSTGRES_PASSWORD",
	} {
		if !strings.Contains(embeddedJoined, expected) {
			t.Fatalf("expected embedded issue containing %q, got %v", expected, embeddedIssues)
		}
	}

	externalURL, _ := url.Parse("postgres://user:secret@db.example.com/acp?sslmode=disable")
	externalIssues := validateDatabaseURL("/tmp/secrets.env", map[string]string{}, "external", externalURL)
	if len(externalIssues) != 1 || !strings.Contains(externalIssues[0], "sslmode=require or stronger") {
		t.Fatalf("expected sslmode issue, got %v", externalIssues)
	}
}

func TestValidateExternalTLSExposure_RequiresCanonicalIngressSettings(t *testing.T) {
	issues := validateExternalTLSExposure("/tmp/secrets.env", map[string]string{
		"CADDYFILE_PATH":         "./config/caddy/Caddyfile.dev",
		"CADDY_ACME_CA":          "staging",
		"CADDY_DOMAIN":           "gateway.example.com",
		"LITELLM_PUBLIC_URL":     "http://gateway.example.com",
		"OTEL_INGEST_AUTH_TOKEN": "short",
	})

	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		"must use CADDYFILE_PATH=./config/caddy/Caddyfile.prod",
		"must use CADDY_ACME_CA=letsencrypt",
		"CADDY_EMAIL is required",
		"LITELLM_PUBLIC_URL must be a valid https:// URL",
		"OTEL_INGEST_AUTH_TOKEN must be at least 32 characters",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, issues)
		}
	}
}

func TestSplitComposePortAndNormalizationHelpers(t *testing.T) {
	t.Parallel()

	host, port := splitComposePort(`"${OTEL_PUBLISH_HOST:-127.0.0.1}:4317:4317/tcp"`)
	if host != "${OTEL_PUBLISH_HOST:-127.0.0.1}" || port != "4317" {
		t.Fatalf("env compose port = (%q, %q)", host, port)
	}
	host, port = splitComposePort(`127.0.0.1:4318:4318`)
	if host != "127.0.0.1" || port != "4318" {
		t.Fatalf("compose port = (%q, %q)", host, port)
	}
	host, port = splitComposePort("4318:4318")
	if host != "" || port != "" {
		t.Fatalf("expected short spec to be rejected, got (%q, %q)", host, port)
	}

	if normalizeDatabaseModeValue(" EXTERNAL ") != "external" {
		t.Fatal("expected external mode normalization")
	}
	if normalizeDatabaseModeValue("other") != "" {
		t.Fatal("expected unknown mode normalization to clear")
	}
	if !isLoopbackHost("localhost") || !isLoopbackHost("127.0.0.1") || !isLoopbackHost("[::1]") {
		t.Fatal("expected loopback hosts to pass")
	}
	if isLoopbackHost("db.example.com") {
		t.Fatal("expected non-loopback host to fail")
	}
}

func TestValidateSecretValueAndPostgresPassword(t *testing.T) {
	t.Parallel()

	secretIssues := validateSecretValue("/tmp/secrets.env", map[string]string{
		"LITELLM_MASTER_KEY": " change-me ",
	}, "LITELLM_MASTER_KEY", 32)
	joined := strings.Join(secretIssues, "\n")
	for _, expected := range []string{
		"LITELLM_MASTER_KEY must be at least 32 characters",
		"LITELLM_MASTER_KEY must not use placeholder/demo values",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, secretIssues)
		}
	}

	passwordIssues := validatePostgresPassword("/tmp/secrets.env", "password")
	passwordJoined := strings.Join(passwordIssues, "\n")
	for _, expected := range []string{
		"POSTGRES_PASSWORD must be at least 16 characters",
		"POSTGRES_PASSWORD must not use demo/default values",
	} {
		if !strings.Contains(passwordJoined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, passwordIssues)
		}
	}
}
