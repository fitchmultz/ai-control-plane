// Package status provides aggregated system health status collection.
//
// Purpose:
//
//	Collect and aggregate typed runtime health status from ACP subsystems into
//	one report model shared by status, doctor, and health workflows.
//
// Responsibilities:
//   - Define canonical collector and report types.
//   - Aggregate concurrent component collection into StatusReport.
//   - Provide JSON and human-readable report rendering.
//   - Centralize typed detail rendering for wide output.
//
// Non-scope:
//   - Does not execute remediation actions.
//   - Does not modify system state.
//
// Invariants/Assumptions:
//   - All collectors are read-only operations.
//   - Collector failures do not prevent other collectors from running.
//   - Wide output renders a stable detail field order.
//
// Scope:
//   - File-local implementation and report types only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	sharedhealth "github.com/mitchfultz/ai-control-plane/internal/health"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

// HealthLevel represents the health status of a component.
type HealthLevel = sharedhealth.Level

const (
	HealthLevelHealthy   = sharedhealth.LevelHealthy
	HealthLevelWarning   = sharedhealth.LevelWarning
	HealthLevelUnhealthy = sharedhealth.LevelUnhealthy
	HealthLevelUnknown   = sharedhealth.LevelUnknown
)

const LookupErrorDatabaseConfigAmbiguous = "database_config_ambiguous"

// ComponentDetails captures typed supplemental runtime details.
type ComponentDetails struct {
	Mode                   string   `json:"mode,omitempty"`
	Scheme                 string   `json:"scheme,omitempty"`
	BaseURL                string   `json:"base_url,omitempty"`
	DatabaseName           string   `json:"database_name,omitempty"`
	DatabaseUser           string   `json:"database_user,omitempty"`
	ContainerID            string   `json:"container_id,omitempty"`
	HTTPStatus             int      `json:"http_status,omitempty"`
	ModelsHTTPStatus       int      `json:"models_http_status,omitempty"`
	HealthReachable        bool     `json:"health_reachable,omitempty"`
	ModelsReachable        bool     `json:"models_reachable,omitempty"`
	HealthAuthorized       bool     `json:"health_authorized,omitempty"`
	ModelsAuthorized       bool     `json:"models_authorized,omitempty"`
	MasterKeyConfigured    bool     `json:"master_key_configured,omitempty"`
	TLSEnabled             bool     `json:"tls_enabled,omitempty"`
	Reachable              bool     `json:"reachable,omitempty"`
	Authorized             bool     `json:"authorized,omitempty"`
	ExpectedTables         int      `json:"expected_tables,omitempty"`
	Version                string   `json:"version,omitempty"`
	Size                   string   `json:"size,omitempty"`
	Connections            int      `json:"connections,omitempty"`
	TotalKeys              int      `json:"total_keys,omitempty"`
	ActiveKeys             int      `json:"active_keys,omitempty"`
	ExpiredKeys            int      `json:"expired_keys,omitempty"`
	TotalBudgets           int      `json:"total_budgets,omitempty"`
	HighUtilizationBudgets int      `json:"high_utilization_budgets,omitempty"`
	ExhaustedBudgets       int      `json:"exhausted_budgets,omitempty"`
	SpendLogsTableExists   bool     `json:"spend_logs_table_exists,omitempty"`
	HighSeverityFindings   int      `json:"high_severity_findings,omitempty"`
	MediumSeverityFindings int      `json:"medium_severity_findings,omitempty"`
	UniqueModels24h        int      `json:"unique_models_24h,omitempty"`
	TotalEntries24h        int      `json:"total_entries_24h,omitempty"`
	RequiredPorts          []int    `json:"required_ports,omitempty"`
	OccupiedPorts          []int    `json:"occupied_ports,omitempty"`
	MissingVars            []string `json:"missing_vars,omitempty"`
	MissingFiles           []string `json:"missing_files,omitempty"`
	AuthStatus             string   `json:"auth_status,omitempty"`
	LookupError            string   `json:"lookup_error,omitempty"`
	Response               string   `json:"response,omitempty"`
	Error                  string   `json:"error,omitempty"`
}

// IsZero reports whether the details struct contains no meaningful values.
func (d ComponentDetails) IsZero() bool {
	return len(d.lines()) == 0
}

// Lines returns stable human-readable detail lines for wide output.
func (d ComponentDetails) Lines() []string {
	return d.lines()
}

