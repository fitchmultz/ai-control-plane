// off_host_recovery_test.go - Tests for off-host recovery contract helpers.
//
// Purpose:
//   - Validate off-host recovery contract normalization and file checks.
//
// Responsibilities:
//   - Reject staged backups that still live under the canonical local backup directory.
//   - Confirm repo-relative inventory paths normalize against the repo root.
//   - Verify compressed backup helpers return payload metadata and digests.
//
// Scope:
//   - Pure helper coverage for internal/db off-host recovery logic.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests remain local-only and do not require Docker or PostgreSQL.
package db

import (
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadOffHostRecoveryContractTrimsFields(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, "off_host_recovery.yaml")
	const content = `backup_file: /var/tmp/recovery/backup.sql.gz
backup_source_uri: "  s3://bucket/backup.sql.gz  "
backup_sha256: " 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef "
inventory_path: " deploy/ansible/inventory/hosts.yml "
secrets_env_file: " /etc/ai-control-plane/secrets.env "
expected_repo_version: " v1.2.3 "
notes: "  staged copy  "
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	contract, raw, err := LoadOffHostRecoveryContract(manifestPath)
	if err != nil {
		t.Fatalf("LoadOffHostRecoveryContract() error = %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected raw contract bytes")
	}
	if contract.BackupSourceURI != "s3://bucket/backup.sql.gz" {
		t.Fatalf("BackupSourceURI = %q", contract.BackupSourceURI)
	}
	if contract.InventoryPath != filepath.Clean("deploy/ansible/inventory/hosts.yml") {
		t.Fatalf("InventoryPath = %q", contract.InventoryPath)
	}
	if contract.SecretsEnvFile != "/etc/ai-control-plane/secrets.env" {
		t.Fatalf("SecretsEnvFile = %q", contract.SecretsEnvFile)
	}
	if contract.ExpectedRepoVersion != "v1.2.3" {
		t.Fatalf("ExpectedRepoVersion = %q", contract.ExpectedRepoVersion)
	}
	if contract.Notes != "staged copy" {
		t.Fatalf("Notes = %q", contract.Notes)
	}
}

func TestNormalizeOffHostRecoveryContractResolvesRelativeInventoryPath(t *testing.T) {
	repoRoot := t.TempDir()
	got := NormalizeOffHostRecoveryContract(repoRoot, OffHostRecoveryContract{
		InventoryPath: "deploy/ansible/inventory/hosts.yml",
	})
	want := filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml")
	if got.InventoryPath != want {
		t.Fatalf("InventoryPath = %q, want %q", got.InventoryPath, want)
	}
}

func TestValidateOffHostRecoveryContractHappyPath(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "deploy", "ansible", "inventory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "demo", "backups"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stagedDir := filepath.Join(t.TempDir(), "staged")
	backupPath := filepath.Join(stagedDir, "backup.sql.gz")
	if err := writeTestGzip(backupPath, "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml")
	if err := os.WriteFile(inventoryPath, []byte("all: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(secretsPath, []byte("TEST=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	contract := NormalizeOffHostRecoveryContract(repoRoot, OffHostRecoveryContract{
		BackupFile:          backupPath,
		BackupSourceURI:     "s3://bucket/backup.sql.gz",
		BackupSHA256:        strings.Repeat("a", 64),
		InventoryPath:       "deploy/ansible/inventory/hosts.yml",
		SecretsEnvFile:      secretsPath,
		ExpectedRepoVersion: "v1.2.3",
	})
	if err := ValidateOffHostRecoveryContract(repoRoot, contract); err != nil {
		t.Fatalf("ValidateOffHostRecoveryContract() error = %v", err)
	}
}

func TestValidateOffHostRecoveryContractRejectsRelativeSecretsPath(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "deploy", "ansible", "inventory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "demo", "backups"), 0o700); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(t.TempDir(), "staged", "backup.sql.gz")
	if err := writeTestGzip(backupPath, "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml")
	if err := os.WriteFile(inventoryPath, []byte("all: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := ValidateOffHostRecoveryContract(repoRoot, OffHostRecoveryContract{
		BackupFile:      backupPath,
		BackupSourceURI: "s3://bucket/backup.sql.gz",
		BackupSHA256:    strings.Repeat("a", 64),
		InventoryPath:   inventoryPath,
		SecretsEnvFile:  "relative.env",
	})
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected relative secrets rejection, got %v", err)
	}
}

func TestValidateOffHostRecoveryContractRejectsRepoVersionMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "deploy", "ansible", "inventory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(t.TempDir(), "staged", "backup.sql.gz")
	if err := writeTestGzip(backupPath, "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml")
	if err := os.WriteFile(inventoryPath, []byte("all: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(secretsPath, []byte("TEST=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := ValidateOffHostRecoveryContract(repoRoot, OffHostRecoveryContract{
		BackupFile:          backupPath,
		BackupSourceURI:     "s3://bucket/backup.sql.gz",
		BackupSHA256:        strings.Repeat("a", 64),
		InventoryPath:       inventoryPath,
		SecretsEnvFile:      secretsPath,
		ExpectedRepoVersion: "v9.9.9",
	})
	if err == nil || !strings.Contains(err.Error(), "expected_repo_version mismatch") {
		t.Fatalf("expected repo version mismatch, got %v", err)
	}
}

func TestValidateOffHostRecoveryContractRejectsCanonicalLocalBackupDir(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "demo", "backups"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("test-version\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(repoRoot, "demo", "backups", "bad.sql.gz")
	if err := writeTestGzip(backupPath, "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml")
	if err := os.MkdirAll(filepath.Dir(inventoryPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventoryPath, []byte("all: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(secretsPath, []byte("TEST=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := ValidateOffHostRecoveryContract(repoRoot, OffHostRecoveryContract{
		BackupFile:          backupPath,
		BackupSourceURI:     "s3://bucket/backup.sql.gz",
		BackupSHA256:        strings.Repeat("a", 64),
		InventoryPath:       inventoryPath,
		SecretsEnvFile:      secretsPath,
		ExpectedRepoVersion: "test-version",
	})
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("expected canonical local backup rejection, got %v", err)
	}
}

func TestRunOffHostRecoveryDrillRejectsInvalidContractBeforeDatabaseAccess(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := (*AdminService)(nil)
	_, err := service.RunOffHostRecoveryDrill(context.Background(), repoRoot, OffHostRecoveryContract{}, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "backup_file is required") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestReadCompressedBackupReturnsDigestAndPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.sql.gz")
	if err := writeTestGzip(path, "CREATE DATABASE litellm;"); err != nil {
		t.Fatal(err)
	}
	payload, digest, sizeBytes, err := readCompressedBackup(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, "CREATE DATABASE") {
		t.Fatalf("expected SQL payload, got %q", payload)
	}
	if len(digest) != 64 {
		t.Fatalf("expected sha256 digest, got %q", digest)
	}
	if sizeBytes <= 0 {
		t.Fatalf("expected positive size, got %d", sizeBytes)
	}
}

func TestReadCompressedBackupRejectsInvalidGzip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.sql.gz")
	if err := os.WriteFile(path, []byte("not-gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := readCompressedBackup(path); err == nil {
		t.Fatal("expected invalid gzip error")
	}
}

func TestReadTrackedRepoVersionAndPathWithinBase(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("v9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	version, err := readTrackedRepoVersion(repoRoot)
	if err != nil {
		t.Fatalf("readTrackedRepoVersion() error = %v", err)
	}
	if version != "v9.9.9" {
		t.Fatalf("version = %q, want %q", version, "v9.9.9")
	}

	base := filepath.Join(repoRoot, "demo", "backups")
	inside, err := pathWithinBase(filepath.Join(base, "backup.sql.gz"), base)
	if err != nil {
		t.Fatalf("pathWithinBase() error = %v", err)
	}
	if !inside {
		t.Fatal("expected file to be inside base path")
	}

	exact, err := pathWithinBase(base, base)
	if err != nil {
		t.Fatalf("pathWithinBase() error = %v", err)
	}
	if !exact {
		t.Fatal("expected base path to count as inside")
	}

	outside, err := pathWithinBase(filepath.Join(repoRoot, "elsewhere", "backup.sql.gz"), base)
	if err != nil {
		t.Fatalf("pathWithinBase() error = %v", err)
	}
	if outside {
		t.Fatal("expected file to be outside base path")
	}
}

func TestLoadOffHostRecoveryContractRejectsInvalidYAML(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(manifestPath, []byte("backup_file: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOffHostRecoveryContract(manifestPath); err == nil {
		t.Fatal("expected YAML parse error")
	}
}

func TestLoadOffHostRecoveryContractRejectsMissingFile(t *testing.T) {
	if _, _, err := LoadOffHostRecoveryContract(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestDropScratchDatabaseRejectsMissingConnector(t *testing.T) {
	admin := &AdminService{}
	err := dropScratchDatabase(admin, "scratch_db")
	if err == nil || !strings.Contains(err.Error(), "database admin service requires a connector") {
		t.Fatalf("expected connector error, got %v", err)
	}
}

func TestExpectedCoreSchemaTableCount(t *testing.T) {
	if got := ExpectedCoreSchemaTableCount(); got != 4 {
		t.Fatalf("ExpectedCoreSchemaTableCount() = %d, want 4", got)
	}
}

func writeTestGzip(path string, payload string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte(payload)); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
