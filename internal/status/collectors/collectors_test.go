// collectors_test.go - Tests for typed status collectors.
//
// Purpose:
//   - Verify collector branch behavior against deterministic fake services.
//
// Responsibilities:
//   - Cover gateway, database, key, budget, and detection collector branches.
//   - Replace environment-coupled collector tests with fake readers.
//   - Lock down operator-facing messages and health levels for CI stability.
//
// Scope:
//   - Collector unit tests only.
//
// Usage:
//   - Run with `go test ./internal/status/collectors`.
//
// Invariants/Assumptions:
//   - Tests do not require live PostgreSQL, Docker, or gateway runtime services.
package collectors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/certlifecycle"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

type fakeGatewayReader struct {
	status gateway.Status
}

func (f fakeGatewayReader) Status(context.Context) gateway.Status {
	return f.status
}

type fakeRuntimeReader struct {
	summary      db.Summary
	summaryErr   error
	configErr    error
	summaryCalls int
}

func (f *fakeRuntimeReader) Summary(context.Context) (db.Summary, error) {
	f.summaryCalls++
	return f.summary, f.summaryErr
}

func (f *fakeRuntimeReader) ConfigError() error {
	return f.configErr
}

type fakeReadonlyReader struct {
	keySummary       db.KeySummary
	keyErr           error
	budgetSummary    db.BudgetSummary
	budgetErr        error
	detectionSummary db.DetectionSummary
	detectionErr     error
	trafficSummary   db.TrafficSummary
	trafficErr       error
}

func (f fakeReadonlyReader) KeySummary(context.Context) (db.KeySummary, error) {
	return f.keySummary, f.keyErr
}

func (f fakeReadonlyReader) BudgetSummary(context.Context) (db.BudgetSummary, error) {
	return f.budgetSummary, f.budgetErr
}

func (f fakeReadonlyReader) DetectionSummary(context.Context) (db.DetectionSummary, error) {
	return f.detectionSummary, f.detectionErr
}

func (f fakeReadonlyReader) TrafficSummary(context.Context) (db.TrafficSummary, error) {
	return f.trafficSummary, f.trafficErr
}

func TestCertificateCollectorCoversInactiveWarningAndHealthyBranches(t *testing.T) {
	t.Parallel()

	inactiveRepo := t.TempDir()
	testutil.WriteRepoFile(t, inactiveRepo, "demo/.env", "ACP_GATEWAY_URL=http://127.0.0.1:4000\n")
	inactive := NewCertificateCollector(inactiveRepo).Collect(context.Background())
	if inactive.Level != status.HealthLevelHealthy || inactive.Message != "TLS overlay inactive; certificate lifecycle not applicable" {
		t.Fatalf("unexpected inactive certificate status: %+v", inactive)
	}

	warningRepo := t.TempDir()
	testutil.WriteRepoFile(t, warningRepo, "demo/.env", "ACP_GATEWAY_URL=https://gateway.example.com\n")
	originalCheck := certificateCheck
	originalStore := newCertificateStore
	certificateCheck = func(context.Context, certlifecycle.Store, certlifecycle.CheckRequest) (certlifecycle.CheckResult, error) {
		return certlifecycle.CheckResult{
			CheckedAt: time.Now().UTC(),
			Status:    certlifecycle.StatusWarning,
			Message:   "Certificate expires in 10 day(s)",
			Certificates: []certlifecycle.CertificateInfo{{
				DNSNames:          []string{"gateway.example.com"},
				NotAfter:          time.Now().UTC().Add(10 * 24 * time.Hour),
				Issuer:            "Issuer",
				Subject:           "gateway.example.com",
				SerialNumber:      "SERIAL",
				ManagedBy:         "lets-encrypt",
				StoragePath:       "/data/caddy/certificates/example/gateway.example.com.crt",
				FingerprintSHA256: "ABC123",
			}},
			Suggestions: []string{"Renew soon"},
		}, nil
	}
	newCertificateStore = func(string) certlifecycle.Store { return nil }
	defer func() {
		certificateCheck = originalCheck
		newCertificateStore = originalStore
	}()

	warning := NewCertificateCollector(warningRepo).Collect(context.Background())
	if warning.Level != status.HealthLevelWarning || warning.Details.Domain != "gateway.example.com" || warning.Details.DaysRemaining == 0 {
		t.Fatalf("unexpected warning certificate status: %+v", warning)
	}
	if len(warning.Suggestions) == 0 {
		t.Fatalf("expected renewal suggestion, got %+v", warning)
	}

	certificateCheck = func(context.Context, certlifecycle.Store, certlifecycle.CheckRequest) (certlifecycle.CheckResult, error) {
		return certlifecycle.CheckResult{
			CheckedAt: time.Now().UTC(),
			Status:    certlifecycle.StatusHealthy,
			Message:   "Certificate valid for 90 day(s)",
			Certificates: []certlifecycle.CertificateInfo{{
				DNSNames:          []string{"gateway.example.com"},
				NotAfter:          time.Now().UTC().Add(90 * 24 * time.Hour),
				Issuer:            "Issuer",
				Subject:           "gateway.example.com",
				SerialNumber:      "SERIAL",
				ManagedBy:         "lets-encrypt",
				StoragePath:       "/data/caddy/certificates/example/gateway.example.com.crt",
				FingerprintSHA256: "ABC123",
			}},
		}, nil
	}
	healthy := NewCertificateCollector(warningRepo).Collect(context.Background())
	if healthy.Level != status.HealthLevelHealthy || healthy.Details.Domain != "gateway.example.com" {
		t.Fatalf("unexpected healthy certificate status: %+v", healthy)
	}
}