func (d ComponentDetails) lines() []string {
	lines := make([]string, 0, 24)
	appendText := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", label, value))
		}
	}
	appendBool := func(label string, value bool) {
		if value {
			lines = append(lines, fmt.Sprintf("%s: %t", label, value))
		}
	}
	appendInt := func(label string, value int) {
		if value != 0 {
			lines = append(lines, fmt.Sprintf("%s: %d", label, value))
		}
	}
	appendPorts := func(label string, ports []int) {
		if len(ports) == 0 {
			return
		}
		parts := make([]string, len(ports))
		for i, port := range ports {
			parts[i] = strconv.Itoa(port)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, strings.Join(parts, ", ")))
	}
	appendList := func(label string, values []string) {
		if len(values) == 0 {
			return
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, strings.Join(values, ", ")))
	}

	appendText("mode", d.Mode)
	appendText("scheme", d.Scheme)
	appendText("base_url", d.BaseURL)
	appendText("database_name", d.DatabaseName)
	appendText("database_user", d.DatabaseUser)
	appendText("container_id", d.ContainerID)
	appendInt("http_status", d.HTTPStatus)
	appendInt("models_http_status", d.ModelsHTTPStatus)
	appendBool("health_reachable", d.HealthReachable)
	appendBool("models_reachable", d.ModelsReachable)
	appendBool("health_authorized", d.HealthAuthorized)
	appendBool("models_authorized", d.ModelsAuthorized)
	appendBool("master_key_configured", d.MasterKeyConfigured)
	appendBool("tls_enabled", d.TLSEnabled)
	appendBool("reachable", d.Reachable)
	appendBool("authorized", d.Authorized)
	appendInt("expected_tables", d.ExpectedTables)
	appendText("version", d.Version)
	appendText("size", d.Size)
	appendInt("connections", d.Connections)
	appendInt("total_keys", d.TotalKeys)
	appendInt("active_keys", d.ActiveKeys)
	appendInt("expired_keys", d.ExpiredKeys)
	appendInt("total_budgets", d.TotalBudgets)
	appendInt("high_utilization_budgets", d.HighUtilizationBudgets)
	appendInt("exhausted_budgets", d.ExhaustedBudgets)
	appendBool("spend_logs_table_exists", d.SpendLogsTableExists)
	appendInt("high_severity_findings", d.HighSeverityFindings)
	appendInt("medium_severity_findings", d.MediumSeverityFindings)
	appendInt("unique_models_24h", d.UniqueModels24h)
	appendInt("total_entries_24h", d.TotalEntries24h)
	appendPorts("required_ports", d.RequiredPorts)
	appendPorts("occupied_ports", d.OccupiedPorts)
	appendList("missing_vars", d.MissingVars)
	appendList("missing_files", d.MissingFiles)
	appendText("auth_status", d.AuthStatus)
	appendText("lookup_error", d.LookupError)
	appendText("response", d.Response)
	appendText("error", d.Error)

	return lines
}

// ComponentStatus represents the status of a single component.
type ComponentStatus struct {
	Name        string           `json:"name"`
	Level       HealthLevel      `json:"level"`
	Message     string           `json:"message"`
	Details     ComponentDetails `json:"details,omitempty"`
	Suggestions []string         `json:"suggestions,omitempty"`
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
	Name() string
	Collect(ctx context.Context) ComponentStatus
}

// Options configures status collection behavior.
type Options struct {
	RepoRoot string
	Wide     bool
}

// CollectAll runs all collectors and returns an aggregated report.
func CollectAll(ctx context.Context, collectors []Collector, opts Options) StatusReport {
	start := time.Now()
	var wg sync.WaitGroup
	results := make(map[string]ComponentStatus)
	var mu sync.Mutex

	for _, collector := range collectors {
		wg.Add(1)
		go func(c Collector) {
			defer wg.Done()
			component := c.Collect(ctx)
			mu.Lock()
			results[c.Name()] = component
			mu.Unlock()
		}(collector)
	}

	wg.Wait()
	duration := time.Since(start)

	overall := HealthLevelHealthy
	for _, component := range results {
		overall = sharedhealth.Worst(overall, component.Level)
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

	fmt.Fprintln(w, colors.Bold+"=== AI Control Plane Status ==="+colors.Reset)
	fmt.Fprintln(w)

	for _, name := range DefaultComponentOrder {
		component, ok := r.Components[name]
		if !ok {
			continue
		}
		paddedName := fmt.Sprintf("%-11s", strings.ToUpper(name[:1])+name[1:])
		fmt.Fprintf(w, "%s %s %s\n", paddedName, formatStatus(component.Level), component.Message)

		if len(component.Suggestions) > 0 && (component.Level == HealthLevelUnhealthy || component.Level == HealthLevelWarning) {
			for _, suggestion := range component.Suggestions {
				fmt.Fprintf(w, "             %s\n", suggestion)
			}
		}

		if wide && !component.Details.IsZero() {
			for _, line := range component.Details.lines() {
				fmt.Fprintf(w, "             %s\n", line)
			}
		}
	}

	fmt.Fprintln(w)
	overall := "UNKNOWN"
	switch r.Overall {
	case HealthLevelHealthy:
		overall = sf.Healthy()
	case HealthLevelWarning:
		overall = sf.Warning()
	case HealthLevelUnhealthy:
		overall = sf.Unhealthy()
	}
	fmt.Fprintf(w, "Overall: %s (%s)\n", overall, r.Duration)
	return nil
}

// WriteJSON writes the report as JSON.
func (r StatusReport) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}
