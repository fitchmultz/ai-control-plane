// Package keygen provides typed virtual-key lifecycle workflows.
//
// Purpose:
//   - Resolve and enforce tenant-safe lifecycle scope boundaries from the
//     tracked tenant design package.
//
// Responsibilities:
//   - Resolve organization/workspace namespace prefixes from tenant design YAML.
//   - Filter key inventory to the allowed namespace boundary.
//   - Enforce that inspect/rotate/revoke aliases stay inside one tenant scope.
//
// Scope:
//   - Key lifecycle tenant-scope helpers only.
//
// Usage:
//   - Used by `cmd/acpctl` key lifecycle commands and lifecycle planners.
//
// Invariants/Assumptions:
//   - Empty scope means repository-wide lifecycle access.
//   - Tenant-managed replacement aliases must preserve source attribution
//     suffixes such as `__cc-<digits>`.
package keygen

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
)

// TenantAccessScope captures one resolved lifecycle access boundary.
type TenantAccessScope struct {
	OrganizationID    string   `json:"organization_id"`
	WorkspaceID       string   `json:"workspace_id,omitempty"`
	NamespacePrefixes []string `json:"namespace_prefixes"`
}

// ResolveTenantAccessScope loads the tracked tenant design and resolves the
// namespace prefixes allowed for one lifecycle scope.
func ResolveTenantAccessScope(ctx context.Context, repoRoot string, tenantConfigPath string, organizationID string, workspaceID string) (*TenantAccessScope, error) {
	orgID := strings.TrimSpace(organizationID)
	workspace := strings.TrimSpace(workspaceID)
	if orgID == "" && workspace == "" {
		return nil, nil
	}
	if orgID == "" {
		return nil, &ValidationError{Field: "organization", Message: "organization is required when workspace is set"}
	}

	resolvedRepoRoot, err := config.NewLoader().WithRepoRoot(repoRoot).RequireRepoRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	designPath := repopath.ResolveRepoPath(resolvedRepoRoot, tenant.DefaultDesignPath)
	if trimmed := strings.TrimSpace(tenantConfigPath); trimmed != "" {
		designPath = repopath.ResolveRepoPath(resolvedRepoRoot, trimmed)
	}
	design, err := tenant.LoadFile(designPath)
	if err != nil {
		return nil, err
	}

	if workspace != "" {
		selection, err := design.LookupWorkspace(orgID, workspace)
		if err != nil {
			return nil, err
		}
		prefix := strings.TrimSpace(selection.Workspace.KeyNamespacePrefix)
		if prefix == "" {
			return nil, &ValidationError{Field: "workspace", Message: fmt.Sprintf("workspace %s/%s has no key namespace prefix", orgID, workspace)}
		}
		return &TenantAccessScope{
			OrganizationID:    selection.Organization.ID,
			WorkspaceID:       selection.Workspace.ID,
			NamespacePrefixes: []string{prefix},
		}, nil
	}

	organization, err := design.LookupOrganization(orgID)
	if err != nil {
		return nil, err
	}
	prefixes := make([]string, 0, len(organization.Workspaces))
	for _, workspace := range organization.Workspaces {
		if prefix := strings.TrimSpace(workspace.KeyNamespacePrefix); prefix != "" {
			prefixes = append(prefixes, prefix)
		}
	}
	prefixes = dedupeSortedScopePrefixes(prefixes)
	if len(prefixes) == 0 {
		return nil, &ValidationError{Field: "organization", Message: fmt.Sprintf("organization %q has no workspace key namespace prefixes", organization.ID)}
	}
	return &TenantAccessScope{
		OrganizationID:    organization.ID,
		NamespacePrefixes: prefixes,
	}, nil
}

// Display returns a stable human-readable scope label.
func (s *TenantAccessScope) Display() string {
	if s == nil {
		return ""
	}
	if strings.TrimSpace(s.WorkspaceID) != "" {
		return strings.TrimSpace(s.OrganizationID) + "/" + strings.TrimSpace(s.WorkspaceID)
	}
	return strings.TrimSpace(s.OrganizationID)
}

// MatchesAlias returns true when the alias stays inside the allowed namespace boundary.
func (s *TenantAccessScope) MatchesAlias(alias string) bool {
	return s.matchingNamespacePrefix(alias) != ""
}

func (s *TenantAccessScope) matchingNamespacePrefix(alias string) string {
	if s == nil {
		return ""
	}
	trimmed := strings.TrimSpace(alias)
	best := ""
	for _, prefix := range s.NamespacePrefixes {
		candidate := strings.TrimSpace(prefix)
		if candidate == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, candidate+"--") {
			continue
		}
		if len(candidate) > len(best) {
			best = candidate
		}
	}
	return best
}

// RequireAliasInTenantScope returns a validation error when the alias falls outside the resolved scope.
func RequireAliasInTenantScope(alias string, scope *TenantAccessScope) error {
	if scope == nil {
		return nil
	}
	if scope.MatchesAlias(alias) {
		return nil
	}
	return &ValidationError{Field: "alias", Message: fmt.Sprintf("alias %q is outside tenant scope %s", strings.TrimSpace(alias), scope.Display())}
}

// ValidateReplacementAliasForTenantScope enforces that a rotated tenant alias
// stays inside the same namespace and preserves the same attribution suffix.
func ValidateReplacementAliasForTenantScope(sourceAlias string, replacementAlias string, scope *TenantAccessScope) error {
	if scope == nil {
		return nil
	}
	sourcePrefix := scope.matchingNamespacePrefix(sourceAlias)
	if sourcePrefix == "" {
		return RequireAliasInTenantScope(sourceAlias, scope)
	}
	if replacementPrefix := scope.matchingNamespacePrefix(replacementAlias); replacementPrefix != sourcePrefix {
		return &ValidationError{Field: "replacement_alias", Message: fmt.Sprintf("replacement alias %q must stay in namespace %q", strings.TrimSpace(replacementAlias), sourcePrefix)}
	}
	sourceSuffix := managedAliasSuffix(sourceAlias)
	if sourceSuffix != "" && managedAliasSuffix(replacementAlias) != sourceSuffix {
		return &ValidationError{Field: "replacement_alias", Message: fmt.Sprintf("replacement alias %q must preserve attribution token %q", strings.TrimSpace(replacementAlias), sourceSuffix)}
	}
	return nil
}

// FilterKeysForTenantScope filters gateway key inventory to one tenant boundary.
func FilterKeysForTenantScope(keys []gateway.KeyInfo, scope *TenantAccessScope) []gateway.KeyInfo {
	if scope == nil {
		return append([]gateway.KeyInfo(nil), keys...)
	}
	filtered := make([]gateway.KeyInfo, 0, len(keys))
	for _, key := range keys {
		if !scope.MatchesAlias(key.Alias()) {
			continue
		}
		filtered = append(filtered, key)
	}
	return filtered
}

func managedAliasSuffix(alias string) string {
	trimmed := strings.TrimSpace(alias)
	_, suffix, ok := strings.Cut(trimmed, "__")
	if !ok {
		return ""
	}
	return "__" + suffix
}

func cloneTenantAccessScope(scope *TenantAccessScope) *TenantAccessScope {
	if scope == nil {
		return nil
	}
	cloned := *scope
	cloned.NamespacePrefixes = append([]string(nil), scope.NamespacePrefixes...)
	return &cloned
}

func dedupeSortedScopePrefixes(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}