func TestGatewayCollectorCoversWarningAndHealthyBranches(t *testing.T) {
	t.Parallel()

	missingMasterKey := NewGatewayCollector(fakeGatewayReader{status: gateway.Status{
		Scheme:              "http",
		BaseURL:             "http://127.0.0.1:4000",
		MasterKeyConfigured: false,
	}}).Collect(context.Background())
	if missingMasterKey.Level != status.HealthLevelWarning || !strings.Contains(missingMasterKey.Message, "LITELLM_MASTER_KEY not set") {
		t.Fatalf("unexpected missing master key gateway status: %+v", missingMasterKey)
	}

	warning := NewGatewayCollector(fakeGatewayReader{status: gateway.Status{
		Scheme:              "http",
		BaseURL:             "http://127.0.0.1:4000",
		MasterKeyConfigured: true,
		Health:              gateway.Probe{Path: "/health", Reachable: true, Authorized: true, Healthy: true, HTTPStatus: http.StatusOK},
		Models:              gateway.Probe{Path: "/v1/models", Reachable: true, Authorized: false, Healthy: false, HTTPStatus: http.StatusUnauthorized},
	}})
	warningStatus := warning.Collect(context.Background())
	if warningStatus.Level != status.HealthLevelWarning || warningStatus.Details.ModelsHTTPStatus != http.StatusUnauthorized {
		t.Fatalf("unexpected warning gateway status: %+v", warningStatus)
	}

	unreachable := NewGatewayCollector(fakeGatewayReader{status: gateway.Status{
		Scheme:              "http",
		BaseURL:             "http://127.0.0.1:4000",
		MasterKeyConfigured: true,
		Health:              gateway.Probe{Error: "connection refused"},
	}}).Collect(context.Background())
	if unreachable.Level != status.HealthLevelUnhealthy || unreachable.Details.Error != "connection refused" {
		t.Fatalf("unexpected unreachable gateway status: %+v", unreachable)
	}

	modelsError := NewGatewayCollector(fakeGatewayReader{status: gateway.Status{
		Scheme:              "http",
		BaseURL:             "http://127.0.0.1:4000",
		MasterKeyConfigured: true,
		Health:              gateway.Probe{Reachable: true, Authorized: true, Healthy: true, HTTPStatus: http.StatusOK},
		Models:              gateway.Probe{Error: "timeout", Reachable: false, Authorized: false},
	}}).Collect(context.Background())
	if modelsError.Level != status.HealthLevelWarning || modelsError.Details.Error != "timeout" {
		t.Fatalf("unexpected models-error gateway status: %+v", modelsError)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	host, port := testutil.HostPortFromURL(t, server.URL)
	healthyStatus := NewGatewayCollector(gateway.NewClient(
		gateway.WithHost(host),
		gateway.WithPort(port),
		gateway.WithMasterKey("sk-master"),
	)).Collect(context.Background())
	if healthyStatus.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy gateway status, got %+v", healthyStatus)
	}
}

