// Package keygen provides typed virtual-key lifecycle workflows.
//
// Purpose:
//   - Centralize inspection and rotation planning for ACP virtual keys.
//
// Responsibilities:
//   - Resolve report-month windows for usage inspection.
//   - Inspect current key inventory plus month-scoped usage.
//   - Stage replacement key cutovers with consistent defaults and messaging.
//
// Scope:
//   - Key lifecycle planning only.
//
// Usage:
//   - Used by `cmd/acpctl` key lifecycle commands.
//
// Invariants/Assumptions:
//   - Key aliases are validated before use.
//   - Rotation preserves current models when available.
package keygen

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
)

// Inventory captures the gateway operations required for key lifecycle workflows.
type Inventory interface {
	ListKeys(context.Context) ([]gateway.KeyInfo, error)
	GenerateKey(context.Context, *gateway.GenerateKeyRequest) (*gateway.GenerateKeyResponse, error)
	DeleteKey(context.Context, string) error
}

// UsageStore captures month-scoped usage lookup for an individual key alias.
type UsageStore interface {
	KeyUsage(context.Context, string, MonthWindow) (KeyUsage, error)
}

// MonthWindow captures the canonical reporting window for a usage request.
type MonthWindow struct {
	ReportMonth string `json:"report_month"`
	Start       string `json:"start"`
	End         string `json:"end"`
}

// ModelUsage captures per-model usage totals for a key.
type ModelUsage struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TokenCount   int64   `json:"token_count"`
	SpendAmount  float64 `json:"spend_amount"`
}

// KeyUsage captures month-scoped usage totals for a key alias.
type KeyUsage struct {
	Alias         string       `json:"alias"`
	ReportMonth   string       `json:"report_month"`
	TotalSpend    float64      `json:"total_spend"`
	TotalRequests int64        `json:"total_requests"`
	TotalTokens   int64        `json:"total_tokens"`
	LastSeen      string       `json:"last_seen,omitempty"`
	ByModel       []ModelUsage `json:"by_model,omitempty"`
}

// Inspection combines the current key inventory entry with month-scoped usage.
type Inspection struct {
	Key         gateway.KeyInfo    `json:"key"`
	Usage       KeyUsage           `json:"usage"`
	TenantScope *TenantAccessScope `json:"tenant_scope,omitempty"`
}

// RotationRequest captures operator intent for key rotation.
type RotationRequest struct {
	SourceAlias      string             `json:"source_alias"`
	ReplacementAlias string             `json:"replacement_alias,omitempty"`
	Budget           float64            `json:"budget,omitempty"`
	RPM              int                `json:"rpm,omitempty"`
	TPM              int                `json:"tpm,omitempty"`
	Parallel         int                `json:"parallel,omitempty"`
	Duration         string             `json:"duration,omitempty"`
	Role             string             `json:"role,omitempty"`
	ReportMonth      string             `json:"report_month,omitempty"`
	DryRun           bool               `json:"dry_run,omitempty"`
	RevokeOld        bool               `json:"revoke_old,omitempty"`
	TenantScope      *TenantAccessScope `json:"tenant_scope,omitempty"`
	RepoRoot         string             `json:"-"`
	TenantConfigPath string             `json:"tenant_config_path,omitempty"`
}

// RotationResult captures the planned and optional executed replacement state.
type RotationResult struct {
	Original          Inspection                   `json:"original"`
	ReplacementPlan   GenerateRequestPlan          `json:"replacement_plan"`
	Replacement       *gateway.GenerateKeyResponse `json:"replacement,omitempty"`
	RevokedOld        bool                         `json:"revoked_old,omitempty"`
	StageInstructions []string                     `json:"stage_instructions"`
}

// ResolveMonthWindow validates a report month and expands it to an inclusive range.
func ResolveMonthWindow(reportMonth string, now time.Time) (MonthWindow, error) {
	month := strings.TrimSpace(reportMonth)
	if month == "" {
		month = now.UTC().Format("2006-01")
	}
	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return MonthWindow{}, fmt.Errorf("invalid report month %q: use YYYY-MM", month)
	}
	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := start.AddDate(0, 1, 0)
	end := nextMonth.AddDate(0, 0, -1)
	return MonthWindow{
		ReportMonth: start.Format("2006-01"),
		Start:       start.Format("2006-01-02"),
		End:         end.Format("2006-01-02"),
	}, nil
}

