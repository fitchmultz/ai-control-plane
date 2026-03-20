// Package tenant defines the tracked design-time multi-tenant contract.
//
// Purpose:
//   - Validate the design-only tenant model independently of runtime systems.
//
// Responsibilities:
//   - Enforce structural and semantic invariants for organizations, workspaces,
//     row-level access design, tenant-safe reporting, chargeback boundaries,
//     and service-provider billing boundaries.
//   - Cross-check tenant-scoped role bindings and allowed models against known
//     repository-owned catalogs provided by callers.
//
// Scope:
//   - In-memory design validation only.
//
// Usage:
//   - Used by `internal/validation` after loading cross-file inputs.
//
// Invariants/Assumptions:
//   - Design validation must remain deterministic.
//   - Validation findings describe design drift, not runtime behavior.
package tenant

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var tenantKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type ValidationOptions struct {
	SourcePath  string
	KnownModels map[string]struct{}
	KnownRoles  map[string]struct{}
	RoleModels  map[string]map[string]struct{}
}

// Validate enforces semantic design invariants for a loaded tenant design.
func Validate(design Design, opts ValidationOptions) []string {
	sourcePath := strings.TrimSpace(opts.SourcePath)
	if sourcePath == "" {
		sourcePath = "tenant design"
	}

	issues := make([]string, 0)
	addIssue := func(format string, args ...any) {
		issues = append(issues, fmt.Sprintf(format, args...))
	}

	if design.Version != 1 {
		addIssue("%s: version must be 1", sourcePath)
	}
	if design.DesignState != DesignStateIncubating {
		addIssue("%s: design_state must remain %q", sourcePath, DesignStateIncubating)
	}
	if len(design.Organizations) == 0 {
		addIssue("%s: organizations must not be empty", sourcePath)
	}

	if design.RowLevelAccess.Mode != EnforcementModeDesignOnly {
		addIssue("%s: row_level_access.mode must remain %q", sourcePath, EnforcementModeDesignOnly)
	}
	for _, predicate := range []string{"organization_id", "workspace_id", "key_namespace_prefix"} {
		if !containsString(design.RowLevelAccess.RequiredPredicates, predicate) {
			addIssue("%s: row_level_access.required_predicates must include %s", sourcePath, predicate)
		}
	}
	if design.RowLevelAccess.WriteBoundary != IsolationBoundaryWorkspace {
		addIssue("%s: row_level_access.write_boundary must be %q", sourcePath, IsolationBoundaryWorkspace)
	}

	if design.Reporting.Mode != EnforcementModeDesignOnly {
		addIssue("%s: reporting.mode must remain %q", sourcePath, EnforcementModeDesignOnly)
	}
	if !design.Reporting.TenantSafeByDefault {
		addIssue("%s: reporting.tenant_safe_by_default must be true", sourcePath)
	}
	if !design.Reporting.MetadataOnly {
		addIssue("%s: reporting.metadata_only must be true", sourcePath)
	}
	if len(design.Reporting.AllowedAggregationLevels) == 0 {
		addIssue("%s: reporting.allowed_aggregation_levels must not be empty", sourcePath)
	}
	if len(design.Reporting.ReportDefinitions) == 0 {
		addIssue("%s: reporting.report_definitions must not be empty", sourcePath)
	}

	allowedAggregations := make(map[AggregationLevel]struct{}, len(design.Reporting.AllowedAggregationLevels))
	for _, level := range design.Reporting.AllowedAggregationLevels {
		allowedAggregations[level] = struct{}{}
	}
	seenReports := make(map[string]struct{}, len(design.Reporting.ReportDefinitions))
	for index, report := range design.Reporting.ReportDefinitions {
		if !tenantKeyPattern.MatchString(strings.TrimSpace(report.ID)) {
			addIssue("%s: reporting.report_definitions[%d].id %q must match %s", sourcePath, index, report.ID, tenantKeyPattern.String())
		}
		if _, exists := seenReports[report.ID]; exists {
			addIssue("%s: duplicate reporting.report_definitions id %q", sourcePath, report.ID)
		}
		seenReports[report.ID] = struct{}{}
		if _, ok := allowedAggregations[report.Aggregation]; !ok {
			addIssue("%s: reporting.report_definitions[%d].aggregation %q is not present in reporting.allowed_aggregation_levels", sourcePath, index, report.Aggregation)
		}
		if report.Redaction != RedactionLevelMetadataOnly {
			addIssue("%s: reporting.report_definitions[%d].redaction must remain %q", sourcePath, index, RedactionLevelMetadataOnly)
		}
		if report.IncludeCrossOrganizationData {
			addIssue("%s: reporting.report_definitions[%d].include_cross_organization_data must be false", sourcePath, index)
		}
	}

	if design.Chargeback.Mode != EnforcementModeDesignOnly {
		addIssue("%s: chargeback.mode must remain %q", sourcePath, EnforcementModeDesignOnly)
	}
	if design.Chargeback.BillableBoundary != IsolationBoundaryWorkspace {
		addIssue("%s: chargeback.billable_boundary must be %q", sourcePath, IsolationBoundaryWorkspace)
	}
	if design.Chargeback.AllowCrossOrganizationAllocation {
		addIssue("%s: chargeback.allow_cross_organization_allocation must be false", sourcePath)
	}
	if !design.Chargeback.RequireWorkspaceCostCenter {
		addIssue("%s: chargeback.require_workspace_cost_center must be true", sourcePath)
	}

	if design.ProviderBilling.Mode != EnforcementModeDesignOnly {
		addIssue("%s: provider_billing.mode must remain %q", sourcePath, EnforcementModeDesignOnly)
	}
	if design.ProviderBilling.CustomerInvoiceBoundary != IsolationBoundaryOrganization {
		addIssue("%s: provider_billing.customer_invoice_boundary must be %q", sourcePath, IsolationBoundaryOrganization)
	}
	if !design.ProviderBilling.PassthroughUsageCosts {
		addIssue("%s: provider_billing.passthrough_usage_costs must be true", sourcePath)
	}
	if !design.ProviderBilling.SeparatePlatformFee {
		addIssue("%s: provider_billing.separate_platform_fee must be true", sourcePath)
	}
	if !design.ProviderBilling.ForbidCrossOrganizationSubsidy {
		addIssue("%s: provider_billing.forbid_cross_organization_subsidy must be true", sourcePath)
	}

	seenOrganizations := make(map[string]struct{}, len(design.Organizations))
	seenNamespaces := make(map[string]struct{})
	for orgIndex, org := range design.Organizations {
		if !tenantKeyPattern.MatchString(strings.TrimSpace(org.ID)) {
			addIssue("%s: organizations[%d].id %q must match %s", sourcePath, orgIndex, org.ID, tenantKeyPattern.String())
		}
		if _, exists := seenOrganizations[org.ID]; exists {
			addIssue("%s: duplicate organizations id %q", sourcePath, org.ID)
		}
		seenOrganizations[org.ID] = struct{}{}
		if strings.TrimSpace(org.DisplayName) == "" {
			addIssue("%s: organizations[%d].display_name must not be blank", sourcePath, orgIndex)
		}
		if strings.TrimSpace(org.Chargeback.CostCenter) == "" {
			addIssue("%s: organizations[%d].chargeback.cost_center must not be blank", sourcePath, orgIndex)
		}
		if strings.TrimSpace(org.Chargeback.BillTo) == "" {
			addIssue("%s: organizations[%d].chargeback.bill_to must not be blank", sourcePath, orgIndex)
		}
		if len(org.Workspaces) == 0 {
			addIssue("%s: organizations[%d].workspaces must not be empty", sourcePath, orgIndex)
		}

		seenWorkspaces := make(map[string]struct{}, len(org.Workspaces))
		for workspaceIndex, workspace := range org.Workspaces {
			if !tenantKeyPattern.MatchString(strings.TrimSpace(workspace.ID)) {
				addIssue("%s: organizations[%d].workspaces[%d].id %q must match %s", sourcePath, orgIndex, workspaceIndex, workspace.ID, tenantKeyPattern.String())
			}
			if _, exists := seenWorkspaces[workspace.ID]; exists {
				addIssue("%s: duplicate workspace id %q under organization %q", sourcePath, workspace.ID, org.ID)
			}
			seenWorkspaces[workspace.ID] = struct{}{}
			if strings.TrimSpace(workspace.DisplayName) == "" {
				addIssue("%s: organizations[%d].workspaces[%d].display_name must not be blank", sourcePath, orgIndex, workspaceIndex)
			}
			wantNamespace := org.ID + "--" + workspace.ID
			if strings.TrimSpace(workspace.KeyNamespacePrefix) == "" {
				addIssue("%s: organizations[%d].workspaces[%d].key_namespace_prefix must not be blank", sourcePath, orgIndex, workspaceIndex)
			} else {
				if workspace.KeyNamespacePrefix != wantNamespace {
					addIssue("%s: organizations[%d].workspaces[%d].key_namespace_prefix must be %q", sourcePath, orgIndex, workspaceIndex, wantNamespace)
				}
				if _, exists := seenNamespaces[workspace.KeyNamespacePrefix]; exists {
					addIssue("%s: duplicate key_namespace_prefix %q", sourcePath, workspace.KeyNamespacePrefix)
				}
				seenNamespaces[workspace.KeyNamespacePrefix] = struct{}{}
			}
			if len(workspace.AllowedModels) == 0 {
				addIssue("%s: organizations[%d].workspaces[%d].allowed_models must not be empty", sourcePath, orgIndex, workspaceIndex)
			}
			if len(workspace.RoleBindings) == 0 {
				addIssue("%s: organizations[%d].workspaces[%d].role_bindings must not be empty", sourcePath, orgIndex, workspaceIndex)
			}
			if design.Chargeback.RequireWorkspaceCostCenter && strings.TrimSpace(workspace.Chargeback.CostCenter) == "" {
				addIssue("%s: organizations[%d].workspaces[%d].chargeback.cost_center must not be blank when chargeback.require_workspace_cost_center=true", sourcePath, orgIndex, workspaceIndex)
			}
			if strings.TrimSpace(workspace.Chargeback.BillTo) == "" {
				addIssue("%s: organizations[%d].workspaces[%d].chargeback.bill_to must not be blank", sourcePath, orgIndex, workspaceIndex)
			}

			workspaceReachable := workspaceRoleModelReachable(workspace.RoleBindings, opts.RoleModels)
			for modelIndex, model := range workspace.AllowedModels {
				trimmed := strings.TrimSpace(model)
				if trimmed == "" {
					addIssue("%s: organizations[%d].workspaces[%d].allowed_models[%d] must not be blank", sourcePath, orgIndex, workspaceIndex, modelIndex)
					continue
				}
				if len(opts.KnownModels) > 0 {
					if _, ok := opts.KnownModels[trimmed]; !ok {
						addIssue("%s: organizations[%d].workspaces[%d].allowed_models[%d] references unknown model alias %q", sourcePath, orgIndex, workspaceIndex, modelIndex, trimmed)
					}
				}
				if len(opts.RoleModels) > 0 {
					if _, ok := workspaceReachable[wildcardModel]; !ok {
						if _, ok := workspaceReachable[trimmed]; !ok {
							addIssue("%s: organizations[%d].workspaces[%d].allowed_models[%d] %q is not reachable by any bound role", sourcePath, orgIndex, workspaceIndex, modelIndex, trimmed)
						}
					}
				}
			}

			for bindingIndex, binding := range workspace.RoleBindings {
				if strings.TrimSpace(binding.Role) == "" {
					addIssue("%s: organizations[%d].workspaces[%d].role_bindings[%d].role must not be blank", sourcePath, orgIndex, workspaceIndex, bindingIndex)
				}
				if binding.Scope != IsolationBoundaryWorkspace {
					addIssue("%s: organizations[%d].workspaces[%d].role_bindings[%d].scope must be %q", sourcePath, orgIndex, workspaceIndex, bindingIndex, IsolationBoundaryWorkspace)
				}
				if len(opts.KnownRoles) > 0 {
					if _, ok := opts.KnownRoles[binding.Role]; !ok {
						addIssue("%s: organizations[%d].workspaces[%d].role_bindings[%d].role references unknown RBAC role %q", sourcePath, orgIndex, workspaceIndex, bindingIndex, binding.Role)
					}
				}
				if len(binding.Subjects) == 0 {
					addIssue("%s: organizations[%d].workspaces[%d].role_bindings[%d].subjects must not be empty", sourcePath, orgIndex, workspaceIndex, bindingIndex)
				}
				for subjectIndex, subject := range binding.Subjects {
					if strings.TrimSpace(subject.Name) == "" {
						addIssue("%s: organizations[%d].workspaces[%d].role_bindings[%d].subjects[%d].name must not be blank", sourcePath, orgIndex, workspaceIndex, bindingIndex, subjectIndex)
					}
				}
			}
		}
	}

	sort.Strings(issues)
	return issues
}

const wildcardModel = "all"

func workspaceRoleModelReachable(bindings []RoleBinding, roleModels map[string]map[string]struct{}) map[string]struct{} {
	reachable := make(map[string]struct{})
	for _, binding := range bindings {
		models, ok := roleModels[binding.Role]
		if !ok {
			continue
		}
		for model := range models {
			reachable[model] = struct{}{}
		}
	}
	return reachable
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
