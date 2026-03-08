// package_test validates the canonical typed status report model.
//
// Purpose:
//
//	Ensure status aggregation and rendering continue to work after the
//	runtime-health cutover to typed detail structs.
//
// Responsibilities:
//   - Verify overall health aggregation.
//   - Verify typed detail rendering in wide human output.
//   - Verify JSON output remains serializable.
//
// Non-scope:
//   - Does not test collector-specific runtime behavior.
//
// Invariants/Assumptions:
//   - Status report rendering remains stable for the typed detail model.
//
// Scope:
//   - Core status model tests only.
//
// Usage:
//   - Used through `go test` for status package coverage.
package status

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

type mockCollector struct {
	name   string
	status ComponentStatus
}

func (m mockCollector) Name() string { return m.name }

func (m mockCollector) Collect(context.Context) ComponentStatus { return m.status }

func TestCollectAllOverallHealth(t *testing.T) {
	report := CollectAll(context.Background(), []Collector{
		mockCollector{name: "gateway", status: ComponentStatus{Name: "gateway", Level: HealthLevelHealthy}},
		mockCollector{name: "database", status: ComponentStatus{Name: "database", Level: HealthLevelWarning}},
	}, Options{})

	if report.Overall != HealthLevelWarning {
		t.Fatalf("expected warning overall, got %s", report.Overall)
	}
	if len(report.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(report.Components))
	}
}

func TestComponentDetailsLines(t *testing.T) {
	details := ComponentDetails{
		BaseURL:             "http://127.0.0.1:4000",
		HTTPStatus:          200,
		ModelsHTTPStatus:    200,
		MasterKeyConfigured: true,
	}

	lines := details.Lines()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "base_url: http://127.0.0.1:4000") {
		t.Fatalf("expected base_url line, got %q", joined)
	}
	if !strings.Contains(joined, "http_status: 200") {
		t.Fatalf("expected http_status line, got %q", joined)
	}
}

func TestStatusReportWriteHumanWide(t *testing.T) {
	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   HealthLevelHealthy,
				Message: "Gateway is responding",
				Details: ComponentDetails{
					BaseURL:             "http://127.0.0.1:4000",
					HTTPStatus:          200,
					ModelsHTTPStatus:    200,
					MasterKeyConfigured: true,
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := report.WriteHuman(&buf, true); err != nil {
		t.Fatalf("WriteHuman returned error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Gateway is responding") {
		t.Fatalf("expected message in output, got %q", output)
	}
	if !strings.Contains(output, "base_url: http://127.0.0.1:4000") {
		t.Fatalf("expected typed detail line in output, got %q", output)
	}
}

func TestStatusReportWriteJSON(t *testing.T) {
	report := StatusReport{
		Overall: HealthLevelHealthy,
		Components: map[string]ComponentStatus{
			"database": {
				Name:    "database",
				Level:   HealthLevelHealthy,
				Message: "Connected",
				Details: ComponentDetails{Mode: "embedded", ExpectedTables: 4},
			},
		},
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "\"expected_tables\": 4") {
		t.Fatalf("expected expected_tables in json output, got %q", output)
	}
}
