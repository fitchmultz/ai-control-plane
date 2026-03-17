// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Adapt budget runtime findings into operator-facing doctor results.
//
// Responsibilities:
//   - Reuse the shared budget component status collected by internal/status.
//   - Preserve consistent doctor severity mapping for budget findings.
//
// Scope:
//   - Budget finding diagnostics only.
//
// Usage:
//   - Used through DefaultChecks and focused unit tests.
//
// Invariants/Assumptions:
//   - Doctor must not duplicate budget interpretation logic.
package doctor

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type budgetFindingsCheck struct{ noFixCheck }

func (c budgetFindingsCheck) ID() string { return "budget_findings" }

func (c budgetFindingsCheck) Run(_ context.Context, opts Options) CheckResult {
	return runtimeComponentCheck(opts, c.ID(), "Budget Findings", "budget", "Budget", func(component status.ComponentStatus) CheckResult {
		return componentCheckResult(c.ID(), "Budget Findings", component, severityForLevel(component.Level))
	})
}
