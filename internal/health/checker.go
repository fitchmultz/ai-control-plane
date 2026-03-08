// Package health provides health check functionality for AI Control Plane services.
//
// Purpose:
//
//	Provide a compatibility wrapper that runs the canonical typed runtime
//	health collectors instead of maintaining a separate health model.
//
// Responsibilities:
//   - Collect status via shared gateway and database services.
//   - Reuse the status package's report model for output and exit handling.
//
// Non-scope:
//   - Does not maintain a separate health probing implementation.
//
// Invariants/Assumptions:
//   - Compatibility output is derived from the canonical status report.
//
// Scope:
//   - Health command compatibility only.
//
// Usage:
//   - Used through its package exports and compatibility tests as applicable.
package health

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// Status represents compatibility health states.
type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusUnhealthy
	StatusWarning
)

// Component represents a rendered compatibility health component.
type Component struct {
	Name       string
	Status     Status
	Message    string
	Details    string
	IsOptional bool
}

// Result contains the overall health check result.
type Result struct {
	Components []Component
	Overall    Status
	Report     status.StatusReport
}

// Checker performs health checks.
type Checker struct {
	repoRoot string
	verbose  bool
}

// NewChecker creates a new health checker.
func NewChecker(compose *docker.Compose, verbose bool) *Checker {
	return &Checker{verbose: verbose}
}

// Run performs all health checks and returns the result.
func (c *Checker) Run(ctx context.Context) *Result {
	inspector := runtimeinspect.NewInspector(detectRepoRoot(ctx))
	defer inspector.Close()
	report := inspector.Collect(ctx, status.Options{RepoRoot: detectRepoRoot(ctx), Wide: c.verbose})
	return &Result{
		Components: convertComponents(report),
		Overall:    convertLevel(report.Overall),
		Report:     report,
	}
}

// PrintSummary writes the shared status report to stdout.
func (c *Checker) PrintSummary(result *Result) {
	if result == nil {
		return
	}
	_ = result.Report.WriteHuman(stdoutWriter{}, c.verbose)
}

func convertLevel(level status.HealthLevel) Status {
	switch level {
	case status.HealthLevelHealthy:
		return StatusHealthy
	case status.HealthLevelWarning:
		return StatusWarning
	case status.HealthLevelUnhealthy:
		return StatusUnhealthy
	default:
		return StatusUnknown
	}
}

func convertComponents(report status.StatusReport) []Component {
	order := []string{"gateway", "database", "keys", "budget", "detections"}
	components := make([]Component, 0, len(order))
	for _, name := range order {
		component, ok := report.Components[name]
		if !ok {
			continue
		}
		components = append(components, Component{
			Name:    component.Name,
			Status:  convertLevel(component.Level),
			Message: component.Message,
			Details: fmt.Sprint(component.Details.Lines()),
		})
	}
	return components
}

type stdoutWriter struct{}

func (stdoutWriter) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

func detectRepoRoot(ctx context.Context) string {
	root, _ := config.NewLoader().RepoRoot(ctx)
	return root
}