func TestDatabaseCollectorCoversConfigErrorRuntimeErrorSchemaWarningAndHealthy(t *testing.T) {
	t.Parallel()

	configReader := &fakeRuntimeReader{configErr: errors.New("ambiguous")}
	configStatus := NewDatabaseCollector(configReader).Collect(context.Background())
	if configStatus.Level != status.HealthLevelUnhealthy || configStatus.Message != "Database configuration is ambiguous" {
		t.Fatalf("unexpected config error status: %+v", configStatus)
	}
	if configStatus.Details.LookupError != status.LookupErrorDatabaseConfigAmbiguous {
		t.Fatalf("expected lookup error %q, got %+v", status.LookupErrorDatabaseConfigAmbiguous, configStatus.Details)
	}
	if configReader.summaryCalls != 0 {
		t.Fatalf("expected config error to short-circuit summary collection, got %d summary call(s)", configReader.summaryCalls)
	}

	errorStatus := NewDatabaseCollector(&fakeRuntimeReader{
		summary:    db.Summary{Mode: config.DatabaseModeExternal, Ping: db.Probe{Error: "connection refused"}},
		summaryErr: errors.New("connection refused"),
	}).Collect(context.Background())
	if errorStatus.Level != status.HealthLevelUnhealthy || errorStatus.Details.Error != "connection refused" {
		t.Fatalf("unexpected runtime error status: %+v", errorStatus)
	}

	warningStatus := NewDatabaseCollector(&fakeRuntimeReader{
		summary: db.Summary{Mode: config.DatabaseModeEmbedded, ExpectedTables: 2, DatabaseName: "litellm"},
	}).Collect(context.Background())
	if warningStatus.Level != status.HealthLevelWarning {
		t.Fatalf("expected schema warning, got %+v", warningStatus)
	}

	healthyStatus := NewDatabaseCollector(&fakeRuntimeReader{
		summary: db.Summary{Mode: config.DatabaseModeEmbedded, ExpectedTables: 4, DatabaseName: "litellm", DatabaseUser: "litellm"},
	}).Collect(context.Background())
	if healthyStatus.Level != status.HealthLevelHealthy || healthyStatus.Details.DatabaseName != "litellm" {
		t.Fatalf("unexpected healthy status: %+v", healthyStatus)
	}
}

func TestBackupCollectorCoversMissingFreshAndStaleBranches(t *testing.T) {
	t.Parallel()

	missing := NewBackupCollector(t.TempDir()).Collect(context.Background())
	if missing.Level != status.HealthLevelWarning || missing.Message != "No database backups found" {
		t.Fatalf("unexpected missing backup status: %+v", missing)
	}

	repoRoot := t.TempDir()
	backupDir := filepath.Join(repoRoot, "demo", "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	freshPath := filepath.Join(backupDir, "litellm-backup-fresh.sql.gz")
	if err := os.WriteFile(freshPath, []byte("fresh"), 0o600); err != nil {
		t.Fatalf("WriteFile(fresh) error = %v", err)
	}
	freshTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(freshPath, freshTime, freshTime); err != nil {
		t.Fatalf("Chtimes(fresh) error = %v", err)
	}

	fresh := NewBackupCollector(repoRoot).Collect(context.Background())
	if fresh.Level != status.HealthLevelHealthy || fresh.Details.BackupPath != freshPath {
		t.Fatalf("unexpected fresh backup status: %+v", fresh)
	}

	if err := os.Remove(freshPath); err != nil {
		t.Fatalf("Remove(fresh) error = %v", err)
	}
	stalePath := filepath.Join(backupDir, "litellm-backup-stale.sql.gz")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("WriteFile(stale) error = %v", err)
	}
	staleTime := time.Now().Add(-(backupCriticalAge + time.Hour))
	if err := os.Chtimes(stalePath, staleTime, staleTime); err != nil {
		t.Fatalf("Chtimes(stale) error = %v", err)
	}

	stale := NewBackupCollector(repoRoot).Collect(context.Background())
	if stale.Level != status.HealthLevelUnhealthy || stale.Details.BackupPath != stalePath {
		t.Fatalf("unexpected stale backup status: %+v", stale)
	}
}

