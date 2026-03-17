// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Adapt detection runtime findings into operator-facing doctor results.
//
// Responsibilities:
//   - Reuse the shared detections component status collected by internal/status.
//   - Preserve consistent doctor severity mapping for detection findings.
//
// Scope:
//   - Detection finding diagnostics only.
//
// Usage:
//   - Used through DefaultChecks and focused unit tests.
//
// Invariants/Assumptions:
//   - Doctor must not duplicate detection interpretation logic.
package doctor

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type detectionsFindingsCheck struct{ noFixCheck }

func (c detectionsFindingsCheck) ID() string { return "detections_findings" }

func (c detectionsFindingsCheck) Run(_ context.Context, opts Options) CheckResult {
	return runtimeComponentCheck(opts, c.ID(), "Security Findings", "detections", "Detections", func(component status.ComponentStatus) CheckResult {
		return componentCheckResult(c.ID(), "Security Findings", component, severityForLevel(component.Level))
	})
}
