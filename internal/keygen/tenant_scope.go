// Package keygen provides typed virtual-key lifecycle workflows.
//
// Purpose:
//   - Project the tracked tenant design package into workspace-scoped key
//     generation plans without overstating broader multi-tenant runtime support.
//
// Responsibilities:
//   - Resolve tracked organization/workspace definitions from the tenant design.
//   - Derive canonical workspace-scoped key aliases and model scopes.
//   - Enforce that tenant-scoped key requests stay inside bound roles and the
//     workspace allowlist.
//
// Scope:
//   - Tenant-aware request planning only.
//
// Usage:
//   - Used by `PlanGenerateRequest` when `OrganizationID` and `WorkspaceID` are
//     provided.
//
// Invariants/Assumptions:
//   - Tenant-scoped aliases reuse the existing `__cc-<n>` attribution token and
//     keep workspace identity in the tracked namespace prefix.
//   - This helper does not imply shared-runtime query/report isolation exists.
package keygen

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/rbac"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
)

// TenantScope captures the workspace boundary applied to a key request plan.
type TenantScope struct {
	OrganizationID  string `json:"organization_id"`
	WorkspaceID     string `json:"workspace_id"`
	NamespacePrefix string `json:"namespace_prefix"`
	CostCenterToken string `json:"cost_center_token"`
}

func hasTenantScope(cfg GenerateRequestConfig) bool {
	return strings.TrimSpace(cfg.OrganizationID) != "" || strings.TrimSpace(cfg.WorkspaceID) != ""
}

func planTenantGenerateRequest(cfg GenerateRequestConfig) (GenerateRequestPlan, error) {
	organizationID := strings.TrimSpace(cfg.OrganizationID)
	workspaceID := strings.TrimSpace(cfg.WorkspaceID)
	if organizationID == "" || workspaceID == "" {
		return GenerateRequestPlan{}, &ValidationError{Field: "workspace", Message: "both organization and workspace are required for tenant-scoped key generation"}
	}

	repoRoot, err := resolveKeyRepoRoot(context.Background(), cfg.RepoRoot)
	if err != nil {
		return GenerateRequestPlan{}, err
	}
	design, err := loadTenantDesign(context.Background(), repoRoot, cfg.TenantConfigPath)
	if err != nil {
		return GenerateRequestPlan{}, err
	}
	selection, err := design.LookupWorkspace(organizationID, workspaceID)
	if err != nil {
		return GenerateRequestPlan{}, err
	}

	roleConfig, approvedModels, err := loadTrackedRoleContractFromRepo(repoRoot)
	if err != nil {
		return GenerateRequestPlan{}, fmt.Errorf("load tracked RBAC contract: %w", err)
	}
	role, err := resolveTenantWorkspaceRole(cfg.Role, selection.Workspace, roleConfig, approvedModels)
	if err != nil {
		return GenerateRequestPlan{}, err
	}

	workspaceModels := dedupeSortedStrings(selection.Workspace.AllowedModels)
	roleModels := dedupeSortedStrings(roleConfig.ModelsForRole(role, approvedModels))
	models := append([]string(nil), roleModels...)
	if len(cfg.Models) > 0 {
		models = dedupeSortedStrings(cfg.Models)
		if outsideRole := differenceStrings(models, roleModels); len(outsideRole) > 0 {
			return GenerateRequestPlan{}, &ValidationError{Field: "models", Message: fmt.Sprintf("requested models exceed role %q access: %s", role, strings.Join(outsideRole, ", "))}
		}
		if outsideWorkspace := differenceStrings(models, workspaceModels); len(outsideWorkspace) > 0 {
			return GenerateRequestPlan{}, &ValidationError{Field: "models", Message: fmt.Sprintf("requested models exceed workspace %s/%s allowlist: %s", organizationID, workspaceID, strings.Join(outsideWorkspace, ", "))}
		}
	} else {
		models = intersectStrings(models, workspaceModels)
	}
	if len(models) == 0 {
		return GenerateRequestPlan{}, &ValidationError{Field: "role", Message: fmt.Sprintf("role %q does not authorize any models for workspace %s/%s", role, organizationID, workspaceID)}
	}

	alias, scope, err := buildTenantScopedAlias(cfg.Alias, selection)
	if err != nil {
		return GenerateRequestPlan{}, err
	}
	if err := ValidateAlias(alias); err != nil {
		return GenerateRequestPlan{}, err
	}

	duration := strings.TrimSpace(cfg.Duration)
	if duration == "" {
		duration = DefaultConfig().Duration
	}

	request := gateway.GenerateKeyRequest{
		KeyAlias:       alias,
		MaxBudget:      cfg.Budget,
		BudgetDuration: duration,
		Models:         append([]string(nil), models...),
	}
	if cfg.RPM > 0 {
		request.RPMLimit = cfg.RPM
	}
	if cfg.TPM > 0 {
		request.TPMLimit = cfg.TPM
	}
	if cfg.Parallel > 0 {
		request.MaxParallelRequests = cfg.Parallel
	}

	return GenerateRequestPlan{
		Request:     request,
		Role:        role,
		Models:      append([]string(nil), models...),
		TenantScope: &scope,
	}, nil
}

