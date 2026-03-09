// check_test_helpers_test.go - Shared helpers for focused doctor check suites.
//
// Purpose:
//   - Centralize common port and runtime-report setup for doctor unit tests.
//
// Responsibilities:
//   - Reserve ephemeral ports deterministically.
//   - Build compact runtime reports for check-specific suites.
//
// Scope:
//   - Test-only helpers for internal/doctor.
//
// Usage:
//   - Imported implicitly by doctor `_test.go` files.
//
// Invariants/Assumptions:
//   - Helpers avoid fixed ports and external runtime dependencies.
package doctor

import (
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func reserveLocalPort(t *testing.T) (int, func()) {
	t.Helper()
	return testutil.ReserveLocalPort(t)
}

func runtimeReportFor(component string, details status.ComponentDetails, level status.HealthLevel, message string) *status.StatusReport {
	return &status.StatusReport{
		Components: map[string]status.ComponentStatus{
			component: {
				Name:        component,
				Level:       level,
				Message:     message,
				Details:     details,
				Suggestions: []string{"stub-suggestion"},
			},
		},
	}
}