// InspectKey loads the current key inventory entry plus month-scoped usage.
func InspectKey(ctx context.Context, inventory Inventory, usageStore UsageStore, alias string, reportMonth string, now time.Time, tenantScope *TenantAccessScope) (Inspection, error) {
	if inventory == nil {
		return Inspection{}, fmt.Errorf("key inventory is required")
	}
	if usageStore == nil {
		return Inspection{}, fmt.Errorf("key usage store is required")
	}
	alias = strings.TrimSpace(alias)
	if err := ValidateAlias(alias); err != nil {
		return Inspection{}, err
	}
	if err := RequireAliasInTenantScope(alias, tenantScope); err != nil {
		return Inspection{}, err
	}

	window, err := ResolveMonthWindow(reportMonth, now)
	if err != nil {
		return Inspection{}, err
	}

	keys, err := inventory.ListKeys(ctx)
	if err != nil {
		return Inspection{}, fmt.Errorf("list keys: %w", err)
	}
	current, err := findKey(keys, alias)
	if err != nil {
		return Inspection{}, err
	}

	usage, err := usageStore.KeyUsage(ctx, alias, window)
	if err != nil {
		return Inspection{}, fmt.Errorf("lookup key usage: %w", err)
	}
	if strings.TrimSpace(usage.Alias) == "" {
		usage.Alias = alias
	}
	if strings.TrimSpace(usage.ReportMonth) == "" {
		usage.ReportMonth = window.ReportMonth
	}

	return Inspection{
		Key:         current,
		Usage:       usage,
		TenantScope: cloneTenantAccessScope(tenantScope),
	}, nil
}

// RotateKey stages or executes a replacement-key cutover for the selected alias.
func RotateKey(ctx context.Context, inventory Inventory, usageStore UsageStore, req RotationRequest, now time.Time) (RotationResult, error) {
	inspection, err := InspectKey(ctx, inventory, usageStore, req.SourceAlias, req.ReportMonth, now, req.TenantScope)
	if err != nil {
		return RotationResult{}, err
	}

	replacementAlias := strings.TrimSpace(req.ReplacementAlias)
	if replacementAlias == "" {
		replacementAlias = defaultReplacementAlias(req.SourceAlias, now)
	}
	if err := ValidateAlias(replacementAlias); err != nil {
		return RotationResult{}, err
	}
	if err := ValidateReplacementAliasForTenantScope(req.SourceAlias, replacementAlias, req.TenantScope); err != nil {
		return RotationResult{}, err
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		inferredRole, err := InferRole(inspection.Key.Models)
		if err != nil {
			return RotationResult{}, err
		}
		role = inferredRole
	}

	duration := strings.TrimSpace(req.Duration)
	if duration == "" {
		duration = strings.TrimSpace(inspection.Key.BudgetDuration)
	}
	if duration == "" {
		duration = DefaultConfig().Duration
	}

	budget := req.Budget
	if budget <= 0 {
		budget = inspection.Key.MaxBudget
	}

	rpm := req.RPM
	if rpm <= 0 {
		rpm = inspection.Key.RPMLimit
	}
	tpm := req.TPM
	if tpm <= 0 {
		tpm = inspection.Key.TPMLimit
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = inspection.Key.MaxParallelRequests
	}

	models := inspection.Key.Models
	if len(models) == 0 {
		resolvedModels, err := GetModelsForRole(role)
		if err != nil {
			return RotationResult{}, err
		}
		models = resolvedModels
	}

	planAlias := replacementAlias
	if req.TenantScope != nil {
		planAlias = tenantRotationAliasSuffix(replacementAlias, req.TenantScope)
	}
	planConfig := GenerateRequestConfig{
		Alias:    planAlias,
		Budget:   budget,
		RPM:      rpm,
		TPM:      tpm,
		Parallel: parallel,
		Duration: duration,
		Role:     role,
		Models:   models,
	}
	if req.TenantScope != nil {
		organizationID, workspaceID, err := resolveRotationTenantPlanningScope(ctx, req)
		if err != nil {
			return RotationResult{}, err
		}
		planConfig.RepoRoot = req.RepoRoot
		planConfig.TenantConfigPath = req.TenantConfigPath
		planConfig.OrganizationID = organizationID
		planConfig.WorkspaceID = workspaceID
	}

	plan, err := PlanGenerateRequest(planConfig)
	if err != nil {
		return RotationResult{}, err
	}

	result := RotationResult{
		Original:          inspection,
		ReplacementPlan:   plan,
		StageInstructions: buildRotationStageInstructions(req, replacementAlias),
	}
	if req.DryRun {
		return result, nil
	}

	replacement, err := inventory.GenerateKey(ctx, &plan.Request)
	if err != nil {
		return RotationResult{}, fmt.Errorf("generate replacement key: %w", err)
	}
	result.Replacement = replacement

	if req.RevokeOld {
		if err := inventory.DeleteKey(ctx, req.SourceAlias); err != nil {
			return RotationResult{}, fmt.Errorf("revoke old key: %w", err)
		}
		result.RevokedOld = true
	}

	return result, nil
}

