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
	"testing"

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

func TestGatewayCollectorCoversWarningAndHealthyBranches(t *testing.T) {
	t.Parallel()

	warning := NewGatewayCollector(fakeGatewayReader{status: gateway.Status{
		Scheme: "http", BaseURL: "http://127.0.0.1:4000",
		Health: gateway.Probe{Path: "/health", Reachable: true, Authorized: true, Healthy: true, HTTPStatus: http.StatusOK},
		Models: gateway.Probe{Path: "/v1/models", Reachable: true, Authorized: false, Healthy: false, HTTPStatus: http.StatusUnauthorized},
	}})
	warningStatus := warning.Collect(context.Background())
	if warningStatus.Level != status.HealthLevelWarning || warningStatus.Details.ModelsHTTPStatus != http.StatusUnauthorized {
		t.Fatalf("unexpected warning gateway status: %+v", warningStatus)
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
