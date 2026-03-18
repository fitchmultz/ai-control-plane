// off_host_recovery.go - Off-host backup contract validation and recovery drills.
//
// Purpose:
//   - Validate customer-owned off-host recovery manifests.
//   - Prove a staged off-host backup copy can restore into a scratch database.
//
// Responsibilities:
//   - Load and normalize off-host recovery contract files.
//   - Enforce the staged-backup contract for replacement-host drills.
//   - Verify digests and run scratch restore validation through the admin service.
//
// Scope:
//   - Off-host backup manifest validation and drill execution only.
//
// Usage:
//   - Used by `acpctl db off-host-drill` and related readiness evidence flows.
//
// Invariants/Assumptions:
//   - ACP validates staged off-host inputs but does not automate customer backup transport.
//   - Staged backup files must remain outside the repo-local `demo/backups/` directory.
package db

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var sha256HexPattern = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)

// OffHostRecoveryContract captures the staged off-host recovery inputs.
type OffHostRecoveryContract struct {
	BackupFile          string `yaml:"backup_file" json:"backup_file"`
	BackupSourceURI     string `yaml:"backup_source_uri" json:"backup_source_uri"`
	BackupSHA256        string `yaml:"backup_sha256" json:"backup_sha256"`
	InventoryPath       string `yaml:"inventory_path" json:"inventory_path"`
	SecretsEnvFile      string `yaml:"secrets_env_file" json:"secrets_env_file"`
	ExpectedRepoVersion string `yaml:"expected_repo_version,omitempty" json:"expected_repo_version,omitempty"`
	Notes               string `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// OffHostRecoveryResult captures the typed drill result and provenance.
type OffHostRecoveryResult struct {
	BackupFile       string              `json:"backup_file"`
	BackupSourceURI  string              `json:"backup_source_uri"`
	BackupSHA256     string              `json:"backup_sha256"`
	BackupSizeBytes  int64               `json:"backup_size_bytes"`
	InventoryPath    string              `json:"inventory_path"`
	SecretsEnvFile   string              `json:"secrets_env_file"`
	RepoVersion      string              `json:"repo_version"`
	LocalBackupDir   string              `json:"local_backup_dir"`
	UsedOffHostInput bool                `json:"used_off_host_input"`
	ScratchDatabase  string              `json:"scratch_database"`
	Verification     RestoreVerification `json:"verification"`
}

// LoadOffHostRecoveryContract reads one YAML recovery contract from disk.
func LoadOffHostRecoveryContract(path string) (OffHostRecoveryContract, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OffHostRecoveryContract{}, nil, fmt.Errorf("read off-host recovery contract: %w", err)
	}

	var contract OffHostRecoveryContract
	if err := yaml.Unmarshal(data, &contract); err != nil {
		return OffHostRecoveryContract{}, nil, fmt.Errorf("parse off-host recovery contract: %w", err)
	}
	return normalizeOffHostContractFields(contract), data, nil
}

// NormalizeOffHostRecoveryContract resolves repo-relative inventory paths.
func NormalizeOffHostRecoveryContract(repoRoot string, contract OffHostRecoveryContract) OffHostRecoveryContract {
	contract = normalizeOffHostContractFields(contract)
	if contract.InventoryPath != "" && !filepath.IsAbs(contract.InventoryPath) {
		contract.InventoryPath = filepath.Join(repoRoot, filepath.Clean(contract.InventoryPath))
	}
	return normalizeOffHostContractFields(contract)
}

// ValidateOffHostRecoveryContract enforces the staged off-host recovery contract.
func ValidateOffHostRecoveryContract(repoRoot string, contract OffHostRecoveryContract) error {
	if strings.TrimSpace(repoRoot) == "" {
		return fmt.Errorf("repo root is required")
	}
	if contract.BackupFile == "" {
		return fmt.Errorf("backup_file is required")
	}
	if !filepath.IsAbs(contract.BackupFile) {
		return fmt.Errorf("backup_file must be an absolute staged path on the replacement host")
	}
	if !strings.HasSuffix(contract.BackupFile, ".sql.gz") {
		return fmt.Errorf("backup_file must end with .sql.gz")
	}
	if contract.BackupSourceURI == "" {
		return fmt.Errorf("backup_source_uri is required")
	}
	if !sha256HexPattern.MatchString(contract.BackupSHA256) {
		return fmt.Errorf("backup_sha256 must be a 64-character hex digest")
	}
	if contract.InventoryPath == "" {
		return fmt.Errorf("inventory_path is required")
	}
	if contract.SecretsEnvFile == "" {
		return fmt.Errorf("secrets_env_file is required")
	}
	if !filepath.IsAbs(contract.SecretsEnvFile) {
		return fmt.Errorf("secrets_env_file must be an absolute path")
	}

	localBackupDir := filepath.Join(repoRoot, "demo", "backups")
	insideLocalBackupDir, err := pathWithinBase(contract.BackupFile, localBackupDir)
	if err != nil {
		return err
	}
	if insideLocalBackupDir {
		return fmt.Errorf("backup_file must reference a staged off-host copy outside %s", localBackupDir)
	}

	if info, err := os.Stat(contract.BackupFile); err != nil {
		return fmt.Errorf("stat backup_file: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("backup_file is not a regular file: %s", contract.BackupFile)
	}
	if info, err := os.Stat(contract.InventoryPath); err != nil {
		return fmt.Errorf("stat inventory_path: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("inventory_path is not a regular file: %s", contract.InventoryPath)
	}
	if info, err := os.Stat(contract.SecretsEnvFile); err != nil {
		return fmt.Errorf("stat secrets_env_file: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("secrets_env_file is not a regular file: %s", contract.SecretsEnvFile)
	}

	if contract.ExpectedRepoVersion != "" {
		repoVersion, err := readTrackedRepoVersion(repoRoot)
		if err != nil {
			return err
		}
		if repoVersion != contract.ExpectedRepoVersion {
			return fmt.Errorf("expected_repo_version mismatch: contract=%q repo=%q", contract.ExpectedRepoVersion, repoVersion)
		}
	}

	return nil
}

// RunOffHostRecoveryDrill validates a staged off-host backup copy through a scratch restore.
func (s *AdminService) RunOffHostRecoveryDrill(ctx context.Context, repoRoot string, contract OffHostRecoveryContract, now time.Time) (OffHostRecoveryResult, error) {
	contract = NormalizeOffHostRecoveryContract(repoRoot, contract)
	if err := ValidateOffHostRecoveryContract(repoRoot, contract); err != nil {
		return OffHostRecoveryResult{}, err
	}

	sqlText, digest, sizeBytes, err := readCompressedBackup(contract.BackupFile)
	if err != nil {
		return OffHostRecoveryResult{}, err
	}
	if !strings.EqualFold(digest, contract.BackupSHA256) {
		return OffHostRecoveryResult{}, fmt.Errorf("backup sha256 mismatch: expected %s got %s", strings.ToLower(contract.BackupSHA256), strings.ToLower(digest))
	}

	scratchDatabase := fmt.Sprintf("acp_offhost_drill_%s", now.UTC().Format("20060102_150405"))
	rewrittenSQL, err := s.RewriteBackupForScratchDatabase(sqlText, scratchDatabase)
	if err != nil {
		return OffHostRecoveryResult{}, fmt.Errorf("prepare scratch restore SQL: %w", err)
	}

	_ = dropScratchDatabase(s, scratchDatabase)
	if err := s.Restore(ctx, strings.NewReader(rewrittenSQL)); err != nil {
		_ = dropScratchDatabase(s, scratchDatabase)
		return OffHostRecoveryResult{}, err
	}

	verification, verifyErr := s.VerifyCoreSchema(ctx, scratchDatabase)
	cleanupErr := dropScratchDatabase(s, scratchDatabase)
	if verifyErr != nil {
		return OffHostRecoveryResult{}, fmt.Errorf("verify restored scratch schema: %w", verifyErr)
	}
	if cleanupErr != nil {
		return OffHostRecoveryResult{}, fmt.Errorf("cleanup scratch database: %w", cleanupErr)
	}
	if verification.FoundTables != verification.ExpectedTables {
		return OffHostRecoveryResult{}, fmt.Errorf("restore verification failed: expected %d core tables, found %d", verification.ExpectedTables, verification.FoundTables)
	}

	repoVersion, err := readTrackedRepoVersion(repoRoot)
	if err != nil {
		return OffHostRecoveryResult{}, err
	}

	return OffHostRecoveryResult{
		BackupFile:       contract.BackupFile,
		BackupSourceURI:  contract.BackupSourceURI,
		BackupSHA256:     strings.ToLower(digest),
		BackupSizeBytes:  sizeBytes,
		InventoryPath:    contract.InventoryPath,
		SecretsEnvFile:   contract.SecretsEnvFile,
		RepoVersion:      repoVersion,
		LocalBackupDir:   filepath.Join(repoRoot, "demo", "backups"),
		UsedOffHostInput: true,
		ScratchDatabase:  scratchDatabase,
		Verification:     verification,
	}, nil
}

func normalizeOffHostContractFields(contract OffHostRecoveryContract) OffHostRecoveryContract {
	contract.BackupFile = strings.TrimSpace(contract.BackupFile)
	contract.BackupSourceURI = strings.TrimSpace(contract.BackupSourceURI)
	contract.BackupSHA256 = strings.TrimSpace(contract.BackupSHA256)
	contract.InventoryPath = strings.TrimSpace(contract.InventoryPath)
	contract.SecretsEnvFile = strings.TrimSpace(contract.SecretsEnvFile)
	contract.ExpectedRepoVersion = strings.TrimSpace(contract.ExpectedRepoVersion)
	contract.Notes = strings.TrimSpace(contract.Notes)
	if contract.BackupFile != "" {
		contract.BackupFile = filepath.Clean(contract.BackupFile)
	}
	if contract.InventoryPath != "" {
		contract.InventoryPath = filepath.Clean(contract.InventoryPath)
	}
	if contract.SecretsEnvFile != "" {
		contract.SecretsEnvFile = filepath.Clean(contract.SecretsEnvFile)
	}
	return contract
}

func readCompressedBackup(path string) (sqlText string, digest string, sizeBytes int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", 0, fmt.Errorf("open compressed backup: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", "", 0, fmt.Errorf("stat compressed backup: %w", err)
	}

	hasher := sha256.New()
	tee := io.TeeReader(file, hasher)
	reader, err := gzip.NewReader(tee)
	if err != nil {
		return "", "", 0, fmt.Errorf("open gzip backup: %w", err)
	}
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	if err != nil {
		return "", "", 0, fmt.Errorf("read gzip backup: %w", err)
	}

	return string(payload), hex.EncodeToString(hasher.Sum(nil)), info.Size(), nil
}

func readTrackedRepoVersion(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read tracked VERSION: %w", err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("tracked VERSION file is empty")
	}
	return version, nil
}

func pathWithinBase(path string, base string) (bool, error) {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("resolve path %s: %w", path, err)
	}
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false, fmt.Errorf("resolve base %s: %w", base, err)
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return false, fmt.Errorf("compare %s against %s: %w", pathAbs, baseAbs, err)
	}
	if rel == "." {
		return true, nil
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func dropScratchDatabase(admin *AdminService, databaseName string) error {
	if admin == nil || admin.connector == nil {
		return fmt.Errorf("database admin service requires a connector")
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return admin.DropDatabaseIfExists(cleanupCtx, databaseName)
}
