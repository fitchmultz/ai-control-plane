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
	"slices"
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
	Key   gateway.KeyInfo `json:"key"`
	Usage KeyUsage        `json:"usage"`
}

// RotationRequest captures operator intent for key rotation.
type RotationRequest struct {
	SourceAlias      string  `json:"source_alias"`
	ReplacementAlias string  `json:"replacement_alias,omitempty"`
	Budget           float64 `json:"budget,omitempty"`
	RPM              int     `json:"rpm,omitempty"`
	TPM              int     `json:"tpm,omitempty"`
	Parallel         int     `json:"parallel,omitempty"`
	Duration         string  `json:"duration,omitempty"`
	Role             string  `json:"role,omitempty"`
	ReportMonth      string  `json:"report_month,omitempty"`
	DryRun           bool    `json:"dry_run,omitempty"`
	RevokeOld        bool    `json:"revoke_old,omitempty"`
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
func InspectKey(ctx context.Context, inventory Inventory, usageStore UsageStore, alias string, reportMonth string, now time.Time) (Inspection, error) {
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
		Key:   current,
		Usage: usage,
	}, nil
}

// RotateKey stages or executes a replacement-key cutover for the selected alias.
func RotateKey(ctx context.Context, inventory Inventory, usageStore UsageStore, req RotationRequest, now time.Time) (RotationResult, error) {
	inspection, err := InspectKey(ctx, inventory, usageStore, req.SourceAlias, req.ReportMonth, now)
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

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = InferRole(inspection.Key.Models)
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
		models = GetModelsForRole(role)
	}

	plan, err := PlanGenerateRequest(GenerateRequestConfig{
		Alias:    replacementAlias,
		Budget:   budget,
		RPM:      rpm,
		TPM:      tpm,
		Parallel: parallel,
		Duration: duration,
		Role:     role,
		Models:   models,
	})
	if err != nil {
		return RotationResult{}, err
	}

	result := RotationResult{
		Original:        inspection,
		ReplacementPlan: plan,
		StageInstructions: []string{
			fmt.Sprintf("Distribute the new secret for alias %q to consumers.", replacementAlias),
			fmt.Sprintf("Verify cutover with: make key-inspect ALIAS=%s", replacementAlias),
			fmt.Sprintf("Review old-key drift with: make key-inspect ALIAS=%s", req.SourceAlias),
			fmt.Sprintf("When consumers have migrated, revoke the old key with: make key-revoke ALIAS=%s", req.SourceAlias),
		},
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
func InferRole(models []string) string {
	normalized := normalizeModels(models)
	if len(normalized) == 0 {
		return "auditor"
	}

	for _, role := range ValidRoles() {
		candidate := normalizeModels(GetModelsForRole(role))
		if slices.Equal(normalized, candidate) {
			return role
		}
	}

	for _, role := range []string{"developer", "team-lead", "admin"} {
		candidate := normalizeModels(GetModelsForRole(role))
		if isSubset(normalized, candidate) {
			return role
		}
	}

	return "developer"
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
	return strings.TrimSpace(sourceAlias) + "-rotated-" + now.UTC().Format("20060102150405")
}

func normalizeModels(models []string) []string {
	result := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	slices.Sort(result)
	return result
}

func isSubset(current []string, candidate []string) bool {
	candidateSet := make(map[string]struct{}, len(candidate))
	for _, model := range candidate {
		candidateSet[model] = struct{}{}
	}
	for _, model := range current {
		if _, ok := candidateSet[model]; !ok {
			return false
		}
	}
	return true
}
