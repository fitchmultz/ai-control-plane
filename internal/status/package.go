// Package status provides aggregated system health status collection.
//
// Purpose:
//
//	Collect and aggregate health status from multiple system domains
//	(gateway, database, keys, budgets, detections) into a unified report.
//
// Responsibilities:
//   - Define Collector interface for domain-specific status collection
//   - Aggregate results from all collectors into StatusReport
//   - Support JSON and human-readable output formats
//   - Provide color-coded status indicators (green/yellow/red)
//
// Non-scope:
//   - Does not execute remediation actions
//   - Does not modify system state
//
// Invariants/Assumptions:
//   - All collectors are read-only operations
//   - Collector failures do not prevent other collectors from running
//   - Overall status is HEALTHY only if all required collectors report healthy
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

// HealthLevel represents the health status of a component.
type HealthLevel string

const (
	HealthLevelHealthy   HealthLevel = "healthy"   // Green - all good
	HealthLevelWarning   HealthLevel = "warning"   // Yellow - attention needed
	HealthLevelUnhealthy HealthLevel = "unhealthy" // Red - action required
	HealthLevelUnknown   HealthLevel = "unknown"   // Grey - unable to determine
)

// ComponentStatus represents the status of a single component.
type ComponentStatus struct {
	Name        string      `json:"name"`
	Level       HealthLevel `json:"level"`
	Message     string      `json:"message"`
	Details     any         `json:"details,omitempty"`
	Suggestions []string    `json:"suggestions,omitempty"`
}

// StatusReport is the aggregated status from all collectors.
type StatusReport struct {
	Overall    HealthLevel                `json:"overall"`
	Components map[string]ComponentStatus `json:"components"`
	Timestamp  string                     `json:"timestamp"`
	Duration   string                     `json:"duration"`
}

// Collector gathers status for a specific domain.
type Collector interface {
	// Name returns the collector's domain name (e.g., "gateway", "database").
	Name() string

	// Collect gathers status information from the domain.
	// Returns a ComponentStatus; errors should be captured in the status level/message.
	Collect(ctx context.Context) ComponentStatus
}

// Options configures status collection behavior.
type Options struct {
	RepoRoot    string
	GatewayHost string
	LITELLMPort string
	Wide        bool // Include extended details
}

// CollectAll runs all collectors and returns an aggregated report.
func CollectAll(ctx context.Context, collectors []Collector, opts Options) StatusReport {
	start := time.Now()

	// Use a wait group to run collectors concurrently
	var wg sync.WaitGroup
	results := make(map[string]ComponentStatus)
	var mu sync.Mutex

	for _, collector := range collectors {
		wg.Add(1)
		go func(c Collector) {
			defer wg.Done()
			status := c.Collect(ctx)
			mu.Lock()
			results[c.Name()] = status
			mu.Unlock()
		}(collector)
	}

	wg.Wait()
	duration := time.Since(start)

	// Determine overall status
	overall := HealthLevelHealthy
	for _, status := range results {
		switch status.Level {
		case HealthLevelUnhealthy:
			overall = HealthLevelUnhealthy
		case HealthLevelWarning:
			if overall == HealthLevelHealthy {
				overall = HealthLevelWarning
			}
		}
	}

	return StatusReport{
		Overall:    overall,
		Components: results,
		Timestamp:  start.UTC().Format(time.RFC3339),
		Duration:   duration.Round(time.Millisecond).String(),
	}
}

// WriteHuman formats the report for terminal output.
func (r StatusReport) WriteHuman(w io.Writer, wide bool) error {
	colors := terminal.NewColors()
	sf := terminal.NewStatusFormatter()

	// Helper to format status using symbols
	formatStatus := func(level HealthLevel) string {
		switch level {
		case HealthLevelHealthy:
			return sf.OK()
		case HealthLevelWarning:
			return sf.Warn()
		case HealthLevelUnhealthy:
			return sf.Fail()
		default:
			return "[UNK]"
		}
	}

	// Print header
	fmt.Fprintln(w, colors.Bold+"=== AI Control Plane Status ==="+colors.Reset)
	fmt.Fprintln(w, "")

	// Define component order
	order := []string{"gateway", "database", "keys", "budget", "detections"}

	// Print each component
	for _, name := range order {
		status, ok := r.Components[name]
		if !ok {
			continue
		}

		// Pad name to align columns
		paddedName := fmt.Sprintf("%-11s", strings.ToUpper(name[:1])+name[1:])
		fmt.Fprintf(w, "%s %s %s\n", paddedName, formatStatus(status.Level), status.Message)

		// Show suggestions if unhealthy or warning
		if len(status.Suggestions) > 0 && (status.Level == HealthLevelUnhealthy || status.Level == HealthLevelWarning) {
			for _, suggestion := range status.Suggestions {
				fmt.Fprintf(w, "             %s\n", suggestion)
			}
		}

		// Show details in wide mode
		if wide && status.Details != nil {
			switch details := status.Details.(type) {
			case map[string]any:
				for k, v := range details {
					fmt.Fprintf(w, "             %s: %v\n", k, v)
				}
			}
		}
	}

	// Print overall status
	fmt.Fprintln(w, "")
	var overallStr string
	switch r.Overall {
	case HealthLevelHealthy:
		overallStr = sf.Healthy()
	case HealthLevelWarning:
		overallStr = sf.Warning()
	case HealthLevelUnhealthy:
		overallStr = sf.Unhealthy()
	default:
		overallStr = "UNKNOWN"
	}
	fmt.Fprintf(w, "Overall: %s (%s)\n", overallStr, r.Duration)

	return nil
}

// WriteJSON writes the report as JSON.
func (r StatusReport) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}
