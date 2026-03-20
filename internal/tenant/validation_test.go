// Package tenant defines the tracked design-time multi-tenant contract.
//
// Purpose:
//   - Verify semantic validation for the tenant design package.
//
// Responsibilities:
//   - Cover successful design validation.
//   - Cover key isolation, role/model, and reporting boundary failures.
//
// Scope:
//   - Unit tests for tenant validation only.
//
// Usage:
//   - Run with `go test ./internal/tenant`.
//
// Invariants/Assumptions:
//   - Validation findings remain deterministic and sorted.
package tenant

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestValidateSuccess(t *testing.T) {
	design, err := LoadFile(writeTempDesign(t, validValidationDesignYAML))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	issues := Validate(design, ValidationOptions{
		SourcePath: "demo/config/tenant_design.yaml",
		KnownModels: map[string]struct{}{
			"openai-gpt5.2":     {},
			"claude-haiku-4-5":  {},
			"claude-sonnet-4-5": {},
		},
		KnownRoles: map[string]struct{}{
			"admin":     {},
			"team-lead": {},
			"developer": {},
			"auditor":   {},
		},
		RoleModels: map[string]map[string]struct{}{
			"developer": {"openai-gpt5.2": {}, "claude-haiku-4-5": {}},
			"team-lead": {"openai-gpt5.2": {}, "claude-haiku-4-5": {}, "claude-sonnet-4-5": {}},
			"auditor":   {},
		},
	})
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestValidateFlagsBoundaryDrift(t *testing.T) {
	design, err := LoadFile(writeTempDesign(t, strings.ReplaceAll(validValidationDesignYAML, "customer_invoice_boundary: organization", "customer_invoice_boundary: workspace")))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	issues := Validate(design, ValidationOptions{SourcePath: "demo/config/tenant_design.yaml"})
	if !containsIssue(issues, "provider_billing.customer_invoice_boundary must be \"organization\"") {
		t.Fatalf("expected provider billing issue, got %v", issues)
	}
}

func TestValidateFlagsUnknownModelAndUnreachableModel(t *testing.T) {
	design, err := LoadFile(writeTempDesign(t, strings.Replace(validValidationDesignYAML, "- openai-gpt5.2\n", "- claude-sonnet-4-5\n", 1)))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	issues := Validate(design, ValidationOptions{
		SourcePath: "demo/config/tenant_design.yaml",
		KnownModels: map[string]struct{}{
			"openai-gpt5.2": {},
		},
		KnownRoles: map[string]struct{}{"developer": {}},
		RoleModels: map[string]map[string]struct{}{
			"developer": {"openai-gpt5.2": {}},
		},
	})
	if !containsIssue(issues, "references unknown model alias \"claude-sonnet-4-5\"") {
		t.Fatalf("expected unknown model issue, got %v", issues)
	}
	if !containsIssue(issues, "is not reachable by any bound role") {
		t.Fatalf("expected unreachable model issue, got %v", issues)
	}
}

func TestValidateFlagsNamespaceDrift(t *testing.T) {
	broken := strings.Replace(validValidationDesignYAML, "key_namespace_prefix: falcon-insurance--claims-adjuster", "key_namespace_prefix: wrong-prefix", 1)
	design, err := LoadFile(writeTempDesign(t, broken))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	issues := Validate(design, ValidationOptions{SourcePath: "demo/config/tenant_design.yaml"})
	if !containsIssue(issues, "key_namespace_prefix must be \"falcon-insurance--claims-adjuster\"") {
		t.Fatalf("expected namespace issue, got %v", issues)
	}
}

func TestValidateFlagsTopLevelDesignDrift(t *testing.T) {
	design := mustLoadValidationDesign(t)
	design.Version = 2
	design.DesignState = DesignState("supported")
	design.RowLevelAccess.Mode = EnforcementMode("runtime")
	design.RowLevelAccess.RequiredPredicates = []string{"organization_id"}
	design.RowLevelAccess.WriteBoundary = IsolationBoundaryOrganization
	design.Reporting.Mode = EnforcementMode("runtime")
	design.Reporting.TenantSafeByDefault = false
	design.Reporting.MetadataOnly = false
	design.Reporting.ReportDefinitions = []ReportDefinition{
		{ID: "bad report", Aggregation: AggregationLevelOrganization, Redaction: RedactionLevel("full"), IncludeCrossOrganizationData: true},
		{ID: "bad report", Aggregation: AggregationLevelOrganization, Redaction: RedactionLevel("full"), IncludeCrossOrganizationData: true},
	}
	design.Chargeback.Mode = EnforcementMode("runtime")
	design.Chargeback.BillableBoundary = IsolationBoundaryOrganization
	design.Chargeback.AllowCrossOrganizationAllocation = true
	design.Chargeback.RequireWorkspaceCostCenter = false
	design.ProviderBilling.Mode = EnforcementMode("runtime")
	design.ProviderBilling.CustomerInvoiceBoundary = IsolationBoundaryWorkspace
	design.ProviderBilling.PassthroughUsageCosts = false
	design.ProviderBilling.SeparatePlatformFee = false
	design.ProviderBilling.ForbidCrossOrganizationSubsidy = false

	issues := Validate(design, ValidationOptions{SourcePath: "demo/config/tenant_design.yaml"})
	for _, snippet := range []string{
		"version must be 1",
		"design_state must remain \"incubating\"",
		"row_level_access.mode must remain \"design-only\"",
		"required_predicates must include workspace_id",
		"required_predicates must include key_namespace_prefix",
		"row_level_access.write_boundary must be \"workspace\"",
		"reporting.mode must remain \"design-only\"",
		"reporting.tenant_safe_by_default must be true",
		"reporting.metadata_only must be true",
		"duplicate reporting.report_definitions id",
		"include_cross_organization_data must be false",
		"chargeback.mode must remain \"design-only\"",
		"chargeback.billable_boundary must be \"workspace\"",
		"chargeback.allow_cross_organization_allocation must be false",
		"chargeback.require_workspace_cost_center must be true",
		"provider_billing.mode must remain \"design-only\"",
		"provider_billing.customer_invoice_boundary must be \"organization\"",
		"provider_billing.passthrough_usage_costs must be true",
		"provider_billing.separate_platform_fee must be true",
		"provider_billing.forbid_cross_organization_subsidy must be true",
	} {
		if !containsIssue(issues, snippet) {
			t.Fatalf("expected issue containing %q, got %v", snippet, issues)
		}
	}
}

func TestValidateFlagsOrganizationAndWorkspaceDrift(t *testing.T) {
	design := mustLoadValidationDesign(t)
	design.Organizations[0].ID = "Falcon"
	design.Organizations[0].DisplayName = ""
	design.Organizations[0].Chargeback.CostCenter = ""
	design.Organizations[0].Chargeback.BillTo = ""
	design.Organizations = append(design.Organizations, design.Organizations[0])
	design.Organizations[0].Workspaces[0].ID = "Claims"
	design.Organizations[0].Workspaces[0].DisplayName = ""
	design.Organizations[0].Workspaces[0].KeyNamespacePrefix = ""
	design.Organizations[0].Workspaces[0].AllowedModels = []string{""}
	design.Organizations[0].Workspaces[0].RoleBindings = []RoleBinding{{Role: "unknown", Scope: IsolationBoundaryOrganization}}
	design.Organizations[0].Workspaces[0].Chargeback.CostCenter = ""
	design.Organizations[0].Workspaces[0].Chargeback.BillTo = ""

	issues := Validate(design, ValidationOptions{
		SourcePath:  "demo/config/tenant_design.yaml",
		KnownModels: map[string]struct{}{"openai-gpt5.2": {}},
		KnownRoles:  map[string]struct{}{"developer": {}},
		RoleModels:  map[string]map[string]struct{}{"developer": {"openai-gpt5.2": {}}},
	})
	for _, snippet := range []string{
		"organizations[0].id \"Falcon\" must match",
		"organizations[0].display_name must not be blank",
		"organizations[0].chargeback.cost_center must not be blank",
		"organizations[0].chargeback.bill_to must not be blank",
		"duplicate organizations id",
		"workspaces[0].id \"Claims\" must match",
		"display_name must not be blank",
		"key_namespace_prefix must not be blank",
		"allowed_models[0] must not be blank",
		"scope must be \"workspace\"",
		"references unknown RBAC role \"unknown\"",
		"subjects must not be empty",
		"chargeback.cost_center must not be blank",
		"chargeback.bill_to must not be blank",
	} {
		if !containsIssue(issues, snippet) {
			t.Fatalf("expected issue containing %q, got %v", snippet, issues)
		}
	}
}

func TestValidateAllowsWildcardRoleModel(t *testing.T) {
	design := mustLoadValidationDesign(t)
	design.Organizations[0].Workspaces[0].AllowedModels = []string{"claude-sonnet-4-5"}
	design.Organizations[0].Workspaces[0].RoleBindings = []RoleBinding{{
		Role:     "admin",
		Scope:    IsolationBoundaryWorkspace,
		Subjects: []SubjectRef{{Kind: SubjectKindGroup, Name: "claims-admins"}},
	}}
	issues := Validate(design, ValidationOptions{
		SourcePath: "demo/config/tenant_design.yaml",
		KnownModels: map[string]struct{}{
			"claude-sonnet-4-5": {},
		},
		KnownRoles: map[string]struct{}{"admin": {}},
		RoleModels: map[string]map[string]struct{}{
			"admin": {wildcardModel: {}},
		},
	})
	if len(issues) != 0 {
		t.Fatalf("expected wildcard role to allow workspace model, got %v", issues)
	}
}

func mustLoadValidationDesign(t *testing.T) Design {
	t.Helper()
	design, err := LoadFile(writeTempDesign(t, validValidationDesignYAML))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	return design
}

func containsIssue(issues []string, snippet string) bool {
	return slices.ContainsFunc(issues, func(issue string) bool {
		return strings.Contains(issue, snippet)
	})
}

func writeTempDesign(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/tenant_design.yaml"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

const validValidationDesignYAML = `version: 1
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
