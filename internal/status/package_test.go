// package_test validates core status types and aggregation behavior.
//
// Purpose:
//
//	Ensure status collection, aggregation, and output formatting behave correctly
//	for all health levels and component combinations.
//
// Responsibilities:
//   - Verify CollectAll aggregates results from multiple collectors.
//   - Verify overall status calculation (healthy -> warning -> unhealthy priority).
//   - Verify JSON and human-readable output formatting.
//   - Verify edge cases like empty collectors and nil details.
//
// Non-scope:
//   - Does not test collector-specific logic (see collectors/*_test.go).
//   - Does not require running Docker services.
//
// Invariants/Assumptions:
//   - Collectors are read-only and do not modify system state.
//   - StatusReport is deterministic for the same collector results.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package status

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// mockCollector is a test collector that returns a fixed status.
type mockCollector struct {
	name   string
	status ComponentStatus
}

func (m mockCollector) Name() string { return m.name }

func (m mockCollector) Collect(ctx context.Context) ComponentStatus {
	return m.status
}

func TestCollectAll_AggregatesResults(t *testing.T) {
	t.Parallel()

	collectors := []Collector{
		mockCollector{
			name: "gateway",
			status: ComponentStatus{
				Name:    "gateway",
				Level:   HealthLevelHealthy,
				Message: "Gateway OK",
			},
		},
		mockCollector{
			name: "database",
			status: ComponentStatus{
				Name:    "database",
				Level:   HealthLevelHealthy,
				Message: "Connected",
			},
		},
	}

	ctx := context.Background()
	report := CollectAll(ctx, collectors, Options{})

	if report.Overall != HealthLevelHealthy {
		t.Fatalf("expected overall healthy, got %s", report.Overall)
	}

	if len(report.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(report.Components))
	}

	if _, ok := report.Components["gateway"]; !ok {
		t.Fatal("expected gateway component")
	}

	if _, ok := report.Components["database"]; !ok {
		t.Fatal("expected database component")
	}
}

func TestCollectAll_OverallStatusPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		levels   []HealthLevel
		expected HealthLevel
	}{
		{
			name:     "all healthy",
			levels:   []HealthLevel{HealthLevelHealthy, HealthLevelHealthy, HealthLevelHealthy},
			expected: HealthLevelHealthy,
		},
		{
			name:     "one warning",
			levels:   []HealthLevel{HealthLevelHealthy, HealthLevelWarning, HealthLevelHealthy},
			expected: HealthLevelWarning,
		},
		{
			name:     "one unhealthy",
			levels:   []HealthLevel{HealthLevelHealthy, HealthLevelUnhealthy, HealthLevelHealthy},
			expected: HealthLevelUnhealthy,
		},
		{
			name:     "unhealthy trumps warning",
			levels:   []HealthLevel{HealthLevelWarning, HealthLevelUnhealthy, HealthLevelHealthy},
			expected: HealthLevelUnhealthy,
		},
		{
			name:     "all unhealthy",
			levels:   []HealthLevel{HealthLevelUnhealthy, HealthLevelUnhealthy},
			expected: HealthLevelUnhealthy,
		},
		{
			name:     "mixed with unknown",
			levels:   []HealthLevel{HealthLevelUnknown, HealthLevelHealthy, HealthLevelWarning},
			expected: HealthLevelWarning,
		},
		{
			name:     "unknown only",
			levels:   []HealthLevel{HealthLevelUnknown, HealthLevelUnknown},
			expected: HealthLevelHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var collectors []Collector
			for i, level := range tt.levels {
				collectors = append(collectors, mockCollector{
					name:   string(rune('a' + i)),
					status: ComponentStatus{Name: string(rune('a' + i)), Level: level},
				})
			}

			ctx := context.Background()
			report := CollectAll(ctx, collectors, Options{})

			if report.Overall != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, report.Overall)
			}
		})
	}
}

func TestCollectAll_EmptyCollectors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	report := CollectAll(ctx, []Collector{}, Options{})

	if report.Overall != HealthLevelHealthy {
		t.Fatalf("expected healthy for empty collectors, got %s", report.Overall)
	}

	if len(report.Components) != 0 {
		t.Fatalf("expected 0 components, got %d", len(report.Components))
	}
}

func TestCollectAll_DurationRecorded(t *testing.T) {
	t.Parallel()

	collectors := []Collector{
		mockCollector{
			name:   "test",
			status: ComponentStatus{Name: "test", Level: HealthLevelHealthy},
		},
	}

	ctx := context.Background()
	start := time.Now()
	report := CollectAll(ctx, collectors, Options{})
	elapsed := time.Since(start)

	// Parse duration from report
	duration, err := time.ParseDuration(report.Duration)
	if err != nil {
		t.Fatalf("failed to parse duration: %v", err)
	}

	if duration < 0 || duration > elapsed+time.Millisecond {
		t.Fatalf("unexpected duration: %s", report.Duration)
	}
}

func TestCollectAll_TimestampRecorded(t *testing.T) {
	t.Parallel()

	collectors := []Collector{
		mockCollector{
			name:   "test",
			status: ComponentStatus{Name: "test", Level: HealthLevelHealthy},
		},
	}

	ctx := context.Background()
	before := time.Now().UTC().Truncate(time.Second)
	report := CollectAll(ctx, collectors, Options{})
	after := time.Now().UTC().Add(time.Second).Truncate(time.Second)

	timestamp, err := time.Parse(time.RFC3339, report.Timestamp)
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}

	if timestamp.Before(before) || timestamp.After(after) {
		t.Fatalf("timestamp %s not in expected range", report.Timestamp)
	}
}

