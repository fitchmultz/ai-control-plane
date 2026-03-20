// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Verify tenant design validation behavior and truth-boundary checks.
//
// Responsibilities:
//   - Cover successful tenant design validation.
//   - Cover support-matrix and claim-discipline drift detection.
//
// Scope:
//   - Unit tests for tenant validation only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use temporary repositories with deterministic tracked fixtures.
package validation

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestValidateTenantConfigSuccess(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantValidationFixtureRepo(t, repoRoot)
	issues, err := ValidateTenantConfig(repoRoot, TenantValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateTenantConfig() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestValidateTenantConfigFlagsTruthDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantValidationFixtureRepo(t, repoRoot)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "GO_TO_MARKET_SCOPE.md"), "# scope\n")
	issues, err := ValidateTenantConfig(repoRoot, TenantValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateTenantConfig() error = %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "Multi-tenant managed-service claims") {
		t.Fatalf("expected scope drift issue, got %v", issues)
	}
}

func TestValidateTenantConfigFlagsMissingSupportSurface(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantValidationFixtureRepo(t, repoRoot)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "support-matrix.yaml"), "public_docs: []\nreference_docs: []\nincubating_terms: []\nsurfaces: []\n")
	issues, err := ValidateTenantConfig(repoRoot, TenantValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateTenantConfig() error = %v", err)
	}
	if !strings.Contains(strings.Join(issues, "\n"), "missing incubating surface \"multi-tenant-isolation-design\"") {
		t.Fatalf("expected support surface issue, got %v", issues)
	}
}

func TestValidateTenantSchemaAndHelpers(t *testing.T) {
	repoRoot := t.TempDir()
	schemaPath := filepath.Join(repoRoot, "tenant_design.schema.json")
	documentPath := filepath.Join(repoRoot, "tenant_design.yaml")
	writeFixtureFile(t, schemaPath, `{"type":"object","required":["version"]}`)
	writeFixtureFile(t, documentPath, "design_state: incubating\n")
	issues := validateTenantSchema(documentPath, schemaPath, "demo/config/tenant_design.yaml")
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "version is required") {
		t.Fatalf("expected schema issue, got %v", issues)
	}

	surfaceIssues := validateTenantSupportSurface(catalog.SupportMatrix{Surfaces: []catalog.SupportSurface{{
		ID:         tenantSupportSurfaceID,
		Status:     "supported",
		Paths:      []string{"demo/config/tenant_design.yaml"},
		Validation: []string{"make something-else"},
	}}})
	joined := strings.Join(surfaceIssues, "\n")
	for _, snippet := range []string{"must remain incubating", "must include path \"docs/policy/MULTI_TENANT_ISOLATION_AND_BILLING.md\"", "must include validation command \"make validate-tenant\""} {
		if !strings.Contains(joined, snippet) {
			t.Fatalf("expected support helper issue %q, got %v", snippet, surfaceIssues)
		}
	}

	if got := displayTenantPath(repoRoot, filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml")); got != "demo/config/tenant_design.yaml" {
		t.Fatalf("displayTenantPath() = %q", got)
	}
	if got := displayTenantPath(repoRoot, "/tmp/outside.yaml"); got != "/tmp/outside.yaml" {
		t.Fatalf("displayTenantPath(outside) = %q", got)
	}
}

func writeTenantValidationFixtureRepo(t *testing.T, repoRoot string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "contract.yaml"), `version: 1
schemas:
  - id: litellm
    path: demo/config/litellm.yaml
    schema: docs/contracts/config/litellm.schema.json
  - id: roles
    path: demo/config/roles.yaml
    schema: docs/contracts/config/roles.schema.json
  - id: demo-presets
    path: demo/config/demo_presets.yaml
    schema: docs/contracts/config/demo_presets.schema.json
  - id: tenant-design
    path: demo/config/tenant_design.yaml
    schema: docs/contracts/config/tenant_design.schema.json
naming:
  model_alias_pattern: '^[a-z0-9]+(?:[-.][a-z0-9]+)*$'
  role_name_pattern: '^[a-z0-9]+(?:-[a-z0-9]+)*$'
  preset_name_pattern: '^[a-z0-9]+(?:-[a-z0-9]+)*$'
runtime:
  allowed_overlays:
    - tls
    - ui
    - dlp
    - offline
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "tenant_design.schema.json"), tenantSchemaFixture)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "litellm.schema.json"), `{}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "roles.schema.json"), `{}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "demo_presets.schema.json"), `{}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "litellm.yaml"), `model_list:
  - model_name: openai-gpt5.2
  - model_name: claude-haiku-4-5
  - model_name: claude-sonnet-4-5
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "roles.yaml"), `roles:
  developer:
    description: Developer
    model_access: [openai-gpt5.2, claude-haiku-4-5]
    budget_ceiling: 25
    can_approve: false
    can_assign_roles: false
    can_create_keys: true
    read_only: false
    approval_authority: null
  team-lead:
    description: Team Lead
    model_access: [openai-gpt5.2, claude-haiku-4-5, claude-sonnet-4-5]
    budget_ceiling: 100
    can_approve: true
    can_assign_roles: false
    can_create_keys: true
    read_only: false
    approval_authority: null
default_role: developer
model_tiers:
  standard: [openai-gpt5.2]
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml"), tenantDesignFixture)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "adr", "0002-multi-tenant-isolation-design.md"), `# 0002: Multi-tenant isolation design

- Status: Accepted
- Date: 2026-03-19

## Context

The repository needs a design-only, incubating package for future tenant isolation.

## Decision

Adopt a design-only tenant package with organization and workspace isolation.

## Consequences

The package remains design-only and incubating until runtime validation exists.
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "policy", "MULTI_TENANT_ISOLATION_AND_BILLING.md"), `# Multi-Tenant Isolation And Billing

This design-only package covers row-level access, workspace isolation, and provider billing boundaries.
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "GO_TO_MARKET_SCOPE.md"), `# Scope

Multi-tenant managed-service claims remain out until additional validation exists.
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "KNOWN_LIMITATIONS.md"), `# Known Limitations

| Severity | Finding | Impact | Mitigation | Owner | Due Date | Status | Evidence Links |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Major | Multi-Tenant Runtime Design-Only | Not supported at runtime yet. | Use dedicated single-tenant deployments until validation exists. | platform | 2026-09-30 | Open | docs/policy/MULTI_TENANT_ISOLATION_AND_BILLING.md |
`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "support-matrix.yaml"), `public_docs: []
reference_docs: []
incubating_terms: []
surfaces:
  - id: multi-tenant-isolation-design
    label: Multi-tenant isolation design package
    status: incubating
    summary: Design-only package for future tenant-safe isolation and billing.
    owner: platform
    paths:
      - demo/config/tenant_design.yaml
      - docs/policy/MULTI_TENANT_ISOLATION_AND_BILLING.md
    validation:
      - make validate-tenant
`)
}

const tenantSchemaFixture = `{"type":"object"}`

const tenantDesignFixture = `version: 1
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
