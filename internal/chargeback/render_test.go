// render_test.go - Tests for canonical chargeback rendering helpers.
//
// Purpose:
//   - Verify chargeback JSON, CSV, and webhook payload rendering stays safe.
//
// Responsibilities:
//   - Cover hostile characters in structured inputs.
//   - Assert spreadsheet-safe CSV escaping behavior.
//   - Confirm JSON payloads remain valid and preserve field content.
//
// Non-scope:
//   - Does not execute shell scripts or database queries.
//   - Does not validate markdown rendering behavior.
//
// Invariants/Assumptions:
//   - Tests use the package-level canonical rendering entrypoints only.
package chargeback

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSONPreservesHostileContent(t *testing.T) {
	input := ReportInput{
		SchemaVersion:      "1.0.0",
		GeneratedAt:        "2026-03-07T17:00:00Z",
		ReportMonth:        "2026-02",
		PeriodStart:        "2026-02-01",
		PeriodEnd:          "2026-02-28",
		TotalSpend:         123.45,
		TotalRequests:      9,
		TotalTokens:        42,
		CostCenters:        []CostCenterAllocation{{CostCenter: "cc,\"line\"\nnext", Team: `ops\blue`, RequestCount: 3, TokenCount: 7, SpendAmount: 23.5, PercentOfTotal: 19.03}, {CostCenter: "unknown-cc", Team: "unassigned", RequestCount: 1, TokenCount: 2, SpendAmount: 4.5, PercentOfTotal: 3.64}},
		Models:             []ModelAllocation{{Model: `gpt-"4o",\nmini`, RequestCount: 2, TokenCount: 5, SpendAmount: 12.34}},
		Variance:           "N/A",
		VarianceThreshold:  15,
		PreviousMonthSpend: 111.11,
		Anomalies:          []Anomaly{{CostCenter: `=@finance`, Team: "team,\nquoted", CurrentSpend: 22.2, PreviousSpend: 10.1, SpikePercent: 119.8, Type: `sp"ike`}},
		ForecastEnabled:    true,
		DailyBurn:          4.2,
		ExhaustionDate:     "N/A",
		TotalBudget:        500,
		BudgetRisk:         BudgetRisk{RiskLevel: "medium", ThresholdExceeded: false},
		AnomalyThreshold:   200,
	}

	var output bytes.Buffer
	if err := RenderJSON(&output, input); err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("rendered JSON invalid: %v\n%s", err, output.String())
	}

	costCenters := decoded["allocations_by_cost_center"].([]any)
	firstCostCenter := costCenters[0].(map[string]any)
	if firstCostCenter["cost_center"] != "cc,\"line\"\nnext" {
		t.Fatalf("cost center content lost: %#v", firstCostCenter["cost_center"])
	}
	if firstCostCenter["team"] != `ops\blue` {
		t.Fatalf("team content lost: %#v", firstCostCenter["team"])
	}

	anomalies := decoded["anomalies"].([]any)
	firstAnomaly := anomalies[0].(map[string]any)
	if firstAnomaly["cost_center"] != "=@finance" {
		t.Fatalf("anomaly cost center content lost: %#v", firstAnomaly["cost_center"])
	}
	if decoded["variance_analysis"].(map[string]any)["variance_percent"] != "N/A" {
		t.Fatalf("variance N/A should remain string, got %#v", decoded["variance_analysis"].(map[string]any)["variance_percent"])
	}
}

func TestRenderCSVEscapesAndSpreadsheetProtects(t *testing.T) {
	var output bytes.Buffer
	rows := []CostCenterAllocation{
		{
			CostCenter:     `=SUM(1,2)`,
			Team:           "\"quoted\",team\nnext",
			SpendAmount:    99.01,
			RequestCount:   12,
			TokenCount:     34,
			PercentOfTotal: 45.6,
		},
		{
			CostCenter:     `normal`,
			Team:           `@cmd`,
			SpendAmount:    1.2,
			RequestCount:   1,
			TokenCount:     2,
			PercentOfTotal: 3.4,
		},
	}

	if err := RenderCSV(&output, "2026-02", rows); err != nil {
		t.Fatalf("RenderCSV returned error: %v", err)
	}

	reader := csv.NewReader(strings.NewReader(output.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV invalid: %v\n%s", err, output.String())
	}
	if got := records[1][0]; got != "'=SUM(1,2)" {
		t.Fatalf("expected spreadsheet protection for formula cell, got %q", got)
	}
	if got := records[1][1]; got != "\"quoted\",team\nnext" {
		t.Fatalf("expected embedded quotes/newlines preserved, got %q", got)
	}
	if got := records[2][1]; got != "'@cmd" {
		t.Fatalf("expected spreadsheet protection for @-prefixed cell, got %q", got)
	}
}

func TestWebhookPayloadsAreValidJSON(t *testing.T) {
	generic, err := BuildGenericWebhookPayload(GenericWebhookInput{
		Event:       "chargeback_report_generated",
		ReportMonth: "2026-02",
		TotalSpend:  18.5,
		Variance:    `12.5`,
		Anomalies:   []Anomaly{{CostCenter: `cc,"ops"`, Team: "team\nx", CurrentSpend: 8, PreviousSpend: 4, SpikePercent: 100, Type: "spike"}},
		Timestamp:   "2026-03-07T17:00:00Z",
	})
	if err != nil {
		t.Fatalf("BuildGenericWebhookPayload returned error: %v", err)
	}

	slack, err := BuildSlackWebhookPayload(SlackWebhookInput{
		ReportMonth: "2026-02",
		TotalSpend:  18.5,
		Variance:    "12.5",
		Color:       "danger",
		Epoch:       12345,
	})
	if err != nil {
		t.Fatalf("BuildSlackWebhookPayload returned error: %v", err)
	}

	for name, payload := range map[string][]byte{"generic": generic, "slack": slack} {
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("%s payload invalid JSON: %v\n%s", name, err, string(payload))
		}
	}
}
