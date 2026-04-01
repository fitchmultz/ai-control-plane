// lifecycle_scope_test.go - Tests for tenant-scoped key lifecycle helpers.
//
// Purpose:
//   - Verify tenant design lookups and namespace enforcement for key lifecycle
//     commands.
//
// Responsibilities:
//   - Cover organization/workspace scope resolution from tenant design YAML.
//   - Cover inventory filtering and replacement-alias enforcement.
//
// Scope:
//   - Tenant lifecycle scope helpers only.
//
// Usage:
//   - Run via `go test ./internal/keygen`.
//
// Invariants/Assumptions:
//   - Tests stay repository-local and do not require a live gateway.
package keygen

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
)

func TestResolveTenantAccessScopeWorkspaceAndOrganization(t *testing.T) {
	repoRoot := t.TempDir()
	writeLifecycleTenantDesignFixture(t, repoRoot)

	workspaceScope, err := ResolveTenantAccessScope(context.Background(), repoRoot, "", "falcon-insurance", "claims-adjuster")
	if err != nil {
		t.Fatalf("ResolveTenantAccessScope(workspace) error = %v", err)
	}
	if !reflect.DeepEqual(workspaceScope.NamespacePrefixes, []string{"falcon-insurance--claims-adjuster"}) {
		t.Fatalf("workspace scope prefixes = %+v", workspaceScope.NamespacePrefixes)
	}
	if got := workspaceScope.Display(); got != "falcon-insurance/claims-adjuster" {
		t.Fatalf("workspace scope display = %q", got)
	}

	organizationScope, err := ResolveTenantAccessScope(context.Background(), repoRoot, "", "falcon-insurance", "")
	if err != nil {
		t.Fatalf("ResolveTenantAccessScope(organization) error = %v", err)
	}
	if !reflect.DeepEqual(organizationScope.NamespacePrefixes, []string{"falcon-insurance--claims-adjuster", "falcon-insurance--finance"}) {
		t.Fatalf("organization scope prefixes = %+v", organizationScope.NamespacePrefixes)
	}
	if got := organizationScope.Display(); got != "falcon-insurance" {
		t.Fatalf("organization scope display = %q", got)
	}
}

func TestFilterKeysForTenantScope(t *testing.T) {
	scope := &TenantAccessScope{OrganizationID: "falcon-insurance", NamespacePrefixes: []string{"falcon-insurance--claims-adjuster"}}
	keys := []gateway.KeyInfo{
		{KeyAlias: "falcon-insurance--claims-adjuster--svc-claims__cc-1100"},
		{KeyAlias: "other-org--workspace--svc__cc-2200"},
	}

	filtered := FilterKeysForTenantScope(keys, scope)
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("filtered key count = %d, want %d", got, want)
	}
	if got := filtered[0].Alias(); got != "falcon-insurance--claims-adjuster--svc-claims__cc-1100" {
		t.Fatalf("filtered alias = %q", got)
	}
}

func TestValidateReplacementAliasForTenantScope(t *testing.T) {
	scope := &TenantAccessScope{
		OrganizationID:    "falcon-insurance",
		WorkspaceID:       "claims-adjuster",
		NamespacePrefixes: []string{"falcon-insurance--claims-adjuster", "falcon-insurance--finance"},
	}
	sourceAlias := "falcon-insurance--claims-adjuster--svc-claims__cc-1100"

	if err := ValidateReplacementAliasForTenantScope(sourceAlias, "falcon-insurance--claims-adjuster--svc-claims-rotated__cc-1100", scope); err != nil {
		t.Fatalf("ValidateReplacementAliasForTenantScope(valid) error = %v", err)
	}
	if err := ValidateReplacementAliasForTenantScope(sourceAlias, "falcon-insurance--finance--svc-claims__cc-1100", scope); err == nil {
		t.Fatalf("expected namespace mismatch error")
	}
	if err := ValidateReplacementAliasForTenantScope(sourceAlias, "falcon-insurance--claims-adjuster--svc-claims__cc-2200", scope); err == nil {
		t.Fatalf("expected attribution mismatch error")
	}
}

func writeLifecycleTenantDesignFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "demo", "config", "tenant_design.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	const content = `version: 1
design_state: incubating
organizations:
  - id: falcon-insurance
    display_name: Falcon Insurance
    chargeback:
      cost_center: cc-1000
      bill_to: falcon-billing
    workspaces:
      - id: claims-adjuster
        display_name: Claims Adjuster
        key_namespace_prefix: falcon-insurance--claims-adjuster
        allowed_models: [openai-gpt5.2]
        role_bindings: []
        chargeback:
          cost_center: cc-1100
          bill_to: claims
      - id: finance
        display_name: Finance
        key_namespace_prefix: falcon-insurance--finance
        allowed_models: [openai-gpt5.2]
        role_bindings: []
        chargeback:
          cost_center: cc-1200
          bill_to: finance
row_level_access:
  mode: design-only
  required_predicates: [organization_id, workspace_id, key_namespace_prefix]
  write_boundary: workspace
reporting:
  mode: design-only
  tenant_safe_by_default: true
  metadata_only: true
  allowed_aggregation_levels: [workspace, organization]
  report_definitions: []
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
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
