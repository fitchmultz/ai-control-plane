// package_test.go validates doctor report aggregation and exit-code behavior.
//
// Purpose:
//
//	Ensure doctor report orchestration combines check results, fix flows, and
//	exit-code precedence consistently for operator-facing health commands.
//
// Responsibilities:
//   - Verify overall health aggregation across check combinations.
//   - Verify skipped checks and fixable checks are handled correctly.
//   - Verify report exit-code precedence and serialization helpers.
//
// Scope:
//   - Covers package-level doctor orchestration, not individual check internals.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Mock checks return deterministic results for a given test case.
//   - Report aggregation is pure for the provided inputs.
package doctor

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// mockCheck is a test helper for creating mock checks.
type mockCheck struct {
	id     string
	result CheckResult
}

func (m mockCheck) ID() string {
	return m.id
}

func (m mockCheck) Run(ctx context.Context, opts Options) CheckResult {
	return m.result
}

func (m mockCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}

func TestRun_AggregatesOverall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		checks   []Check
		expected status.HealthLevel
	}{
		{
			name: "all healthy returns healthy",
			checks: []Check{
				mockCheck{"check1", CheckResult{Level: status.HealthLevelHealthy}},
				mockCheck{"check2", CheckResult{Level: status.HealthLevelHealthy}},
			},
			expected: status.HealthLevelHealthy,
		},
		{
			name: "one warning returns warning",
			checks: []Check{
				mockCheck{"check1", CheckResult{Level: status.HealthLevelHealthy}},
				mockCheck{"check2", CheckResult{Level: status.HealthLevelWarning}},
			},
			expected: status.HealthLevelWarning,
		},
		{
			name: "one unhealthy returns unhealthy",
			checks: []Check{
				mockCheck{"check1", CheckResult{Level: status.HealthLevelHealthy}},
				mockCheck{"check2", CheckResult{Level: status.HealthLevelUnhealthy}},
			},
			expected: status.HealthLevelUnhealthy,
		},
		{
			name: "unhealthy overrides warning",
			checks: []Check{
				mockCheck{"check1", CheckResult{Level: status.HealthLevelWarning}},
				mockCheck{"check2", CheckResult{Level: status.HealthLevelUnhealthy}},
			},
			expected: status.HealthLevelUnhealthy,
		},
		{
			name:     "empty checks returns healthy",
			checks:   []Check{},
			expected: status.HealthLevelHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			opts := Options{}

			report := Run(ctx, tt.checks, opts)

			if report.Overall != tt.expected {
				t.Errorf("expected overall %v, got %v", tt.expected, report.Overall)
			}
		})
	}
}

func TestRun_SkipsChecks(t *testing.T) {
	t.Parallel()

	check1 := mockCheck{"check1", CheckResult{ID: "check1", Level: status.HealthLevelHealthy}}
	check2 := mockCheck{"check2", CheckResult{ID: "check2", Level: status.HealthLevelHealthy}}

	ctx := context.Background()
	opts := Options{
		SkipChecks: map[string]struct{}{
			"check1": {},
		},
	}

	report := Run(ctx, []Check{check1, check2}, opts)

	if len(report.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(report.Results))
	}

	if report.Results[0].ID != "check2" {
		t.Errorf("expected check2, got %s", report.Results[0].ID)
	}
}

func TestRun_AppliesFixes(t *testing.T) {
	t.Parallel()

	// Create a check that can be fixed
	check := &fixableCheck{
		id:         "fixable",
		fixed:      false,
		initialLvl: status.HealthLevelUnhealthy,
		fixedLvl:   status.HealthLevelHealthy,
	}

	ctx := context.Background()
	opts := Options{Fix: true}

	report := Run(ctx, []Check{check}, opts)

	if !check.fixed {
		t.Error("expected fix to be applied")
	}

	if report.Results[0].Level != status.HealthLevelHealthy {
		t.Errorf("expected healthy after fix, got %v", report.Results[0].Level)
	}

	if !report.Results[0].FixApplied {
		t.Error("expected FixApplied to be true")
	}
}

