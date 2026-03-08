// checker_test.go - Tests for the health compatibility wrapper.
//
// Purpose:
//
//	Verify the health package still maps canonical status levels into the
//	legacy compatibility result shape.
//
// Responsibilities:
//   - Assert status level conversion remains stable.
//
// Non-scope:
//   - Does not exercise live gateway or database probes.
//
// Invariants/Assumptions:
//   - Compatibility status conversion remains a pure mapping.
//
// Scope:
//   - Compatibility mapping tests only.
//
// Usage:
//   - Used through `go test` for health package coverage.
package health

import (
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestConvertLevel(t *testing.T) {
	tests := []struct {
		level status.HealthLevel
		want  Status
	}{
		{status.HealthLevelHealthy, StatusHealthy},
		{status.HealthLevelWarning, StatusWarning},
		{status.HealthLevelUnhealthy, StatusUnhealthy},
		{status.HealthLevelUnknown, StatusUnknown},
	}

	for _, tt := range tests {
		if got := convertLevel(tt.level); got != tt.want {
			t.Fatalf("convertLevel(%q) = %v, want %v", tt.level, got, tt.want)
		}
	}
}
