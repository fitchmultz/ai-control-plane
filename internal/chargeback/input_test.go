// input_test.go - Tests for chargeback typed input adapters.
//
// Purpose:
//   - Verify report, render, and payload inputs are normalized inside the
//     chargeback package instead of CLI adapters.
//
// Responsibilities:
//   - Cover command defaulting, env decoding, and invalid input handling.
//
// Scope:
//   - Chargeback input adapter behavior only.
//
// Usage:
//   - Used through package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Tests use a fake environment to keep defaults deterministic.
package chargeback

import (
	"strings"
	"testing"
	"time"
)

type fakeEnv struct {
	values        map[string]string
	int64Values   map[string]*int64
	float64Values map[string]*float64
	forecast      [3]*float64
}

func (f fakeEnv) String(key string) string {
	return f.values[key]
}

func (f fakeEnv) Int64Ptr(key string) *int64 {
	return f.int64Values[key]
}

func (f fakeEnv) Float64Ptr(key string) *float64 {
	return f.float64Values[key]
}

func (f fakeEnv) ChargebackForecast() (*float64, *float64, *float64) {
	return f.forecast[0], f.forecast[1], f.forecast[2]
}

func (f fakeEnv) ChargebackTimestamp(now time.Time) string {
	if value := f.values["CHARGEBACK_PAYLOAD_TIMESTAMP"]; value != "" {
		return value
	}
	return now.UTC().Format(time.RFC3339)
}

func TestNewReportWorkflowInputAppliesDefaultsAndEnvNotifications(t *testing.T) {
	t.Parallel()

	env := fakeEnv{
		values: map[string]string{
			"GENERIC_WEBHOOK_URL": "https://generic.example",
			"SLACK_WEBHOOK_URL":   "https://slack.example",
		},
	}
	input, err := NewReportWorkflowInput(ReportCommandInput{}, env, "/repo", func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewReportWorkflowInput returned error: %v", err)
	}

	if input.Request.Format != ReportFormatMarkdown {
		t.Fatalf("expected markdown default, got %s", input.Request.Format)
	}
	if input.Request.ArchiveDir != defaultArchiveDir {
		t.Fatalf("expected archive dir default, got %s", input.Request.ArchiveDir)
	}
	if !input.Request.ForecastEnabled {
		t.Fatal("expected forecast enabled default")
	}
	if input.Notification.GenericWebhookURL == "" || input.Notification.SlackWebhookURL == "" {
		t.Fatalf("expected webhook URLs, got %#v", input.Notification)
	}
}

func TestNewRenderRequestDecodesEnvironmentPayloads(t *testing.T) {
	t.Parallel()

	daysRemaining := int64(4)
	budgetPercent := 82.5
	forecast1 := 10.0
	env := fakeEnv{
		values: map[string]string{
			"CHARGEBACK_REPORT_MONTH":                   "2026-02",
			"CHARGEBACK_COST_CENTER_JSON":               `[{"cost_center":"1001","team":"platform","request_count":5,"token_count":50,"spend_amount":25.5,"percent_of_total":100}]`,
			"CHARGEBACK_MODEL_JSON":                     `[{"model":"gpt-4o-mini","request_count":5,"token_count":50,"spend_amount":25.5}]`,
			"CHARGEBACK_ANOMALIES_JSON":                 `[{"cost_center":"1001","team":"platform","current_spend":25.5,"previous_spend":10,"spike_percent":155,"type":"spike"}]`,
			"CHARGEBACK_TOTAL_SPEND":                    "25.5",
			"CHARGEBACK_TOTAL_REQUESTS":                 "5",
			"CHARGEBACK_TOTAL_TOKENS":                   "50",
			"CHARGEBACK_VARIANCE":                       "12.5",
			"CHARGEBACK_BUDGET_RISK_LEVEL":              "high",
			"CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED": "true",
			"CHARGEBACK_EXHAUSTION_DATE":                "2026-02-20",
			"CHARGEBACK_GENERATED_AT":                   "2026-03-08T18:00:00Z",
		},
		int64Values:   map[string]*int64{"CHARGEBACK_DAYS_REMAINING": &daysRemaining},
		float64Values: map[string]*float64{"CHARGEBACK_BUDGET_RISK_PERCENT": &budgetPercent},
		forecast:      [3]*float64{&forecast1, nil, nil},
	}

	request, err := NewRenderRequest(RenderCommandInput{Format: "json"}, env, nil)
	if err != nil {
		t.Fatalf("NewRenderRequest returned error: %v", err)
	}
	if request.Format != ReportFormatJSON {
		t.Fatalf("expected json format, got %s", request.Format)
	}
	if request.Input.TotalSpend != 25.5 || len(request.Input.CostCenters) != 1 || len(request.Input.Anomalies) != 1 {
		t.Fatalf("unexpected decoded render input: %#v", request.Input)
	}
	if request.Input.BudgetRisk.BudgetPercent == nil || *request.Input.BudgetRisk.BudgetPercent != budgetPercent {
		t.Fatalf("expected budget percent, got %#v", request.Input.BudgetRisk.BudgetPercent)
	}
}

func TestNewPayloadRequestRejectsInvalidAnomalyJSON(t *testing.T) {
	t.Parallel()

	_, err := NewPayloadRequest(PayloadCommandInput{Target: "generic"}, fakeEnv{
		values: map[string]string{
			"CHARGEBACK_ANOMALIES_JSON": "{not-json}",
		},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid CHARGEBACK_ANOMALIES_JSON") {
		t.Fatalf("expected anomaly decode error, got %v", err)
	}
}
