// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Verify chargeback notification payloads and delivery failures directly.
//
// Responsibilities:
//   - Cover generic and Slack webhook payload construction.
//   - Verify noop, success, and non-2xx notification flows.
//   - Lock down HTTP delivery behavior for CI-facing workflow tests.
//
// Scope:
//   - Unit tests for notification helpers only.
//
// Usage:
//   - Run with `go test ./internal/chargeback`.
//
// Invariants/Assumptions:
//   - Tests use httptest servers instead of real webhooks.
//   - Notification delivery stays opt-in and deterministic.
package chargeback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookNotifier_NoWebhookConfiguredIsNoop(t *testing.T) {
	t.Parallel()

	if err := (WebhookNotifier{}).Notify(context.Background(), NotificationConfig{}, ReportData{}); err != nil {
		t.Fatalf("expected noop notify, got %v", err)
	}
}

func TestBuildSlackWebhookPayload_ContainsExpectedFields(t *testing.T) {
	t.Parallel()

	payload, err := BuildSlackWebhookPayload(SlackWebhookInput{
		ReportMonth: "2026-02",
		TotalSpend:  42.5,
		Variance:    "12.5",
		Color:       "danger",
		Epoch:       12345,
	})
	if err != nil {
		t.Fatalf("BuildSlackWebhookPayload returned error: %v", err)
	}
	var decoded struct {
		Text        string `json:"text"`
		Attachments []struct {
			Color  string `json:"color"`
			Title  string `json:"title"`
			Footer string `json:"footer"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if decoded.Text == "" || len(decoded.Attachments) != 1 {
		t.Fatalf("unexpected slack payload: %+v", decoded)
	}
	if decoded.Attachments[0].Color != "danger" || !strings.Contains(decoded.Attachments[0].Title, "2026-02") {
		t.Fatalf("unexpected slack attachment: %+v", decoded.Attachments[0])
	}
}

func TestPostJSONAndWebhookNotifier_ReportFailures(t *testing.T) {
	t.Parallel()

	var genericSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/generic":
			genericSeen = true
			w.WriteHeader(http.StatusCreated)
		case "/slack":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data := ReportData{
		Range: MonthRange{GeneratedAtUTC: time.Date(2026, time.March, 9, 19, 0, 0, 0, time.UTC)},
		Input: ReportInput{
			ReportMonth: "2026-02",
			TotalSpend:  42.5,
			Variance:    "12.5",
			Anomalies:   []Anomaly{{CostCenter: "1001", Team: "platform", SpikePercent: 150}},
		},
		VarianceExceeded: true,
	}
	err := (WebhookNotifier{Client: server.Client()}).Notify(context.Background(), NotificationConfig{
		GenericWebhookURL: server.URL + "/generic",
		SlackWebhookURL:   server.URL + "/slack",
	}, data)
	if err == nil || !strings.Contains(err.Error(), "send slack notification: unexpected HTTP 500") {
		t.Fatalf("expected slack HTTP failure, got %v", err)
	}
	if !genericSeen {
		t.Fatal("expected generic webhook to be sent before slack failure")
	}

	if err := postJSON(context.Background(), server.Client(), "://bad-url", []byte(`{}`)); err == nil {
		t.Fatal("expected invalid URL to fail")
	}
}
