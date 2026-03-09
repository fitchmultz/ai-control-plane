// Package doctor provides environment preflight diagnostics for the AI Control Plane.
//
// Purpose:
//
//	Validate runtime environment before operations to catch common
//	failures early and provide actionable remediation steps.
//
// Responsibilities:
//   - Define check interface and result types
//   - Implement concrete checks for Docker, ports, env vars, DB, gateway, config
//   - Support --json output for CI integration
//   - Support --fix flag for safe auto-remediation
//   - Return deterministic exit codes aligned with exitcodes package
//
// Non-scope:
//   - Does not perform actual deployments or system modifications (unless --fix)
//   - Does not replace host-preflight script; complements it with typed checks
//
// Invariants/Assumptions:
//   - Checks are read-only unless Fix() is explicitly called
//   - Secrets are never logged in Details or Message
//   - Exit codes follow precedence: runtime > prereq > domain > success
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package doctor

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// Severity represents the severity level of a check failure.
type Severity string

const (
	// SeverityDomain indicates a domain-level failure (e.g., service unhealthy).
	SeverityDomain Severity = "domain"
	// SeverityPrereq indicates a prerequisite not ready (e.g., Docker not installed).
	SeverityPrereq Severity = "prereq"
	// SeverityRuntime indicates a runtime/internal error.
	SeverityRuntime Severity = "runtime"
)

// CheckResult represents the outcome of a single diagnostic check.
type CheckResult struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Level       status.HealthLevel      `json:"level"`
	Severity    Severity                `json:"severity,omitempty"`
	Message     string                  `json:"message"`
	Details     status.ComponentDetails `json:"details,omitempty"`
	Suggestions []string                `json:"suggestions,omitempty"`
	FixApplied  bool                    `json:"fix_applied,omitempty"`
	FixMessage  string                  `json:"fix_message,omitempty"`
}

// Check defines the interface for all diagnostic checks.
type Check interface {
	// ID returns a unique identifier for this check.
	ID() string
	// Run executes the check and returns the result.
	Run(ctx context.Context, opts Options) CheckResult
	// Fix attempts to auto-remediate the issue. Returns (applied, message, error).
	Fix(ctx context.Context, opts Options) (bool, string, error)
}

// Options configures diagnostic check behavior.
type Options struct {
	RepoRoot      string
	Gateway       config.GatewaySettings
	RequiredPorts []int
	SkipChecks    map[string]struct{}
	Fix           bool
	Wide          bool
	RuntimeReport *status.StatusReport
}

// Report aggregates all check results.
type Report struct {
	Overall   status.HealthLevel `json:"overall"`
	Timestamp string             `json:"timestamp"`
	Duration  string             `json:"duration"`
	Results   []CheckResult      `json:"results"`
}

// WriteJSON writes the report as indented JSON.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ExitCodeForReport determines the exit code based on report results.
// Precedence: runtime > prereq > domain > success
func ExitCodeForReport(r Report) int {
	hasDomain := false
	hasPrereq := false
	hasRuntime := false

	for _, result := range r.Results {
		switch result.Severity {
		case SeverityRuntime:
			hasRuntime = true
		case SeverityPrereq:
			hasPrereq = true
		case SeverityDomain:
			if result.Level != status.HealthLevelHealthy {
				hasDomain = true
			}
		}
	}

	if hasRuntime {
		return exitcodes.ACPExitRuntime
	}
	if hasPrereq {
		return exitcodes.ACPExitPrereq
	}
	if hasDomain || r.Overall != status.HealthLevelHealthy {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

// Run executes all checks and returns an aggregated report.
func Run(ctx context.Context, checks []Check, opts Options) Report {
	start := time.Now()
	results := make([]CheckResult, 0, len(checks))
	overall := status.HealthLevelHealthy

	for _, check := range checks {
		if _, skipped := opts.SkipChecks[check.ID()]; skipped {
			continue
		}

		result := check.Run(ctx, opts)

		// Attempt auto-remediation if requested and check is not healthy
		if opts.Fix && result.Level != status.HealthLevelHealthy {
			if applied, msg, err := check.Fix(ctx, opts); err == nil && applied {
				result.FixApplied = true
				result.FixMessage = msg
				// Re-run check to verify fix
				result = check.Run(ctx, opts)
				result.FixApplied = true
				result.FixMessage = msg
			}
		}

		if result.Level == status.HealthLevelUnhealthy {
			overall = status.HealthLevelUnhealthy
		} else if result.Level == status.HealthLevelWarning && overall == status.HealthLevelHealthy {
			overall = status.HealthLevelWarning
		}

		results = append(results, result)
	}

	return Report{
		Overall:   overall,
		Timestamp: start.UTC().Format(time.RFC3339),
		Duration:  time.Since(start).Round(time.Millisecond).String(),
		Results:   results,
	}
}