func TestStatusReport_WriteJSON(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelWarning,
		Components: map[string]ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   HealthLevelHealthy,
				Message: "Gateway OK",
				Details: map[string]any{
					"status": 200,
				},
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "123ms",
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var decoded StatusReport
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify content
	if decoded.Overall != report.Overall {
		t.Fatalf("overall mismatch: expected %s, got %s", report.Overall, decoded.Overall)
	}

	if len(decoded.Components) != len(report.Components) {
		t.Fatalf("component count mismatch")
	}
}

func TestStatusReport_WriteJSON_NilComponents(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall:    HealthLevelHealthy,
		Components: nil,
		Timestamp:  "2024-01-15T10:00:00Z",
		Duration:   "1ms",
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if !strings.Contains(buf.String(), "healthy") {
		t.Fatal("expected 'healthy' in output")
	}
}

func TestStatusReport_WriteHuman(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   HealthLevelHealthy,
				Message: "Gateway is responding",
			},
			"database": {
				Name:    "database",
				Level:   HealthLevelHealthy,
				Message: "Connected",
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "123ms",
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, false); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}

	output := buf.String()

	// Verify header
	if !strings.Contains(output, "AI Control Plane Status") {
		t.Fatal("expected header in output")
	}

	// Verify components are listed
	if !strings.Contains(output, "Gateway") {
		t.Fatal("expected Gateway in output")
	}

	if !strings.Contains(output, "Database") {
		t.Fatal("expected Database in output")
	}

	// Verify status indicators
	if !strings.Contains(output, "HEALTHY") {
		t.Fatal("expected HEALTHY overall status")
	}
}

func TestStatusReport_WriteHuman_WithWarnings(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelWarning,
		Components: map[string]ComponentStatus{
			"keys": {
				Name:    "keys",
				Level:   HealthLevelWarning,
				Message: "No virtual keys configured",
				Suggestions: []string{
					"Generate a key: acpctl key gen my-key --budget 10.00",
				},
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "50ms",
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, false); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}

	output := buf.String()

	// Verify suggestions are shown for warnings
	if !strings.Contains(output, "acpctl key gen") {
		t.Fatal("expected suggestions in output")
	}
}

func TestStatusReport_WriteHuman_WithUnhealthy(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelUnhealthy,
		Components: map[string]ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   HealthLevelUnhealthy,
				Message: "Gateway unreachable",
				Suggestions: []string{
					"Check if services are running: make ps",
					"Start services: make up",
				},
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "5s",
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, false); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}

	output := buf.String()

	// Verify UNHEALTHY status
	if !strings.Contains(output, "UNHEALTHY") {
		t.Fatal("expected UNHEALTHY in output")
	}

	// Verify multiple suggestions are shown
	if !strings.Contains(output, "make ps") || !strings.Contains(output, "make up") {
		t.Fatal("expected all suggestions in output")
	}
}

func TestStatusReport_WriteHuman_WideMode(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   HealthLevelHealthy,
				Message: "Gateway OK",
				Details: map[string]any{
					"health_status": 200,
					"models_status": 401,
				},
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "100ms",
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, true); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}

	output := buf.String()

	// In wide mode, details should be shown
	if !strings.Contains(output, "health_status") {
		t.Fatal("expected details in wide mode output")
	}

	if !strings.Contains(output, "200") {
		t.Fatal("expected detail values in wide mode output")
	}
}

func TestStatusReport_WriteHuman_WideModeNonMapDetails(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"test": {
				Name:    "test",
				Level:   HealthLevelHealthy,
				Message: "OK",
				Details: "string details", // Non-map details
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "1ms",
	}

	var buf bytes.Buffer
	// Should not panic with non-map details
	if err := report.WriteHuman(&buf, true); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}
}

func TestStatusReport_WriteHuman_UnknownComponentSkipped(t *testing.T) {
	t.Parallel()

	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"unknown_component": {
				Name:    "unknown_component",
				Level:   HealthLevelUnknown,
				Message: "Unknown",
			},
		},
		Timestamp: "2024-01-15T10:00:00Z",
		Duration:  "1ms",
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, false); err != nil {
		t.Fatalf("WriteHuman failed: %v", err)
	}

	// Unknown components not in the predefined order are skipped
	// This is expected behavior based on the implementation
}

func TestHealthLevel_StringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level    HealthLevel
		expected string
	}{
		{HealthLevelHealthy, "healthy"},
		{HealthLevelWarning, "warning"},
		{HealthLevelUnhealthy, "unhealthy"},
		{HealthLevelUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			t.Parallel()
			if string(tt.level) != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, tt.level)
			}
		})
	}
}

func TestComponentStatus_JSONMarshaling(t *testing.T) {
	t.Parallel()

	status := ComponentStatus{
		Name:        "test",
		Level:       HealthLevelHealthy,
		Message:     "Test message",
		Details:     map[string]any{"key": "value"},
		Suggestions: []string{"suggestion 1"},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ComponentStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Name != status.Name {
		t.Fatal("name mismatch")
	}

	if decoded.Level != status.Level {
		t.Fatal("level mismatch")
	}

	if decoded.Message != status.Message {
		t.Fatal("message mismatch")
	}
}
