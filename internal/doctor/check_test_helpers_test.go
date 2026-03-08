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
	"net"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func reserveLocalPort(t *testing.T) (int, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve local port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return port, func() {
		_ = listener.Close()
	}
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
