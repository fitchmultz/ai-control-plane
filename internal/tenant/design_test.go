// Package tenant defines the tracked design-time multi-tenant contract.
//
// Purpose:
//   - Verify tenant design loading and summary behavior.
//
// Responsibilities:
//   - Cover strict YAML loading and summary derivation.
//   - Keep design-only helpers deterministic for CLI inspection.
//
// Scope:
//   - Unit tests for the tenant design package only.
//
// Usage:
//   - Run with `go test ./internal/tenant`.
//
// Invariants/Assumptions:
//   - Fixtures stay temporary and deterministic.
package tenant

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadFileAndSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenant_design.yaml")
	if err := os.WriteFile(path, []byte(validDesignYAML), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	design, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	summary := design.Summary()
	if summary.OrganizationCount != 2 || summary.WorkspaceCount != 3 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	if summary.ProviderBillingBoundary != "organization" || summary.ChargebackBoundary != "workspace" {
		t.Fatalf("unexpected boundaries: %+v", summary)
	}
	if !reflect.DeepEqual(summary.WorkspaceRefs, []string{"falcon-insurance/claims-adjuster", "falcon-insurance/underwriting", "northwind-health/research"}) {
		t.Fatalf("WorkspaceRefs = %v", summary.WorkspaceRefs)
	}
	if !reflect.DeepEqual(summary.KeyNamespaces, []string{"falcon-insurance--claims-adjuster", "falcon-insurance--underwriting", "northwind-health--research"}) {
		t.Fatalf("KeyNamespaces = %v", summary.KeyNamespaces)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenant_design.yaml")
	if err := os.WriteFile(path, []byte("version: 1\ndesign_state: incubating\nextra: nope\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected strict YAML load error")
	}
}

const validDesignYAML = `version: 1
design_state: incubating
organizations:
  - id: falcon-insurance
    display_name: Falcon Insurance
    chargeback:
      cost_center: cc-1000
      bill_to: ap@falcon.example.com
    workspaces:
      - id: claims-adjuster
        display_name: Claims Adjuster
        key_namespace_prefix: falcon-insurance--claims-adjuster
        allowed_models:
          - openai-gpt5.2
          - claude-haiku-4-5
        role_bindings:
          - role: developer
            scope: workspace
            subjects:
              - kind: group
                name: claims-adjusters
        chargeback:
          cost_center: cc-1100
          bill_to: claims-finops@falcon.example.com
      - id: underwriting
        display_name: Underwriting
        key_namespace_prefix: falcon-insurance--underwriting
        allowed_models:
          - openai-gpt5.2
          - claude-sonnet-4-5
        role_bindings:
          - role: team-lead
            scope: workspace
            subjects:
              - kind: group
                name: underwriters
        chargeback:
          cost_center: cc-1200
          bill_to: underwriting-finops@falcon.example.com
  - id: northwind-health
    display_name: Northwind Health
    chargeback:
      cost_center: cc-2000
      bill_to: ap@northwind.example.com
    workspaces:
      - id: research
        display_name: Research
        key_namespace_prefix: northwind-health--research
        allowed_models:
          - openai-gpt5.2
        role_bindings:
          - role: auditor
            scope: workspace
            subjects:
              - kind: group
                name: research-auditors
        chargeback:
          cost_center: cc-2100
          bill_to: research-finops@northwind.example.com
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
