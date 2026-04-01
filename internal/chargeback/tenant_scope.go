// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Resolve tenant-safe report scopes from the tracked tenant design package.
//
// Responsibilities:
//   - Map chargeback CLI scope inputs onto tracked organization/workspace
//     boundaries.
//   - Enforce that scoped reports only use tenant-safe metadata-only report
//     definitions declared in the design package.
//   - Derive database query filters and archive-safe scope labels.
//
// Scope:
//   - Chargeback report scope resolution only.
//
// Usage:
//   - Used by report input normalization before database composition.
//
// Invariants/Assumptions:
//   - Scoped reporting reuses tracked key namespace prefixes.
//   - Global reports remain available for single-tenant operator workflows.
package chargeback

import (
	"fmt"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
)

func resolveReportScope(repoRoot string, command ReportCommandInput) (ReportScope, *db.ChargebackQueryScope, error) {
	organizationID := strings.TrimSpace(command.OrganizationID)
	workspaceID := strings.TrimSpace(command.WorkspaceID)
	if organizationID == "" && workspaceID == "" {
		return ReportScope{
			Kind:        ReportScopeGlobal,
			Label:       "global",
			Aggregation: "global",
		}, nil, nil
	}
	if organizationID == "" {
		return ReportScope{}, nil, fmt.Errorf("--organization is required when --workspace is set")
	}
	if strings.TrimSpace(repoRoot) == "" {
		return ReportScope{}, nil, fmt.Errorf("repository root is required for tenant-scoped chargeback reports")
	}

	designPath := repopath.ResolveRepoPath(repoRoot, tenant.DefaultDesignPath)
	if trimmed := strings.TrimSpace(command.TenantFile); trimmed != "" {
		designPath = repopath.ResolveRepoPath(repoRoot, trimmed)
	}
	design, err := tenant.LoadFile(designPath)
	if err != nil {
		return ReportScope{}, nil, err
	}

	if workspaceID != "" {
		if err := requireTenantSafeAggregation(design, tenant.AggregationLevelWorkspace); err != nil {
			return ReportScope{}, nil, err
		}
		selection, err := design.LookupWorkspace(organizationID, workspaceID)
		if err != nil {
			return ReportScope{}, nil, err
		}
		return ReportScope{
			Kind:           ReportScopeWorkspace,
			Label:          fmt.Sprintf("workspace/%s/%s", organizationID, workspaceID),
			Aggregation:    string(tenant.AggregationLevelWorkspace),
			OrganizationID: organizationID,
			WorkspaceID:    workspaceID,
			ArchiveSuffix:  scopeArchiveSuffix(string(ReportScopeWorkspace), organizationID, workspaceID),
		}, &db.ChargebackQueryScope{NamespacePrefixes: []string{selection.Workspace.KeyNamespacePrefix}}, nil
	}

	if err := requireTenantSafeAggregation(design, tenant.AggregationLevelOrganization); err != nil {
		return ReportScope{}, nil, err
	}
	organization, err := design.LookupOrganization(organizationID)
	if err != nil {
		return ReportScope{}, nil, err
	}
	prefixes := make([]string, 0, len(organization.Workspaces))
	for _, workspace := range organization.Workspaces {
		prefix := strings.TrimSpace(workspace.KeyNamespacePrefix)
		if prefix != "" {
			prefixes = append(prefixes, prefix)
		}
	}
	return ReportScope{
		Kind:           ReportScopeOrganization,
		Label:          fmt.Sprintf("organization/%s", organizationID),
		Aggregation:    string(tenant.AggregationLevelOrganization),
		OrganizationID: organizationID,
		ArchiveSuffix:  scopeArchiveSuffix(string(ReportScopeOrganization), organizationID),
	}, &db.ChargebackQueryScope{NamespacePrefixes: prefixes}, nil
}

func requireTenantSafeAggregation(design tenant.Design, aggregation tenant.AggregationLevel) error {
	if !design.Reporting.TenantSafeByDefault {
		return fmt.Errorf("tenant design must keep reporting.tenant_safe_by_default=true for scoped chargeback reports")
	}
	if !design.Reporting.MetadataOnly {
		return fmt.Errorf("tenant design must keep reporting.metadata_only=true for scoped chargeback reports")
	}
	if !supportsAggregation(design.Reporting.AllowedAggregationLevels, aggregation) {
		return fmt.Errorf("tenant design does not allow %q scoped reporting", aggregation)
	}
	for _, definition := range design.Reporting.ReportDefinitions {
		if definition.Aggregation != aggregation {
			continue
		}
		if definition.Redaction != tenant.RedactionLevelMetadataOnly {
			continue
		}
		if definition.IncludeCrossOrganizationData {
			continue
		}
		return nil
	}
	return fmt.Errorf("tenant design does not define a metadata-only %q report shape", aggregation)
}

func supportsAggregation(levels []tenant.AggregationLevel, want tenant.AggregationLevel) bool {
	for _, level := range levels {
		if level == want {
			return true
		}
	}
	return false
}

func normalizedReportScope(scope ReportScope) ReportScope {
	if scope.Kind == "" {
		scope.Kind = ReportScopeGlobal
	}
	if strings.TrimSpace(scope.Label) == "" {
		switch scope.Kind {
		case ReportScopeWorkspace:
			scope.Label = fmt.Sprintf("workspace/%s/%s", scope.OrganizationID, scope.WorkspaceID)
		case ReportScopeOrganization:
			scope.Label = fmt.Sprintf("organization/%s", scope.OrganizationID)
		default:
			scope.Label = "global"
		}
	}
	if strings.TrimSpace(scope.Aggregation) == "" {
		switch scope.Kind {
		case ReportScopeWorkspace:
			scope.Aggregation = string(tenant.AggregationLevelWorkspace)
		case ReportScopeOrganization:
			scope.Aggregation = string(tenant.AggregationLevelOrganization)
		default:
			scope.Aggregation = "global"
		}
	}
	if strings.TrimSpace(scope.ArchiveSuffix) == "" {
		switch scope.Kind {
		case ReportScopeWorkspace:
			scope.ArchiveSuffix = scopeArchiveSuffix(string(scope.Kind), scope.OrganizationID, scope.WorkspaceID)
		case ReportScopeOrganization:
			scope.ArchiveSuffix = scopeArchiveSuffix(string(scope.Kind), scope.OrganizationID)
		}
	}
	return scope
}

func scopeArchiveSuffix(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(part))
		if trimmed == "" {
			continue
		}
		trimmed = strings.ReplaceAll(trimmed, "/", "-")
		trimmed = strings.ReplaceAll(trimmed, "_", "-")
		cleaned = append(cleaned, trimmed)
	}
	return strings.Join(cleaned, "-")
}
