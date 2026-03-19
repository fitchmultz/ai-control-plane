// workflow_test.go - Coverage for typed vendor evidence ingest workflows.
//
// Purpose:
//   - Verify vendor ingest normalization, schema validation, and artifact output.
//
// Responsibilities:
//   - Cover successful compliance-export ingest runs.
//   - Cover validation failures against the tracked schema contract.
//   - Keep ingest tests independent from live repo artifacts and runtimes.
//
// Scope:
//   - internal/ingest unit and workflow tests only.
//
// Usage:
//   - Run via `go test ./internal/ingest`.
//
// Invariants/Assumptions:
//   - Tests use temp repositories and deterministic timestamps.
package ingest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIngestComplianceExportWritesArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	writeSchemaFixture(t, repoRoot)
	originalNow := nowUTC
	nowUTC = func() time.Time { return time.Date(2026, 3, 19, 20, 30, 0, 0, time.UTC) }
	defer func() { nowUTC = originalNow }()

	result, err := Ingest(context.Background(), Options{
		RepoRoot:  repoRoot,
		Format:    FormatComplianceAPI,
		InputPath: filepath.Join(repoRoot, "input.json"),
		InputPayload: []byte(`{
  "compliance_export": {
    "request_id": "req-123",
    "model": "openai-gpt5.2",
    "provider": "openai",
    "created_at": "2026-03-19T19:00:00Z",
    "policy_action": "allowed",
    "session_id": "sess-1",
    "user": {"email": "alice@example.com"},
    "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_cost": 0.12}
  }
}`),
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Summary.OverallStatus != "PASS" || result.Summary.RecordCount != 1 || len(result.Issues) != 0 {
		t.Fatalf("unexpected summary/result: %+v issues=%v", result.Summary, result.Issues)
	}
	if !strings.HasSuffix(result.Summary.NormalizedPath, NormalizedJSONName) {
		t.Fatalf("normalized path = %q", result.Summary.NormalizedPath)
	}
	data, err := os.ReadFile(result.Summary.NormalizedPath)
	if err != nil {
		t.Fatalf("ReadFile(normalized) error = %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal(normalized) error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("normalized record count = %d", len(records))
	}
	principal := records[0]["principal"].(map[string]any)
	if principal["id"] != "alice@example.com" || principal["type"] != "user" {
		t.Fatalf("unexpected principal block: %+v", principal)
	}
	ai := records[0]["ai"].(map[string]any)
	request := ai["request"].(map[string]any)
	if request["id"] != "req-123" {
		t.Fatalf("unexpected request block: %+v", request)
	}
}

func TestIngestComplianceExportReportsSchemaIssues(t *testing.T) {
	repoRoot := t.TempDir()
	writeSchemaFixture(t, repoRoot)
	originalNow := nowUTC
	nowUTC = func() time.Time { return time.Date(2026, 3, 19, 20, 30, 0, 0, time.UTC) }
	defer func() { nowUTC = originalNow }()

	result, err := Ingest(context.Background(), Options{
		RepoRoot: repoRoot,
		Format:   FormatComplianceAPI,
		InputPayload: []byte(`{
  "request_id": "req-123",
  "model": "openai-gpt5.2",
  "provider": "openai",
  "created_at": "not-a-timestamp",
  "policy_action": "mystery",
  "user": {"email": ""},
  "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_cost": 0.12}
}`),
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Summary.OverallStatus != "FAIL" || len(result.Issues) == 0 {
		t.Fatalf("expected validation failure, got summary=%+v issues=%v", result.Summary, result.Issues)
	}
	joined := strings.Join(result.Issues, "\n")
	if !strings.Contains(joined, "principal.id is required") || !strings.Contains(joined, "policy.action must be one of") || !strings.Contains(joined, "ai.request.timestamp must be RFC3339") {
		t.Fatalf("unexpected validation issues: %s", joined)
	}
}

func writeSchemaFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "demo", "config")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
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
        - name: principal.email
          type: string
          required: false
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
    - name: ai_usage
      fields:
        - name: ai.tokens.prompt
          type: integer
          required: false
        - name: ai.tokens.completion
          type: integer
          required: false
        - name: ai.tokens.total
          type: integer
          required: false
        - name: ai.cost.amount
          type: number
          required: false
        - name: ai.cost.currency
          type: string
          required: false
    - name: policy_enforcement
      fields:
        - name: policy.action
          type: enum[allowed, blocked, redacted, rate_limited, error]
          required: true
    - name: correlation
      fields:
        - name: correlation.session.id
          type: string
          required: false
    - name: source
      fields:
        - name: source.type
          type: enum[gateway, otel, compliance_api]
          required: true
        - name: source.service.name
          type: string
          required: false
`
	if err := os.WriteFile(filepath.Join(path, "normalized_schema.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
