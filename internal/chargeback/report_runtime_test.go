// report_runtime_test.go - Tests for typed chargeback report generation.
//
// Purpose:
//   - Verify the typed report workflow replaces the legacy shell orchestration.
//
// Responsibilities:
//   - Cover report rendering, archival, analytics, and notification outcomes.
//
// Scope:
//   - Chargeback report workflow behavior only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Tests remain deterministic through fake stores and fixed clocks.
package chargeback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	currentCostCenters  []CostCenterAllocation
	previousCostCenters []CostCenterAllocation
	models              []ModelAllocation
	principals          []PrincipalSpend
	totalSpend          float64
	previousSpend       float64
	metrics             Metrics
	history             []HistoricalSpend
	totalBudget         float64
}

func (f fakeStore) CostCenterAllocations(_ context.Context, monthStart string, _ string) ([]CostCenterAllocation, error) {
	if strings.HasPrefix(monthStart, "2026-01") {
		return f.currentCostCenters, nil
	}
	return f.previousCostCenters, nil
}

func (f fakeStore) ModelAllocations(context.Context, string, string) ([]ModelAllocation, error) {
	return f.models, nil
}

func (f fakeStore) TopPrincipals(context.Context, string, string, int) ([]PrincipalSpend, error) {
	return f.principals, nil
}

func (f fakeStore) TotalSpend(_ context.Context, monthStart string, _ string) (float64, error) {
	if strings.HasPrefix(monthStart, "2026-01") {
		return f.totalSpend, nil
	}
	return f.previousSpend, nil
}

func (f fakeStore) Metrics(context.Context, string, string) (Metrics, error) {
	return f.metrics, nil
}

func (f fakeStore) HistoricalSpend(context.Context, int, string) ([]HistoricalSpend, error) {
	return f.history, nil
}

func (f fakeStore) TotalBudget(context.Context) (float64, error) {
	return f.totalBudget, nil
}

func TestGenerateReportBuildsOutputsAndArchives(t *testing.T) {
	t.Parallel()

	store := fakeStore{
		currentCostCenters: []CostCenterAllocation{
			{CostCenter: "1001", Team: "platform", RequestCount: 10, TokenCount: 1000, SpendAmount: 50, PercentOfTotal: 83.33},
			{CostCenter: "unknown-cc", Team: "unknown-team", RequestCount: 2, TokenCount: 200, SpendAmount: 10, PercentOfTotal: 16.67},
		},
		previousCostCenters: []CostCenterAllocation{
			{CostCenter: "1001", Team: "platform", RequestCount: 8, TokenCount: 800, SpendAmount: 20, PercentOfTotal: 100},
		},
		models:        []ModelAllocation{{Model: "gpt-4o-mini", RequestCount: 12, TokenCount: 1200, SpendAmount: 60}},
		principals:    []PrincipalSpend{{Principal: "team-platform__cc-1001", Team: "platform", CostCenter: "1001", RequestCount: 10, SpendAmount: 50}},
		totalSpend:    60,
		previousSpend: 20,
		metrics:       Metrics{TotalRequests: 12, TotalTokens: 1200},
		history: []HistoricalSpend{
			{Month: "2025-12", Spend: 25},
			{Month: "2025-11", Spend: 20},
			{Month: "2025-10", Spend: 15},
		},
		totalBudget: 100,
	}

	repoRoot := t.TempDir()
	result, err := GenerateReport(context.Background(), store, ReportOptions{
		Format:           "all",
		RepoRoot:         repoRoot,
		ArchiveDir:       "demo/backups/chargeback",
		ForecastEnabled:  true,
		AnomalyThreshold: 100,
		Now: func() time.Time {
			return time.Date(2026, time.February, 7, 12, 0, 0, 0, time.FixedZone("MST", -7*60*60))
		},
	})
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}

	if !strings.Contains(result.Outputs.Markdown, "# Financial Chargeback Report") {
		t.Fatalf("expected markdown report, got %q", result.Outputs.Markdown)
	}
	if !strings.Contains(result.Outputs.CSV, "CostCenter,Team,SpendAmount") {
		t.Fatalf("expected csv header, got %q", result.Outputs.CSV)
	}
	if !strings.Contains(result.Outputs.JSON, "\"schema_version\"") {
		t.Fatalf("expected json payload, got %q", result.Outputs.JSON)
	}
	if !result.Data.VarianceExceeded {
		t.Fatal("expected variance threshold exceeded")
	}
	if !result.Data.HasAnomalies {
		t.Fatal("expected anomalies")
	}

	for extension := range map[string]struct{}{"md": {}, "json": {}, "csv": {}} {
		path := result.Outputs.Archived[extension]
		if path == "" {
			t.Fatalf("expected archived path for %s", extension)
		}
		if !strings.HasPrefix(path, filepath.Join(repoRoot, "demo", "backups", "chargeback")) {
			t.Fatalf("unexpected archive path %s", path)
		}
	}
}

func TestGenerateReportSendsNotifications(t *testing.T) {
	t.Parallel()

	var genericSeen bool
	var slackSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/generic":
			genericSeen = true
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode generic payload: %v", err)
			}
			if payload["event"] != defaultGenericNotificationEvent {
				t.Fatalf("unexpected generic event: %#v", payload["event"])
			}
		case "/slack":
			slackSeen = true
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := fakeStore{
		currentCostCenters:  []CostCenterAllocation{{CostCenter: "1001", Team: "platform", RequestCount: 10, TokenCount: 1000, SpendAmount: 50, PercentOfTotal: 100}},
		previousCostCenters: []CostCenterAllocation{{CostCenter: "1001", Team: "platform", RequestCount: 10, TokenCount: 1000, SpendAmount: 50, PercentOfTotal: 100}},
		models:              []ModelAllocation{{Model: "gpt-4o-mini", RequestCount: 10, TokenCount: 1000, SpendAmount: 50}},
		principals:          []PrincipalSpend{{Principal: "platform", Team: "platform", CostCenter: "1001", RequestCount: 10, SpendAmount: 50}},
		totalSpend:          50,
		previousSpend:       50,
		metrics:             Metrics{TotalRequests: 10, TotalTokens: 1000},
	}

	_, err := GenerateReport(context.Background(), store, ReportOptions{
		Notify:            true,
		GenericWebhookURL: server.URL + "/generic",
		SlackWebhookURL:   server.URL + "/slack",
		RepoRoot:          t.TempDir(),
		Now: func() time.Time {
			return time.Date(2026, time.February, 7, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}
	if !genericSeen || !slackSeen {
		t.Fatalf("expected both notification endpoints, generic=%t slack=%t", genericSeen, slackSeen)
	}
}