// InferRole picks the least-privileged canonical role that matches the model set.
func tenantRotationAliasSuffix(alias string, scope *TenantAccessScope) string {
	trimmed := strings.TrimSpace(alias)
	if scope == nil {
		return trimmed
	}
	prefix := scope.matchingNamespacePrefix(trimmed)
	if prefix == "" {
		return trimmed
	}
	remainder := strings.TrimPrefix(trimmed, prefix+"--")
	base, _, ok := strings.Cut(remainder, "__")
	if ok {
		return base
	}
	return remainder
}

func resolveRotationTenantPlanningScope(ctx context.Context, req RotationRequest) (string, string, error) {
	if req.TenantScope == nil {
		return "", "", nil
	}
	organizationID := strings.TrimSpace(req.TenantScope.OrganizationID)
	workspaceID := strings.TrimSpace(req.TenantScope.WorkspaceID)
	if organizationID == "" {
		return "", "", &ValidationError{Field: "organization", Message: "organization is required for tenant-scoped rotation"}
	}
	if workspaceID != "" {
		return organizationID, workspaceID, nil
	}
	design, err := loadTenantDesign(ctx, req.RepoRoot, req.TenantConfigPath)
	if err != nil {
		return "", "", err
	}
	organization, err := design.LookupOrganization(organizationID)
	if err != nil {
		return "", "", err
	}
	prefix := req.TenantScope.matchingNamespacePrefix(req.SourceAlias)
	if prefix == "" {
		return "", "", RequireAliasInTenantScope(req.SourceAlias, req.TenantScope)
	}
	for _, workspace := range organization.Workspaces {
		if strings.TrimSpace(workspace.KeyNamespacePrefix) == prefix {
			return organization.ID, workspace.ID, nil
		}
	}
	return "", "", fmt.Errorf("organization %q has no workspace for namespace prefix %q", organization.ID, prefix)
}

func buildRotationStageInstructions(req RotationRequest, replacementAlias string) []string {
	makeScopeArgs := ""
	if req.TenantScope != nil {
		if organizationID := strings.TrimSpace(req.TenantScope.OrganizationID); organizationID != "" {
			makeScopeArgs += fmt.Sprintf(" ORG=%s", organizationID)
		}
		if workspaceID := strings.TrimSpace(req.TenantScope.WorkspaceID); workspaceID != "" {
			makeScopeArgs += fmt.Sprintf(" WORKSPACE=%s", workspaceID)
		}
		if trimmed := strings.TrimSpace(req.TenantConfigPath); trimmed != "" {
			makeScopeArgs += fmt.Sprintf(" TENANT_FILE=%s", trimmed)
		}
	}
	return []string{
		fmt.Sprintf("Distribute the new secret for alias %q to consumers.", replacementAlias),
		fmt.Sprintf("Verify cutover with: make key-inspect ALIAS=%s%s", replacementAlias, makeScopeArgs),
		fmt.Sprintf("Review old-key drift with: make key-inspect ALIAS=%s%s", req.SourceAlias, makeScopeArgs),
		fmt.Sprintf("When consumers have migrated, revoke the old key with: make key-revoke ALIAS=%s%s", req.SourceAlias, makeScopeArgs),
	}
}

func InferRole(models []string) (string, error) {
	cfg, approvedModels, err := loadTrackedRoleContract()
	if err != nil {
		return "", fmt.Errorf("load tracked RBAC contract: %w", err)
	}
	role := strings.TrimSpace(cfg.InferLeastPrivilegedRole(models, approvedModels))
	if role == "" {
		return "", &ValidationError{Field: "role", Message: "no matching RBAC role configured"}
	}
	return role, nil
}

func findKey(keys []gateway.KeyInfo, alias string) (gateway.KeyInfo, error) {
	for _, key := range keys {
		if key.Alias() == alias {
			return key, nil
		}
	}
	return gateway.KeyInfo{}, fmt.Errorf("key alias %q not found", alias)
}

func defaultReplacementAlias(sourceAlias string, now time.Time) string {
	trimmed := strings.TrimSpace(sourceAlias)
	rotationToken := "-r" + now.UTC().Format("060102150405")
	base, suffix, ok := strings.Cut(trimmed, "__")
	suffixToken := ""
	if ok {
		suffixToken = "__" + suffix
	}
	maxBaseLength := 64 - len(rotationToken) - len(suffixToken)
	if maxBaseLength < 1 {
		maxBaseLength = 1
	}
	if len(base) > maxBaseLength {
		candidate := base[:maxBaseLength]
		if cut := strings.LastIndexAny(candidate, "-._"); cut > 0 {
			candidate = candidate[:cut]
		}
		candidate = strings.TrimRight(candidate, "-._")
		if strings.TrimSpace(candidate) != "" {
			base = candidate
		} else {
			base = base[:maxBaseLength]
		}
	}
	return base + rotationToken + suffixToken
}
