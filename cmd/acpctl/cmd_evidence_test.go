// cmd_evidence_test.go - Tests for the typed evidence ingest command.
//
// Purpose:
//   - Verify `acpctl evidence ingest` binds inputs and writes ingest artifacts.
//
// Responsibilities:
//   - Cover successful file-based ingest.
//   - Cover missing-input usage errors.
//   - Keep command tests independent from the live repository.
//
// Scope:
//   - Command-layer evidence ingest behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests use temp repositories and deterministic command output files.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunEvidenceIngestTypedIngestsFileInput(t *testing.T) {
	repoRoot := t.TempDir()
	writeEvidenceSchemaFixture(t, repoRoot)
	inputPath := filepath.Join(repoRoot, "compliance-export.json")
	writeFile(t, inputPath, `{
  "compliance_export": {
    "request_id": "req-123",
    "model": "openai-gpt5.2",
    "provider": "openai",
    "created_at": "2026-03-19T19:00:00Z",
    "policy_action": "allowed",
    "user": {"email": "alice@example.com"},
    "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_cost": 0.12}
  }
}`)
	stdout, stderr := newCommandOutputFiles(t)
	code := runEvidenceIngestTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, evidenceIngestOptions{
		RepoRoot:   repoRoot,
		OutputRoot: filepath.Join(repoRoot, "demo", "logs", "evidence", "vendor-ingest"),
		InputPath:  inputPath,
		SourceName: "vendor-compliance-export",
		Format:     "compliance-api",
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runEvidenceIngestTyped() exit = %d, want %d stderr=%s", code, exitcodes.ACPExitSuccess, readDBCommandOutput(t, stderr))
	}
	output := readDBCommandOutput(t, stdout)
	if !strings.Contains(output, "Vendor evidence ingest complete") || !strings.Contains(output, "Normalized") {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunEvidenceIngestTypedRejectsMissingInput(t *testing.T) {
	repoRoot := t.TempDir()
	stdout, stderr := newCommandOutputFiles(t)
	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()
	file, err := os.Create(filepath.Join(t.TempDir(), "stdin-empty"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()
	os.Stdin = file

	code := runEvidenceIngestTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, evidenceIngestOptions{
		RepoRoot: repoRoot,
		Format:   "compliance-api",
	})
	if code != exitcodes.ACPExitUsage {
		t.Fatalf("runEvidenceIngestTyped() exit = %d, want %d", code, exitcodes.ACPExitUsage)
	}
	if got := readDBCommandOutput(t, stderr); !strings.Contains(got, "stdin was empty") {
		t.Fatalf("stderr = %q", got)
	}
}

func writeEvidenceSchemaFixture(t *testing.T, repoRoot string) {
	t.Helper()
	content := `normalized_evidence_schema:
  version: "1.0.0"
  entity_groups:
    - name: principal
      fields:
        - name: principal.id
          type: string
          required: true
        - name: principal.type
          type: enum[user, service, api_key, unknown]
          required: true
    - name: ai_request
      fields:
        - name: ai.model.id
          type: string
          required: true
        - name: ai.provider
          type: string
          required: true
        - name: ai.request.id
          type: string
          required: true
        - name: ai.request.timestamp
          type: timestamp
          required: true
    - name: policy_enforcement
      fields:
        - name: policy.action
          type: enum[allowed, blocked, redacted, rate_limited, error]
          required: true
    - name: source
      fields:
        - name: source.type
          type: enum[gateway, otel, compliance_api]
          required: true
`
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "normalized_schema.yaml"), content)
}
