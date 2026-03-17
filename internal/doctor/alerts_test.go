// alerts_test.go - Coverage for doctor alert adapters.
//
// Purpose:
//   - Verify actionable doctor findings fan out through configured adapters.
//
// Responsibilities:
//   - Cover generic and Slack webhook delivery.
//   - Ensure only budget and detection findings are emitted.
//
// Scope:
//   - Doctor alert payload and delivery behavior only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use local httptest servers instead of external webhooks.
package doctor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestAlertAdaptersHelpersAndErrorPaths(t *testing.T) {
	if (GenericWebhookAdapter{}).Name() != "generic" {
		t.Fatal("expected generic adapter name")
	}
	if (SlackWebhookAdapter{}).Name() != "slack" {
		t.Fatal("expected slack adapter name")
	}
	if (GenericWebhookAdapter{}).client() == nil || (SlackWebhookAdapter{}).client() == nil {
		t.Fatal("expected default clients")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	payload := AlertPayload{Event: "doctor_findings", Source: "acpctl doctor", Overall: status.HealthLevelWarning, Timestamp: "2026-03-07T18:00:00Z"}
	if err := (GenericWebhookAdapter{URL: server.URL}).Send(context.Background(), payload); err == nil {
		t.Fatal("expected generic adapter send error")
	}
	if err := (SlackWebhookAdapter{URL: server.URL}).Send(context.Background(), payload); err == nil {
		t.Fatal("expected slack adapter send error")
	}
	if err := postDoctorJSON(context.Background(), server.Client(), "://bad-url", []byte(`{}`)); err == nil {
		t.Fatal("expected bad URL error")
	}

	noActionable := BuildAlertPayload(Report{Results: []CheckResult{{ID: "gateway_healthy", Level: status.HealthLevelHealthy}}})
	if len(noActionable.Findings) != 0 {
		t.Fatalf("expected no actionable findings, got %+v", noActionable.Findings)
	}
	if err := NotifyActionableFindings(context.Background(), config.AlertSettings{}, Report{}); err != nil {
		t.Fatalf("expected empty config to no-op, got %v", err)
	}
}

func TestNotifyActionableFindingsSendsBudgetAndDetectionAlerts(t *testing.T) {
	var genericSeen bool
	var slackSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/generic":
			genericSeen = true
		case "/slack":
			slackSeen = true
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	report := Report{
		Overall:   status.HealthLevelWarning,
		Timestamp: "2026-03-07T18:00:00Z",
		Results: []CheckResult{
			{ID: "budget_findings", Name: "Budget Findings", Level: status.HealthLevelWarning, Severity: SeverityDomain, Message: "1 budgets, 1 >80% utilized"},
			{ID: "detections_findings", Name: "Security Findings", Level: status.HealthLevelUnhealthy, Severity: SeverityDomain, Message: "2 high-severity findings in last 24h"},
			{ID: "gateway_healthy", Name: "Gateway Healthy", Level: status.HealthLevelHealthy, Severity: SeverityDomain, Message: "ok"},
		},
	}

	err := NotifyActionableFindings(context.Background(), config.AlertSettings{
		GenericWebhookURL: server.URL + "/generic",
		SlackWebhookURL:   server.URL + "/slack",
	}, report)
	if err != nil {
		t.Fatalf("NotifyActionableFindings() error = %v", err)
	}
	if !genericSeen || !slackSeen {
		t.Fatalf("expected both adapters, generic=%t slack=%t", genericSeen, slackSeen)
	}
}
