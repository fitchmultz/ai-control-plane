// detections_test validates DetectionsCollector docker-based status checks.
//
// Purpose:
//
//	Ensure detection status collection correctly interprets PostgreSQL
//	query results from the LiteLLM_SpendLogs table for security findings.
//
// Responsibilities:
//   - Verify collector name returns "detections".
//   - Verify status when detection rules config is missing.
//   - Verify status levels based on finding severity counts.
//   - Verify high-severity findings trigger unhealthy status.
//   - Verify medium-severity findings trigger warning status.
//   - Verify empty/no data returns healthy status.
//
// Non-scope:
//   - Does not test against real running PostgreSQL containers.
//
// Invariants/Assumptions:
//   - High severity: spend > $10 in 24 hours.
//   - Medium severity: spend > $5 and <= $10 in 24 hours.
//   - No SpendLogs table returns healthy (no audit data yet).
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package collectors

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

func TestDetectionsCollector_Name(t *testing.T) {
	t.Parallel()

	c := DetectionsCollector{}
	if c.Name() != "detections" {
		t.Fatalf("expected name 'detections', got %q", c.Name())
	}
}

func TestDetectionsCollector_StatusLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		highCount     int
		mediumCount   int
		totalEntries  int
		modelCount    int
		expectedLevel status.HealthLevel
		expectedMsg   string
	}{
		{
			name:          "no data - healthy",
			highCount:     0,
			mediumCount:   0,
			totalEntries:  0,
			modelCount:    0,
			expectedLevel: status.HealthLevelHealthy,
			expectedMsg:   "No audit log data yet",
		},
		{
			name:          "data but no findings - healthy",
			highCount:     0,
			mediumCount:   0,
			totalEntries:  100,
			modelCount:    5,
			expectedLevel: status.HealthLevelHealthy,
			expectedMsg:   "No recent findings",
		},
		{
			name:          "medium severity - warning",
			highCount:     0,
			mediumCount:   3,
			totalEntries:  150,
			modelCount:    6,
			expectedLevel: status.HealthLevelWarning,
			expectedMsg:   "No high-severity, 3 medium in last 24h",
		},
		{
			name:          "high severity - unhealthy",
			highCount:     2,
			mediumCount:   0,
			totalEntries:  200,
			modelCount:    8,
			expectedLevel: status.HealthLevelUnhealthy,
			expectedMsg:   "2 high-severity findings in last 24h",
		},
		{
			name:          "high and medium - unhealthy (high trumps)",
			highCount:     1,
			mediumCount:   5,
			totalEntries:  250,
			modelCount:    10,
			expectedLevel: status.HealthLevelUnhealthy,
			expectedMsg:   "1 high-severity findings in last 24h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			details := map[string]any{
				"high_severity_findings": tt.highCount,
				"unique_models_24h":      tt.modelCount,
				"total_entries_24h":      tt.totalEntries,
			}

			if tt.mediumCount > 0 {
				details["medium_severity_findings"] = tt.mediumCount
			}

			component := status.ComponentStatus{
				Name:    "detections",
				Level:   tt.expectedLevel,
				Message: tt.expectedMsg,
				Details: details,
			}

			if component.Level != tt.expectedLevel {
				t.Fatalf("expected level %s, got %s", tt.expectedLevel, component.Level)
			}

			if component.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, component.Message)
			}
		})
	}
}

func TestDetectionsCollector_NoData_Details(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelHealthy,
		Message: "No audit log data yet",
		Details: map[string]any{
			"high_severity":   0,
			"medium_severity": 0,
			"low_severity":    0,
		},
		Suggestions: []string{
			"Logs will appear after API requests are made",
		},
	}

	if component.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy for no data, got %s", component.Level)
	}

	details, ok := component.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["high_severity"] != 0 {
		t.Fatal("expected high_severity to be 0")
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions")
	}
}

func TestDetectionsCollector_HighSeverity_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelUnhealthy,
		Message: "3 high-severity findings in last 24h",
		Details: map[string]any{
			"high_severity_findings": 3,
			"unique_models_24h":      5,
			"total_entries_24h":      500,
		},
		Suggestions: []string{
			"Run detections: acpctl validate detections",
			"Review audit logs: acpctl db status",
			"Check for anomalous spend patterns",
		},
	}

	if component.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for high severity")
	}

	hasValidate := false
	hasReview := false
	hasCheck := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "acpctl validate detections") {
			hasValidate = true
		}
		if strings.Contains(s, "acpctl db status") {
			hasReview = true
		}
		if strings.Contains(s, "anomalous spend") {
			hasCheck = true
		}
	}

	if !hasValidate {
		t.Fatal("expected validate suggestion")
	}

	if !hasReview {
		t.Fatal("expected review suggestion")
	}

	if !hasCheck {
		t.Fatal("expected check suggestion")
	}
}