func TestReadinessCollectorCoversMissingFailedAndHealthyBranches(t *testing.T) {
	t.Parallel()

	missing := NewReadinessCollector(t.TempDir()).Collect(context.Background())
	if missing.Level != status.HealthLevelWarning || missing.Message != "No readiness evidence run found" {
		t.Fatalf("unexpected missing readiness status: %+v", missing)
	}

	repoRoot := t.TempDir()
	evidenceRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	runDir := filepath.Join(evidenceRoot, "readiness-fixture")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(runDir) error = %v", err)
	}
	for _, name := range []string{"readiness-summary.md", "presentation-readiness-tracker.md", "go-no-go-decision.md"} {
		if err := os.WriteFile(filepath.Join(runDir, name), []byte("fixture\n"), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	failedSummary := `{
  "run_id": "readiness-fixture",
  "generated_at_utc": "2026-03-18T00:00:00Z",
  "run_directory": "` + runDir + `",
  "overall_status": "FAIL",
  "failing_gate_count": 1,
  "skipped_gate_count": 0,
  "gate_results": []
}
`
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), []byte(failedSummary), 0o600); err != nil {
		t.Fatalf("WriteFile(summary.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "evidence-inventory.txt"), []byte("evidence-inventory.txt\ngo-no-go-decision.md\npresentation-readiness-tracker.md\nreadiness-summary.md\nsummary.json\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(inventory) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "latest-run.txt"), []byte(runDir+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(latest-run.txt) error = %v", err)
	}

	failed := NewReadinessCollector(repoRoot).Collect(context.Background())
	if failed.Level != status.HealthLevelUnhealthy || failed.Details.ReadinessOverallStatus != "FAIL" {
		t.Fatalf("unexpected failed readiness status: %+v", failed)
	}

	bundlePath := filepath.Join(repoRoot, "bundle.tar.gz")
	checksumPath := bundlePath + ".sha256"
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o600); err != nil {
		t.Fatalf("WriteFile(bundle) error = %v", err)
	}
	if err := os.WriteFile(checksumPath, []byte("checksum"), 0o600); err != nil {
		t.Fatalf("WriteFile(checksum) error = %v", err)
	}
	freshSummary := `{
  "run_id": "readiness-fixture",
  "generated_at_utc": "` + time.Now().UTC().Format(time.RFC3339) + `",
  "run_directory": "` + runDir + `",
  "bundle_path": "` + bundlePath + `",
  "bundle_checksum_path": "` + checksumPath + `",
  "overall_status": "PASS",
  "failing_gate_count": 0,
  "skipped_gate_count": 1,
  "gate_results": []
}
`
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), []byte(freshSummary), 0o600); err != nil {
		t.Fatalf("WriteFile(summary.json fresh) error = %v", err)
	}

	healthy := NewReadinessCollector(repoRoot).Collect(context.Background())
	if healthy.Level != status.HealthLevelHealthy || healthy.Details.ReadinessOverallStatus != "PASS" {
		t.Fatalf("unexpected healthy readiness status: %+v", healthy)
	}

	staleSummary := `{
  "run_id": "readiness-fixture",
  "generated_at_utc": "` + time.Now().UTC().Add(-(readinessStaleAge + time.Hour)).Format(time.RFC3339) + `",
  "run_directory": "` + runDir + `",
  "bundle_path": "` + bundlePath + `",
  "bundle_checksum_path": "` + checksumPath + `",
  "overall_status": "PASS",
  "failing_gate_count": 0,
  "skipped_gate_count": 0,
  "gate_results": []
}
`
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), []byte(staleSummary), 0o600); err != nil {
		t.Fatalf("WriteFile(summary.json stale) error = %v", err)
	}
	stale := NewReadinessCollector(repoRoot).Collect(context.Background())
	if stale.Level != status.HealthLevelWarning || !strings.Contains(stale.Message, "stale") {
		t.Fatalf("unexpected stale readiness status: %+v", stale)
	}

	invalidSummary := `{
  "run_id": "readiness-fixture",
  "generated_at_utc": "not-a-timestamp",
  "run_directory": "` + runDir + `",
  "bundle_path": "` + bundlePath + `",
  "bundle_checksum_path": "` + checksumPath + `",
  "overall_status": "PASS",
  "failing_gate_count": 0,
  "skipped_gate_count": 0,
  "gate_results": []
}
`
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), []byte(invalidSummary), 0o600); err != nil {
		t.Fatalf("WriteFile(summary.json invalid) error = %v", err)
	}
	invalid := NewReadinessCollector(repoRoot).Collect(context.Background())
	if invalid.Level != status.HealthLevelWarning || invalid.Details.Error == "" {
		t.Fatalf("unexpected invalid-timestamp readiness status: %+v", invalid)
	}
}

