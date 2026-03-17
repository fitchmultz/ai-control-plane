// check_detections_test.go - Coverage for detections doctor check adaptation.
//
// Purpose:
//   - Verify detection findings map directly from runtime status into doctor output.
//
// Responsibilities:
//   - Preserve level, severity, and shared detail propagation.
//
// Scope:
//   - Detection finding doctor check behavior only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic runtime reports.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestDetectionsFindingsCheckRunPassthrough(t *testing.T) {
	result := (detectionsFindingsCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("detections", status.ComponentDetails{
			HighSeverityFindings: 2,
		}, status.HealthLevelUnhealthy, "2 high-severity findings in last 24h"),
	})

	if result.Level != status.HealthLevelUnhealthy || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}
