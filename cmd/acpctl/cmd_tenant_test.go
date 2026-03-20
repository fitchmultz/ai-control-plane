// cmd_tenant_test.go - Tests for tenant design-package commands.
//
// Purpose:
//   - Verify the typed tenant inspection and validation workflows.
//
// Responsibilities:
//   - Cover `acpctl tenant inspect` summary output.
//   - Cover `acpctl tenant validate` and the `validate tenant` alias.
//
// Scope:
//   - Command-layer tenant behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests use isolated temporary repositories and deterministic fixtures.
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunTenantInspectTypedPrintsSummary(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newCommandOutputFiles(t)
	code := runTenantInspectTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, tenantInspectOptions{
		RepoRoot:   repoRoot,
		ConfigPath: filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml"),
		Format:     "text",
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runTenantInspectTyped() exit = %d, want %d stderr=%s", code, exitcodes.ACPExitSuccess, readDBCommandOutput(t, stderr))
	}
	output := readDBCommandOutput(t, stdout)
	if !strings.Contains(output, "Tenant design package") || !strings.Contains(output, "Organizations:") {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func TestRunTenantValidateAliasPasses(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newTestFiles(t)
	code := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "validate", "tenant")
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runTestCommand(validate tenant) exit = %d stderr=%s", code, readFile(t, stderr))
	}
	if got := readFile(t, stdout); !strings.Contains(got, "Tenant design validation passed") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunTenantValidateFailsOnTruthDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	writeFile(t, filepath.Join(repoRoot, "docs", "GO_TO_MARKET_SCOPE.md"), "# Scope\n")
	stdout, stderr := newCommandOutputFiles(t)
	code := runTenantValidateTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, tenantValidateOptions{
		RepoRoot:   repoRoot,
		ConfigPath: filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml"),
	})
	if code != exitcodes.ACPExitDomain {
		t.Fatalf("runTenantValidateTyped() exit = %d, want %d", code, exitcodes.ACPExitDomain)
	}
	if got := readDBCommandOutput(t, stderr); !strings.Contains(got, "Multi-tenant managed-service claims") {
		t.Fatalf("stderr = %q", got)
	}
}

func writeTenantCommandFixtureRepo(t *testing.T, repoRoot string) {
	t.Helper()
	writeFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "contract.yaml"), `version: 1
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
	writeFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "tenant_design.schema.json"), tenantDesignSchemaJSON)
	writeFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "litellm.schema.json"), `{}`)
	writeFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "roles.schema.json"), `{}`)
	writeFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "demo_presets.schema.json"), `{}`)
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "litellm.yaml"), `model_list:
  - model_name: openai-gpt5.2
  - model_name: claude-haiku-4-5
  - model_name: claude-sonnet-4-5
`)
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "roles.yaml"), `roles:
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
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml"), tenantDesignYAMLFixture)
	writeFile(t, filepath.Join(repoRoot, "docs", "adr", "0002-multi-tenant-isolation-design.md"), `# 0002: Multi-tenant isolation design

- Status: Accepted
- Date: 2026-03-19

## Context

The repository needs an incubating design-only package.

## Decision

Adopt organization and workspace isolation as the design boundary.

## Consequences

The package remains design-only and incubating until runtime validation exists.
`)
	writeFile(t, filepath.Join(repoRoot, "docs", "policy", "MULTI_TENANT_ISOLATION_AND_BILLING.md"), `# Multi-Tenant Isolation And Billing

This design-only package covers row-level access, workspace isolation, and provider billing boundaries.
`)
	writeFile(t, filepath.Join(repoRoot, "docs", "GO_TO_MARKET_SCOPE.md"), `# Scope

Multi-tenant managed-service claims remain out until additional validation exists.
`)
	writeFile(t, filepath.Join(repoRoot, "docs", "KNOWN_LIMITATIONS.md"), `# Known Limitations

| Severity | Finding | Impact | Mitigation | Owner | Due Date | Status | Evidence Links |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Major | Multi-Tenant Runtime Design-Only | Not supported at runtime yet. | Use dedicated single-tenant deployments until validation exists. | platform | 2026-09-30 | Open | docs/policy/MULTI_TENANT_ISOLATION_AND_BILLING.md |
`)
	writeFile(t, filepath.Join(repoRoot, "docs", "support-matrix.yaml"), `public_docs: []
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

const tenantDesignSchemaJSON = `{"type":"object"}`

const tenantDesignYAMLFixture = `version: 1
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