func resolveTenantWorkspaceRole(explicitRole string, workspace tenant.Workspace, roleConfig rbac.Config, approvedModels []string) (string, error) {
	boundRoles := workspaceBoundRoles(workspace)
	if explicit := strings.TrimSpace(explicitRole); explicit != "" {
		if !roleConfig.HasRole(explicit) {
			return "", &ValidationError{Field: "role", Message: fmt.Sprintf("invalid role: %s", explicit)}
		}
		if !containsTrimmed(boundRoles, explicit) {
			return "", &ValidationError{Field: "role", Message: fmt.Sprintf("role %q is not bound to workspace %q", explicit, workspace.ID)}
		}
		if !roleConfig.Roles[explicit].CanCreateKeys {
			return "", &ValidationError{Field: "role", Message: fmt.Sprintf("role %q cannot create keys", explicit)}
		}
		return explicit, nil
	}

	candidates := workspaceBoundRolesByPrivilege(boundRoles, roleConfig, approvedModels)
	for _, candidate := range candidates {
		if !roleConfig.Roles[candidate].CanCreateKeys {
			continue
		}
		if roleCoversModels(roleConfig.ModelsForRole(candidate, approvedModels), workspace.AllowedModels) {
			return candidate, nil
		}
	}
	return "", &ValidationError{Field: "role", Message: fmt.Sprintf("no workspace-bound role can create a key covering all allowed models for workspace %q", workspace.ID)}
}

func workspaceBoundRoles(workspace tenant.Workspace) []string {
	seen := make(map[string]struct{}, len(workspace.RoleBindings))
	roles := make([]string, 0, len(workspace.RoleBindings))
	for _, binding := range workspace.RoleBindings {
		role := strings.TrimSpace(binding.Role)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
}

func workspaceBoundRolesByPrivilege(boundRoles []string, roleConfig rbac.Config, approvedModels []string) []string {
	roles := append([]string(nil), boundRoles...)
	sort.SliceStable(roles, func(i, j int) bool {
		left := roleConfig.Roles[roles[i]]
		right := roleConfig.Roles[roles[j]]

		leftModels := len(roleConfig.ModelsForRole(roles[i], approvedModels))
		rightModels := len(roleConfig.ModelsForRole(roles[j], approvedModels))
		if leftModels != rightModels {
			return leftModels < rightModels
		}
		if left.CanCreateKeys != right.CanCreateKeys {
			return !left.CanCreateKeys && right.CanCreateKeys
		}
		if left.CanApprove != right.CanApprove {
			return !left.CanApprove && right.CanApprove
		}
		if left.CanAssignRoles != right.CanAssignRoles {
			return !left.CanAssignRoles && right.CanAssignRoles
		}
		if left.BudgetCeiling != right.BudgetCeiling {
			return left.BudgetCeiling < right.BudgetCeiling
		}
		return roles[i] < roles[j]
	})
	return roles
}

func roleCoversModels(roleModels []string, requiredModels []string) bool {
	roleSet := make(map[string]struct{}, len(roleModels))
	for _, model := range roleModels {
		roleSet[strings.TrimSpace(model)] = struct{}{}
	}
	for _, model := range requiredModels {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, ok := roleSet[trimmed]; !ok {
			return false
		}
	}
	return true
}

func buildTenantScopedAlias(aliasSuffix string, selection tenant.WorkspaceSelection) (string, TenantScope, error) {
	subject := strings.TrimSpace(aliasSuffix)
	if strings.Contains(subject, "__") {
		return "", TenantScope{}, &ValidationError{Field: "alias", Message: "workspace-scoped alias suffix must not include __ tokens; ACP derives tenant/team/cost-center segments"}
	}
	costCenterToken, err := tenant.NormalizeChargebackCostCenterToken(selection.Workspace.Chargeback.CostCenter)
	if err != nil {
		return "", TenantScope{}, &ValidationError{Field: "workspace", Message: err.Error()}
	}
	scope := TenantScope{
		OrganizationID:  selection.Organization.ID,
		WorkspaceID:     selection.Workspace.ID,
		NamespacePrefix: selection.Workspace.KeyNamespacePrefix,
		CostCenterToken: costCenterToken,
	}
	alias := fmt.Sprintf("%s--%s__cc-%s", scope.NamespacePrefix, subject, scope.CostCenterToken)
	return alias, scope, nil
}

func containsTrimmed(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(want) {
			return true
		}
	}
	return false
}

func intersectStrings(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[strings.TrimSpace(value)] = struct{}{}
	}
	result := make([]string, 0, len(left))
	for _, value := range left {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := rightSet[trimmed]; ok {
			result = append(result, trimmed)
		}
	}
	return dedupeSortedStrings(result)
}

func differenceStrings(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[strings.TrimSpace(value)] = struct{}{}
	}
	result := make([]string, 0)
	for _, value := range left {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := rightSet[trimmed]; !ok {
			result = append(result, trimmed)
		}
	}
	return dedupeSortedStrings(result)
}

func dedupeSortedStrings(values []string) []string {
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
