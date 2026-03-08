// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose a detections collector that combines local rule presence with the
//	shared typed database service for recent finding summaries.
//
// Responsibilities:
//   - Verify detection rules configuration exists locally.
//   - Convert typed spend-log counts into status.ComponentStatus.
//   - Preserve operator guidance for recent findings and empty audit history.
//
// Non-scope:
//   - Does not execute collector-local SQL command helpers.
//
// Invariants/Assumptions:
//   - Detection counts come from the shared typed database service.
//
// Scope:
//   - Detection status collection only.
//
// Usage:
//   - Construct with NewDetectionsCollector(repoRoot, client) and call Collect(ctx).
package collectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// DetectionsCollector summarizes recent detection findings.
type DetectionsCollector struct {
	repoRoot string
	reader   db.ReadonlyServiceReader
}

// NewDetectionsCollector creates a new detections collector.
func NewDetectionsCollector(repoRoot string, reader db.ReadonlyServiceReader) DetectionsCollector {
	return DetectionsCollector{repoRoot: repoRoot, reader: reader}
}

// Name returns the collector's domain name.
func (c DetectionsCollector) Name() string {
	return "detections"
}

// Collect gathers detection status information.
func (c DetectionsCollector) Collect(ctx context.Context) status.ComponentStatus {
	configPath := filepath.Join(c.repoRoot, "demo", "config", "detection_rules.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Detection rules config not found",
			Suggestions: []string{
				"Verify installation: detection_rules.yaml should exist",
			},
		}
	}

	summary, err := c.reader.DetectionSummary(ctx)
	details := status.ComponentDetails{
		SpendLogsTableExists:   summary.SpendLogsTableExists,
		HighSeverityFindings:   summary.HighSeverity,
		MediumSeverityFindings: summary.MediumSeverity,
		UniqueModels24h:        summary.UniqueModels24h,
		TotalEntries24h:        summary.TotalEntries24h,
	}
	if err != nil {
		details.Error = err.Error()
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Could not query detection data",
			Details: details,
		}
	}

	if !summary.SpendLogsTableExists {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelHealthy,
			Message: "No audit log data yet",
			Details: details,
			Suggestions: []string{
				"Logs will appear after API requests are made",
			},
		}
	}

	if summary.HighSeverity > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("%d high-severity findings in last 24h", summary.HighSeverity),
			Details: details,
			Suggestions: []string{
				"Run detections: acpctl validate detections",
				"Review audit logs: acpctl db status",
				"Check for anomalous spend patterns",
			},
		}
	}

	if summary.MediumSeverity > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("No high-severity, %d medium in last 24h", summary.MediumSeverity),
			Details: details,
			Suggestions: []string{
				"Review elevated spend patterns",
				"Run full detection scan: acpctl validate detections",
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: "No recent findings",
		Details: details,
	}
}
