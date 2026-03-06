// profile.go - Benchmark profile loading for the performance baseline.
//
// Purpose:
//
//	Load runnable benchmark workload profiles from the repository's canonical
//	threshold configuration.
//
// Responsibilities:
//   - Parse benchmark profile configuration JSON
//   - Resolve named workload profiles for CLI and Make workflows
//
// Scope:
//   - Covers local profile definitions only
//   - Does not execute benchmarks directly
//
// Usage:
//   - Called from `acpctl benchmark baseline --profile <name>`
//
// Invariants/Assumptions:
//   - Profile definitions are sourced from `demo/config/benchmark_thresholds.json`
//   - Unknown profile names are rejected
package performance

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ProfileCatalog is the canonical benchmark threshold configuration format.
type ProfileCatalog struct {
	SchemaVersion string                      `json:"schema_version"`
	Defaults      BenchmarkWorkloadProfile    `json:"defaults"`
	Profiles      map[string]BenchmarkProfile `json:"profiles"`
}

// BenchmarkProfile describes one runnable benchmark profile.
type BenchmarkProfile struct {
	Description string                   `json:"description"`
	Workload    BenchmarkWorkloadProfile `json:"workload"`
}

// BenchmarkWorkloadProfile defines the benchmark shape for a named profile.
type BenchmarkWorkloadProfile struct {
	WarmupRequests   int `json:"warmup_requests"`
	MeasuredRequests int `json:"measured_requests"`
	Concurrency      int `json:"concurrency"`
}

// LoadProfileCatalog loads the benchmark profile catalog from disk.
func LoadProfileCatalog(path string) (*ProfileCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile catalog: %w", err)
	}
	var catalog ProfileCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse profile catalog: %w", err)
	}
	if len(catalog.Profiles) == 0 {
		return nil, fmt.Errorf("profile catalog has no profiles")
	}
	return &catalog, nil
}

// ResolveProfile returns the named benchmark profile.
func (c *ProfileCatalog) ResolveProfile(name string) (BenchmarkProfile, error) {
	if c == nil {
		return BenchmarkProfile{}, fmt.Errorf("profile catalog is nil")
	}
	profile, ok := c.Profiles[name]
	if !ok {
		return BenchmarkProfile{}, fmt.Errorf("unknown benchmark profile %q (available: %s)", name, joinProfileNames(c))
	}
	return profile, nil
}

func joinProfileNames(c *ProfileCatalog) string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Sprintf("%v", names)
}
