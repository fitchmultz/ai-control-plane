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
	"os"
	"path/filepath"
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
	if input.Scope.Kind != ReportScopeGlobal || input.Scope.Label != "global" {
		t.Fatalf("expected global scope default, got %#v", input.Scope)
	}
	if input.Notification.GenericWebhookURL == "" || input.Notification.SlackWebhookURL == "" {
		t.Fatalf("expected webhook URLs, got %#v", input.Notification)
	}
}

func TestNewReportWorkflowInputResolvesWorkspaceScope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "demo", "config"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml"), []byte(testTenantDesignYAML), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	input, err := NewReportWorkflowInput(ReportCommandInput{
		OrganizationID: "falcon-insurance",
		WorkspaceID:    "claims-adjuster",
	}, fakeEnv{}, repoRoot, func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewReportWorkflowInput returned error: %v", err)
	}
	if input.Scope.Kind != ReportScopeWorkspace || input.Scope.Label != "workspace/falcon-insurance/claims-adjuster" {
		t.Fatalf("unexpected scope: %#v", input.Scope)
	}
	if input.QueryScope == nil || len(input.QueryScope.NamespacePrefixes) != 1 || input.QueryScope.NamespacePrefixes[0] != "falcon-insurance--claims-adjuster" {
		t.Fatalf("unexpected query scope: %#v", input.QueryScope)
	}
}

func TestNewReportWorkflowInputRejectsWorkspaceWithoutOrganization(t *testing.T) {
	t.Parallel()

	_, err := NewReportWorkflowInput(ReportCommandInput{WorkspaceID: "claims-adjuster"}, fakeEnv{}, "/repo", nil)
	if err == nil || !strings.Contains(err.Error(), "--organization is required") {
		t.Fatalf("expected partial scope error, got %v", err)
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
			"CHARGEBACK_SCOPE_KIND":                     "workspace",
			"CHARGEBACK_SCOPE_LABEL":                    "workspace/falcon-insurance/claims-adjuster",
			"CHARGEBACK_SCOPE_AGGREGATION":              "workspace",
			"CHARGEBACK_SCOPE_ORGANIZATION_ID":          "falcon-insurance",
			"CHARGEBACK_SCOPE_WORKSPACE_ID":             "claims-adjuster",
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
	if request.Input.Scope.Label != "workspace/falcon-insurance/claims-adjuster" {
		t.Fatalf("expected scope label, got %#v", request.Input.Scope)
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

const testTenantDesignYAML = `version: 1
design_state: incubating
organizations:
  - id: falcon-insurance
    display_name: Falcon Insurance
    chargeback:
      cost_center: cc-1000
      bill_to: ap@falcon-insurance.example.com
    workspaces:
      - id: claims-adjuster
        display_name: Claims Adjuster
        key_namespace_prefix: falcon-insurance--claims-adjuster
        allowed_models:
          - openai-gpt5.2
        role_bindings:
          - role: developer
            scope: workspace
            subjects:
              - kind: group
                name: claims-adjusters
        chargeback:
          cost_center: cc-1100
          bill_to: claims-finops@falcon-insurance.example.com
row_level_access:
  mode: design-only
  required_predicates:
    - organization_id
    - workspace_id
    - key_namespace_prefix
  write_boundary: workspace
reporting:
  mode: design-only
  tenant_safe_by_default: true
  metadata_only: true
  allowed_aggregation_levels:
    - workspace
    - organization
  report_definitions:
    - id: workspace-showback
      aggregation: workspace
      redaction: metadata-only
      include_cross_organization_data: false
    - id: organization-portfolio
      aggregation: organization
      redaction: metadata-only
      include_cross_organization_data: false
chargeback:
  mode: design-only
  billable_boundary: workspace
  allow_cross_organization_allocation: false
  require_workspace_cost_center: true
provider_billing:
  mode: design-only
  customer_invoice_boundary: organization
  passthrough_usage_costs: true
  separate_platform_fee: true
  forbid_cross_organization_subsidy: true
`
