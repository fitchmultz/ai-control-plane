// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Adapt the canonical gateway runtime inspection result into doctor output.
//
// Responsibilities:
//   - Reuse the shared gateway component status collected by internal/status.
//   - Escalate missing master-key state into a doctor prerequisite failure.
//
// Scope:
//   - LiteLLM gateway diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Doctor must not duplicate gateway probe interpretation logic.
package doctor

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type gatewayHealthyCheck struct{}

func (c gatewayHealthyCheck) ID() string { return "gateway_healthy" }

func (c gatewayHealthyCheck) Run(_ context.Context, opts Options) CheckResult {
	return runtimeComponentCheck(opts, c.ID(), "Gateway Healthy", "gateway", "Gateway", func(component status.ComponentStatus) CheckResult {
		if !component.Details.MasterKeyConfigured {
			return withCheckDetails(
				newCheckResult(c.ID(), "Gateway Healthy", status.HealthLevelUnhealthy, SeverityPrereq, "LITELLM_MASTER_KEY not set; cannot run authorized gateway check"),
				component.Details,
				"Set LITELLM_MASTER_KEY in demo/.env",
				"Or export it in your shell environment",
			)
		}

		return componentCheckResult(c.ID(), "Gateway Healthy", component, severityForLevel(component.Level))
	})
}

func (c gatewayHealthyCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return recoverGatewayRuntime(ctx, opts)
}

func severityForLevel(level status.HealthLevel) Severity {
	if level == status.HealthLevelUnknown {
		return SeverityRuntime
	}
	return SeverityDomain
}
