// Package tenant defines the tracked design-time multi-tenant contract.
//
// Purpose:
//   - Provide reusable workspace lookup and alias-token helpers for workflows
//     that consume the tracked tenant design package.
//
// Responsibilities:
//   - Resolve one organization/workspace pair from the loaded design.
//   - Normalize workspace chargeback cost centers into alias-safe `__cc-<n>`
//     tokens reused by existing chargeback flows.
//
// Scope:
//   - Design-package lookup and alias-token helpers only.
//
// Usage:
//   - Used by key-generation and tenant-validation workflows.
//
// Invariants/Assumptions:
//   - Helpers stay design-driven and deterministic.
//   - Normalized cost-center tokens are digits only for compatibility with the
//     tracked alias-parsing contract.
package tenant

import (
	"fmt"
	"regexp"
	"strings"
)

var costCenterAliasTokenPattern = regexp.MustCompile(`^(?:cc-)?([0-9]+)$`)

// WorkspaceSelection captures one resolved organization/workspace pair.
type WorkspaceSelection struct {
	Organization Organization `json:"organization"`
	Workspace    Workspace    `json:"workspace"`
}

// LookupOrganization resolves one organization from the design.
func (d Design) LookupOrganization(organizationID string) (Organization, error) {
	orgID := strings.TrimSpace(organizationID)
	for _, org := range d.Organizations {
		if org.ID == orgID {
			return org, nil
		}
	}
	return Organization{}, fmt.Errorf("organization %q not found", orgID)
}

// LookupWorkspace resolves one workspace under the named organization.
func (d Design) LookupWorkspace(organizationID string, workspaceID string) (WorkspaceSelection, error) {
	orgID := strings.TrimSpace(organizationID)
	wsID := strings.TrimSpace(workspaceID)
	for _, org := range d.Organizations {
		if org.ID != orgID {
			continue
		}
		for _, workspace := range org.Workspaces {
			if workspace.ID == wsID {
				return WorkspaceSelection{Organization: org, Workspace: workspace}, nil
			}
		}
		return WorkspaceSelection{}, fmt.Errorf("workspace %q not found under organization %q", wsID, orgID)
	}
	return WorkspaceSelection{}, fmt.Errorf("organization %q not found", orgID)
}

// NormalizeChargebackCostCenterToken converts tracked cost-center values into
// the digits-only token expected by the existing `__cc-<digits>` alias shape.
func NormalizeChargebackCostCenterToken(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	matches := costCenterAliasTokenPattern.FindStringSubmatch(trimmed)
	if len(matches) != 2 {
		return "", fmt.Errorf("cost center %q must match cc-<digits> or <digits>", strings.TrimSpace(raw))
	}
	return matches[1], nil
}
