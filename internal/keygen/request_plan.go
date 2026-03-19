// request_plan.go - Canonical key generation request planning.
//
// Purpose:
//   - Centralize role resolution, validation, model selection, and gateway
//     request assembly for key generation workflows.
//
// Responsibilities:
//   - Normalize typed key-generation inputs.
//   - Build canonical gateway key-generation requests.
//   - Preserve consistent request semantics across CLI and onboarding flows.
//
// Scope:
//   - Shared request-planning helpers only.
//
// Usage:
//   - Used by `cmd/acpctl` and `internal/onboard`.
//
// Invariants/Assumptions:
//   - Empty durations fall back to the canonical `30d` default.
//   - Rate-limit fields are only set when positive.
package keygen

import (
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
)

type GenerateRequestConfig struct {
	Alias    string
	Budget   float64
	RPM      int
	TPM      int
	Parallel int
	Duration string
	Role     string
	Models   []string
}

type GenerateRequestPlan struct {
	Request gateway.GenerateKeyRequest
	Role    string
	Models  []string
}

func PlanGenerateRequest(cfg GenerateRequestConfig) (GenerateRequestPlan, error) {
	if err := ValidateAlias(cfg.Alias); err != nil {
		return GenerateRequestPlan{}, err
	}

	role, err := ResolveRole(cfg.Role)
	if err != nil {
		return GenerateRequestPlan{}, err
	}
	if err := ValidateRole(role); err != nil {
		return GenerateRequestPlan{}, err
	}

	duration := strings.TrimSpace(cfg.Duration)
	if duration == "" {
		duration = DefaultConfig().Duration
	}

	models, err := GetModelsForRole(role)
	if err != nil {
		return GenerateRequestPlan{}, err
	}
	if len(cfg.Models) > 0 {
		models = append([]string(nil), cfg.Models...)
	}
	request := gateway.GenerateKeyRequest{
		KeyAlias:       cfg.Alias,
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
		Request: request,
		Role:    role,
		Models:  append([]string(nil), models...),
	}, nil
}

func (plan GenerateRequestPlan) WithAlias(alias string) GenerateRequestPlan {
	cloned := plan
	cloned.Request.KeyAlias = alias
	return cloned
}