func TestKeysCollectorCoversWarningAndHealthyBranches(t *testing.T) {
	t.Parallel()

	errorStatus := NewKeysCollector(fakeReadonlyReader{keyErr: errors.New("missing table")}).Collect(context.Background())
	if errorStatus.Level != status.HealthLevelWarning || errorStatus.Details.Error != "missing table" {
		t.Fatalf("unexpected key error status: %+v", errorStatus)
	}

	emptyStatus := NewKeysCollector(fakeReadonlyReader{keySummary: db.KeySummary{Total: 0}}).Collect(context.Background())
	if emptyStatus.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning for no keys, got %+v", emptyStatus)
	}

	expiredStatus := NewKeysCollector(fakeReadonlyReader{keySummary: db.KeySummary{Total: 3, Active: 2, Expired: 1}}).Collect(context.Background())
	if expiredStatus.Level != status.HealthLevelWarning || expiredStatus.Details.ExpiredKeys != 1 {
		t.Fatalf("unexpected expired key status: %+v", expiredStatus)
	}

	healthyStatus := NewKeysCollector(fakeReadonlyReader{keySummary: db.KeySummary{Total: 2, Active: 2, Expired: 0}}).Collect(context.Background())
	if healthyStatus.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy key status, got %+v", healthyStatus)
	}
}

func TestBudgetCollectorCoversWarningUnhealthyAndHealthyBranches(t *testing.T) {
	t.Parallel()

	errorStatus := NewBudgetCollector(fakeReadonlyReader{budgetErr: errors.New("missing budgets table")}).Collect(context.Background())
	if errorStatus.Level != status.HealthLevelWarning || errorStatus.Details.Error != "missing budgets table" {
		t.Fatalf("unexpected budget error status: %+v", errorStatus)
	}

	emptyStatus := NewBudgetCollector(fakeReadonlyReader{budgetSummary: db.BudgetSummary{Total: 0}}).Collect(context.Background())
	if emptyStatus.Level != status.HealthLevelHealthy || emptyStatus.Message != "No budgets configured" {
		t.Fatalf("unexpected empty budget status: %+v", emptyStatus)
	}

	exhaustedStatus := NewBudgetCollector(fakeReadonlyReader{budgetSummary: db.BudgetSummary{Total: 4, Exhausted: 1}}).Collect(context.Background())
	if exhaustedStatus.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected exhausted budget to be unhealthy, got %+v", exhaustedStatus)
	}

	warningStatus := NewBudgetCollector(fakeReadonlyReader{budgetSummary: db.BudgetSummary{Total: 4, HighUtilization: 2}}).Collect(context.Background())
	if warningStatus.Level != status.HealthLevelWarning {
		t.Fatalf("expected high utilization warning, got %+v", warningStatus)
	}

	healthyStatus := NewBudgetCollector(fakeReadonlyReader{budgetSummary: db.BudgetSummary{Total: 4}}).Collect(context.Background())
	if healthyStatus.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy budget status, got %+v", healthyStatus)
	}
}

