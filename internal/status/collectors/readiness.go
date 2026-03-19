// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose the latest readiness-evidence result for operator status and
//	dashboard views.
//
// Responsibilities:
//   - Resolve the latest generated readiness run.
//   - Surface run freshness, pass/fail state, and gate counts as one typed
//     status result.
//   - Preserve operator guidance when no readiness evidence exists yet.
//
// Scope:
//   - Readiness-evidence status collection only.
//
// Usage:
//   - Construct with NewReadinessCollector(repoRoot) and call Collect(ctx).
//
// Invariants/Assumptions:
//   - Latest readiness pointers are owned by internal/readiness.
//   - PASS runs older than one week are treated as stale warnings.
package collectors

import (
	"context"
	"fmt"
	"time"

	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const readinessStaleAge = 7 * 24 * time.Hour

// ReadinessCollector summarizes the latest readiness-evidence run.
type ReadinessCollector struct {
	repoRoot string
}

// NewReadinessCollector creates a readiness collector for the repository runtime.
func NewReadinessCollector(repoRoot string) ReadinessCollector {
	return ReadinessCollector{repoRoot: repoRoot}
}

// Name returns the collector's domain name.
func (c ReadinessCollector) Name() string {
	return "readiness"
}

// Collect gathers readiness-evidence status information.
func (c ReadinessCollector) Collect(ctx context.Context) status.ComponentStatus {
	outputRoot := repopath.DemoLogsPath(c.repoRoot, "evidence")
	runDir, err := readiness.ResolveLatestRun(outputRoot)
	if err != nil {
		return componentStatus(c.Name(), status.HealthLevelWarning, "No readiness evidence run found", status.ComponentDetails{},
			"Run make readiness-evidence to generate a fresh proof pack",
		)
	}
	verifier := readiness.NewVerifier()
	summary, err := verifier.VerifyRun(ctx, runDir)
	if err != nil {
		return componentStatus(c.Name(), status.HealthLevelWarning,
			fmt.Sprintf("Latest readiness run could not be verified: %v", err),
			status.ComponentDetails{Error: err.Error()},
			"Run make readiness-evidence-verify to inspect the current run",
		)
	}

	generatedAt, parseErr := time.Parse(time.RFC3339, summary.GeneratedAtUTC)
	if parseErr != nil {
		return componentStatus(c.Name(), status.HealthLevelWarning,
			fmt.Sprintf("Latest readiness run has an invalid timestamp: %v", parseErr),
			status.ComponentDetails{
				ReadinessRunID:         summary.RunID,
				ReadinessGeneratedAt:   summary.GeneratedAtUTC,
				ReadinessOverallStatus: summary.OverallStatus,
				FailingGateCount:       summary.FailingGateCount,
				SkippedGateCount:       summary.SkippedGateCount,
				Error:                  parseErr.Error(),
			},
		)
	}

	age := time.Since(generatedAt).Round(time.Minute)
	details := status.ComponentDetails{
		ReadinessRunID:         summary.RunID,
		ReadinessGeneratedAt:   summary.GeneratedAtUTC,
		ReadinessOverallStatus: summary.OverallStatus,
		ReadinessAge:           age.String(),
		FailingGateCount:       summary.FailingGateCount,
		SkippedGateCount:       summary.SkippedGateCount,
	}
	switch {
	case summary.OverallStatus != "PASS":
		return componentStatus(c.Name(), status.HealthLevelUnhealthy,
			fmt.Sprintf("Latest readiness run failed %d required gates", summary.FailingGateCount),
			details,
			"Review the latest readiness run before external reuse",
		)
	case age > readinessStaleAge:
		return componentStatus(c.Name(), status.HealthLevelWarning,
			fmt.Sprintf("Latest readiness run passed but is stale (%s old)", age),
			details,
			"Regenerate current proof with make readiness-evidence",
		)
	default:
		return componentStatus(c.Name(), status.HealthLevelHealthy,
			fmt.Sprintf("Latest readiness run passed %s ago", age),
			details,
		)
	}
}