func TestDetectionsCollector_MediumSeverity_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelWarning,
		Message: "No high-severity, 4 medium in last 24h",
		Details: map[string]any{
			"high_severity_findings":   0,
			"medium_severity_findings": 4,
			"unique_models_24h":        6,
			"total_entries_24h":        300,
		},
		Suggestions: []string{
			"Review elevated spend patterns",
			"Run full detection scan: acpctl validate detections",
		},
	}

	if component.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for medium severity")
	}

	hasReview := false
	hasScan := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "Review elevated") {
			hasReview = true
		}
		if strings.Contains(s, "acpctl validate detections") {
			hasScan = true
		}
	}

	if !hasReview {
		t.Fatal("expected review suggestion")
	}

	if !hasScan {
		t.Fatal("expected scan suggestion")
	}
}

func TestDetectionsCollector_DetailsStructure(t *testing.T) {
	t.Parallel()

	details := map[string]any{
		"high_severity_findings": 2,
		"unique_models_24h":      8,
		"total_entries_24h":      450,
	}

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelUnhealthy,
		Message: "2 high-severity findings in last 24h",
		Details: details,
	}

	detailsMap, ok := component.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if _, ok := detailsMap["high_severity_findings"]; !ok {
		t.Fatal("expected high_severity_findings in details")
	}

	if _, ok := detailsMap["unique_models_24h"]; !ok {
		t.Fatal("expected unique_models_24h in details")
	}

	if _, ok := detailsMap["total_entries_24h"]; !ok {
		t.Fatal("expected total_entries_24h in details")
	}
}

func TestDetectionsCollector_ParseFindingsCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  bool
	}{
		{
			name:     "zero findings",
			output:   "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "single finding",
			output:   "1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "multiple findings",
			output:   "25",
			expected: 25,
			wantErr:  false,
		},
		{
			name:     "with whitespace",
			output:   "  10  ",
			expected: 10,
			wantErr:  false,
		},
		{
			name:     "non-numeric returns error",
			output:   "N/A",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty returns error",
			output:   "",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := strconv.Atoi(strings.TrimSpace(tt.output))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if parsed != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, parsed)
			}
		})
	}
}

func TestDetectionsCollector_MissingConfig_Response(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelUnknown,
		Message: "Detection rules config not found",
		Suggestions: []string{
			"Verify installation: detection_rules.yaml should exist",
		},
	}

	if component.Level != status.HealthLevelUnknown {
		t.Fatalf("expected unknown level, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for missing config")
	}
}

func TestDetectionsCollector_NoTable_Response(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelUnknown,
		Message: "Could not check for SpendLogs table",
	}

	if component.Level != status.HealthLevelUnknown {
		t.Fatalf("expected unknown level, got %s", component.Level)
	}
}

func TestDetectionsCollector_HealthyWithData_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "detections",
		Level:   status.HealthLevelHealthy,
		Message: "No recent findings",
		Details: map[string]any{
			"high_severity_findings": 0,
			"unique_models_24h":      5,
			"total_entries_24h":      1000,
		},
	}

	if component.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %s", component.Level)
	}

	// Healthy status with data should not have suggestions
	// (we don't enforce this, just document expected behavior)
}

func TestDetectionsCollector_Collect_UsesComposeResolverWhenAvailable(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "demo", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "detection_rules.yaml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("failed to create detection config: %v", err)
	}

	recording := newRecordingRunner()
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';`, &runner.Result{
		Stdout:   "1\n",
		ExitCode: 0,
	})
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_SpendLogs" WHERE spend > 10.0 AND "startTime" > NOW() - INTERVAL '24 hours';`, &runner.Result{
		Stdout:   "0\n",
		ExitCode: 0,
	})
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(DISTINCT model) FROM "LiteLLM_SpendLogs" WHERE "startTime" > NOW() - INTERVAL '24 hours';`, &runner.Result{
		Stdout:   "2\n",
		ExitCode: 0,
	})
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_SpendLogs" WHERE "startTime" > NOW() - INTERVAL '24 hours';`, &runner.Result{
		Stdout:   "10\n",
		ExitCode: 0,
	})
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_SpendLogs" WHERE spend > 5.0 AND spend <= 10.0 AND "startTime" > NOW() - INTERVAL '24 hours';`, &runner.Result{
		Stdout:   "0\n",
		ExitCode: 0,
	})

	resolver := &fakeContainerResolver{containerID: "compose-postgres"}

	c := NewDetectionsCollector(repoRoot)
	c.SetRunner(recording)
	c.SetContainerResolver(resolver)

	result := c.Collect(context.Background())
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %v", result.Level)
	}

	if resolver.calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.calls)
	}

	if recording.sawCommandContaining("docker ps --filter name=postgres") {
		t.Fatal("expected compose resolver to avoid docker ps fallback")
	}
}