func TestTrafficCollectorCoversWarningAndHealthyBranches(t *testing.T) {
	t.Parallel()

	errorStatus := NewTrafficCollector(fakeReadonlyReader{trafficErr: errors.New("missing spend logs")}).Collect(context.Background())
	if errorStatus.Level != status.HealthLevelWarning || errorStatus.Details.Error != "missing spend logs" {
		t.Fatalf("unexpected traffic error status: %+v", errorStatus)
	}

	noData := NewTrafficCollector(fakeReadonlyReader{trafficSummary: db.TrafficSummary{SpendLogsTableExists: false}}).Collect(context.Background())
	if noData.Level != status.HealthLevelHealthy || noData.Message != "No gateway traffic data yet" {
		t.Fatalf("unexpected no-data traffic status: %+v", noData)
	}

	warning := NewTrafficCollector(fakeReadonlyReader{trafficSummary: db.TrafficSummary{
		SpendLogsTableExists: true,
		TotalRequests24h:     20,
		TotalTokens24h:       2000,
		TotalSpend24h:        4.5,
		ErrorRequests24h:     3,
	}}).Collect(context.Background())
	if warning.Level != status.HealthLevelWarning || warning.Details.ErrorRatePercent24h <= trafficErrorRateWarningThreshold {
		t.Fatalf("unexpected warning traffic status: %+v", warning)
	}

	noRequests := NewTrafficCollector(fakeReadonlyReader{trafficSummary: db.TrafficSummary{SpendLogsTableExists: true}}).Collect(context.Background())
	if noRequests.Level != status.HealthLevelHealthy || noRequests.Message != "No gateway traffic in last 24h" {
		t.Fatalf("unexpected no-requests traffic status: %+v", noRequests)
	}

	healthy := NewTrafficCollector(fakeReadonlyReader{trafficSummary: db.TrafficSummary{
		SpendLogsTableExists: true,
		TotalRequests24h:     5,
		TotalTokens24h:       500,
		TotalSpend24h:        1.25,
		ErrorRequests24h:     0,
	}}).Collect(context.Background())
	if healthy.Level != status.HealthLevelHealthy || healthy.Details.TotalRequests24h != 5 {
		t.Fatalf("unexpected healthy traffic status: %+v", healthy)
	}
}

func TestDetectionsCollectorCoversUnknownWarningAndHealthyBranches(t *testing.T) {
	repoRoot := t.TempDir()

	missingConfig := NewDetectionsCollector(repoRoot, fakeReadonlyReader{}).Collect(context.Background())
	if missingConfig.Level != status.HealthLevelUnknown {
		t.Fatalf("expected missing config to be unknown, got %+v", missingConfig)
	}

	testutil.WriteRepoFile(t, repoRoot, "demo/config/detection_rules.yaml", "detection_rules: []\n")

	errorStatus := NewDetectionsCollector(repoRoot, fakeReadonlyReader{detectionErr: errors.New("query failed")}).Collect(context.Background())
	if errorStatus.Level != status.HealthLevelUnknown || errorStatus.Details.Error != "query failed" {
		t.Fatalf("unexpected detection error status: %+v", errorStatus)
	}

	noLogs := NewDetectionsCollector(repoRoot, fakeReadonlyReader{detectionSummary: db.DetectionSummary{SpendLogsTableExists: false}}).Collect(context.Background())
	if noLogs.Level != status.HealthLevelHealthy || noLogs.Message != "No audit log data yet" {
		t.Fatalf("unexpected no logs status: %+v", noLogs)
	}

	highSeverity := NewDetectionsCollector(repoRoot, fakeReadonlyReader{detectionSummary: db.DetectionSummary{
		SpendLogsTableExists: true,
		HighSeverity:         2,
	}}).Collect(context.Background())
	if highSeverity.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected high severity findings to be unhealthy, got %+v", highSeverity)
	}

	mediumSeverity := NewDetectionsCollector(repoRoot, fakeReadonlyReader{detectionSummary: db.DetectionSummary{
		SpendLogsTableExists: true,
		MediumSeverity:       3,
	}}).Collect(context.Background())
	if mediumSeverity.Level != status.HealthLevelWarning {
		t.Fatalf("expected medium severity findings to be warning, got %+v", mediumSeverity)
	}

	healthy := NewDetectionsCollector(repoRoot, fakeReadonlyReader{detectionSummary: db.DetectionSummary{
		SpendLogsTableExists: true,
		UniqueModels24h:      4,
		TotalEntries24h:      120,
	}}).Collect(context.Background())
	if healthy.Level != status.HealthLevelHealthy || healthy.Details.TotalEntries24h != 120 {
		t.Fatalf("unexpected healthy detections status: %+v", healthy)
	}
}