// fixableCheck is a test helper that can be fixed.
type fixableCheck struct {
	id         string
	fixed      bool
	initialLvl status.HealthLevel
	fixedLvl   status.HealthLevel
}

func (f *fixableCheck) ID() string {
	return f.id
}

func (f *fixableCheck) Run(ctx context.Context, opts Options) CheckResult {
	if f.fixed {
		return CheckResult{ID: f.id, Level: f.fixedLvl}
	}
	return CheckResult{ID: f.id, Level: f.initialLvl}
}

func (f *fixableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	f.fixed = true
	return true, "fixed!", nil
}

func TestExitCodeForReport_Precedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		results      []CheckResult
		overall      status.HealthLevel
		expectedCode int
	}{
		{
			name:         "healthy returns 0",
			results:      []CheckResult{{Level: status.HealthLevelHealthy}},
			overall:      status.HealthLevelHealthy,
			expectedCode: 0,
		},
		{
			name:         "domain failure returns 1",
			results:      []CheckResult{{Level: status.HealthLevelUnhealthy, Severity: SeverityDomain}},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 1,
		},
		{
			name:         "prereq failure returns 2",
			results:      []CheckResult{{Level: status.HealthLevelUnhealthy, Severity: SeverityPrereq}},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 2,
		},
		{
			name:         "runtime failure returns 3",
			results:      []CheckResult{{Level: status.HealthLevelUnhealthy, Severity: SeverityRuntime}},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 3,
		},
		{
			name: "runtime precedes prereq",
			results: []CheckResult{
				{Level: status.HealthLevelUnhealthy, Severity: SeverityPrereq},
				{Level: status.HealthLevelUnhealthy, Severity: SeverityRuntime},
			},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 3,
		},
		{
			name: "runtime precedes domain",
			results: []CheckResult{
				{Level: status.HealthLevelUnhealthy, Severity: SeverityDomain},
				{Level: status.HealthLevelUnhealthy, Severity: SeverityRuntime},
			},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 3,
		},
		{
			name: "prereq precedes domain",
			results: []CheckResult{
				{Level: status.HealthLevelUnhealthy, Severity: SeverityDomain},
				{Level: status.HealthLevelUnhealthy, Severity: SeverityPrereq},
			},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 2,
		},
		{
			name:         "unhealthy overall with no severity returns 1",
			results:      []CheckResult{{Level: status.HealthLevelHealthy}},
			overall:      status.HealthLevelUnhealthy,
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := Report{
				Overall: tt.overall,
				Results: tt.results,
			}

			code := ExitCodeForReport(report)

			if code != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, code)
			}
		})
	}
}

func TestReport_WriteJSON(t *testing.T) {
	t.Parallel()

	report := Report{
		Overall:   status.HealthLevelHealthy,
		Timestamp: "2026-01-01T00:00:00Z",
		Duration:  "100ms",
		Results: []CheckResult{
			{ID: "test", Name: "Test", Level: status.HealthLevelHealthy, Message: "OK"},
		},
	}

	// Just verify it doesn't panic and produces valid JSON
	var buf strings.Builder
	err := report.WriteJSON(&buf)
	if err != nil {
		t.Errorf("WriteJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"overall": "healthy"`) {
		t.Error("JSON output missing expected fields")
	}
}

func TestDefaultChecks(t *testing.T) {
	t.Parallel()

	checks := DefaultChecks()

	expectedIDs := []string{
		"docker_available",
		"ports_free",
		"env_vars_set",
		"gateway_healthy",
		"db_connectable",
		"config_valid",
		"credentials_valid",
		"budget_findings",
		"detections_findings",
	}

	if len(checks) != len(expectedIDs) {
		t.Errorf("expected %d checks, got %d", len(expectedIDs), len(checks))
	}

	for i, id := range expectedIDs {
		if checks[i].ID() != id {
			t.Errorf("expected check %d to be %s, got %s", i, id, checks[i].ID())
		}
	}
}
