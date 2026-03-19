// cmd_db_off_host_drill_test.go - Tests for off-host drill evidence rendering.
//
// Purpose:
//   - Verify replacement-host recovery evidence records drill mode and host.
//
// Responsibilities:
//   - Assert JSON evidence includes top-level drill metadata.
//   - Assert markdown evidence includes the correct claim-boundary statement.
//
// Scope:
//   - Pure command-package evidence rendering tests only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests remain local-only and do not require Docker or PostgreSQL.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/db"
)

func TestPersistReplacementHostRecoveryEvidenceIncludesDrillMetadata(t *testing.T) {
	repoRoot := t.TempDir()
	outputRoot := filepath.Join(t.TempDir(), "evidence")
	manifestPath := filepath.Join(repoRoot, "off_host_recovery.yaml")
	now := time.Date(2026, 3, 19, 1, 2, 3, 0, time.UTC)

	result := db.OffHostRecoveryResult{
		DrillMode:        db.OffHostRecoveryDrillModeSeparateHost,
		DrillHost:        "recovery-vm-01",
		BackupFile:       "/var/tmp/ai-control-plane-recovery/litellm-backup-20260319-010203.sql.gz",
		BackupSourceURI:  "s3://customer-dr-bucket/acp/litellm-backup-20260319-010203.sql.gz",
		BackupSHA256:     strings.Repeat("a", 64),
		BackupSizeBytes:  12345,
		InventoryPath:    "/opt/ai-control-plane/deploy/ansible/inventory/hosts.yml",
		SecretsEnvFile:   "/etc/ai-control-plane/secrets.env",
		RepoVersion:      "0.1.0",
		LocalBackupDir:   "/opt/ai-control-plane/demo/backups",
		UsedOffHostInput: true,
		ScratchDatabase:  "acp_offhost_drill_20260319_010203",
		Verification: db.RestoreVerification{
			DatabaseName:   "acp_offhost_drill_20260319_010203",
			ExpectedTables: 4,
			FoundTables:    4,
			Version:        "PostgreSQL test",
		},
	}

	summary, err := persistReplacementHostRecoveryEvidence(
		repoRoot,
		outputRoot,
		manifestPath,
		[]byte("drill_mode: separate-host\n"),
		result,
		now,
	)
	if err != nil {
		t.Fatalf("persistReplacementHostRecoveryEvidence() error = %v", err)
	}

	payload, err := os.ReadFile(filepath.Join(summary.RunDirectory, recoveryEvidenceSummaryJSON))
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}

	var decoded replacementHostRecoverySummary
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal summary json: %v", err)
	}
	if decoded.DrillMode != db.OffHostRecoveryDrillModeSeparateHost {
		t.Fatalf("DrillMode = %q, want %q", decoded.DrillMode, db.OffHostRecoveryDrillModeSeparateHost)
	}
	if decoded.DrillHost != "recovery-vm-01" {
		t.Fatalf("DrillHost = %q, want %q", decoded.DrillHost, "recovery-vm-01")
	}
	if !strings.Contains(decoded.EvidenceBoundary, "Separate-host or separate-VM proof") {
		t.Fatalf("EvidenceBoundary = %q", decoded.EvidenceBoundary)
	}

	markdown, err := os.ReadFile(filepath.Join(summary.RunDirectory, recoveryEvidenceSummaryMarkdown))
	if err != nil {
		t.Fatalf("read markdown summary: %v", err)
	}
	text := string(markdown)
	if !strings.Contains(text, "Drill mode: `separate-host`") {
		t.Fatalf("markdown missing drill mode: %s", text)
	}
	if !strings.Contains(text, "Drill host: `recovery-vm-01`") {
		t.Fatalf("markdown missing drill host: %s", text)
	}
	if !strings.Contains(text, "Separate-host or separate-VM proof") {
		t.Fatalf("markdown missing boundary statement: %s", text)
	}
}
