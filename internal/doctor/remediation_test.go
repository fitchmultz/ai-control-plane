// remediation_test.go - Coverage for doctor remediation helpers.
//
// Purpose:
//   - Verify safe doctor remediation hooks behave deterministically.
//
// Responsibilities:
//   - Cover gateway and embedded-database recovery triggers.
//   - Exercise check Fix methods and the no-fix helper contract.
//
// Scope:
//   - Doctor remediation behavior only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests stub compose execution and avoid live Docker interaction.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestRecoverGatewayRuntimeAndFix(t *testing.T) {
	original := runComposeUp
	defer func() { runComposeUp = original }()

	var services []string
	runComposeUp = func(ctx context.Context, repoRoot string, requested ...string) error {
		services = append([]string(nil), requested...)
		return nil
	}

	opts := Options{
		RepoRoot: "/repo",
		RuntimeReport: &status.StatusReport{Components: map[string]status.ComponentStatus{
			"gateway": {
				Name:    "gateway",
				Level:   status.HealthLevelUnhealthy,
				Details: status.ComponentDetails{MasterKeyConfigured: true},
			},
			"database": {
				Name:    "database",
				Level:   status.HealthLevelUnhealthy,
				Details: status.ComponentDetails{Mode: "embedded"},
			},
		}},
	}

	applied, message, err := recoverGatewayRuntime(context.Background(), opts)
	if err != nil || !applied || message == "" {
		t.Fatalf("recoverGatewayRuntime() = %t, %q, %v", applied, message, err)
	}
	if len(services) != 2 || services[0] != "postgres" || services[1] != "litellm" {
		t.Fatalf("unexpected services started: %v", services)
	}

	applied, message, err = (gatewayHealthyCheck{}).Fix(context.Background(), opts)
	if err != nil || !applied || message == "" {
		t.Fatalf("gatewayHealthyCheck.Fix() = %t, %q, %v", applied, message, err)
	}
}

func TestRecoverEmbeddedDatabaseAndNoFixCheck(t *testing.T) {
	original := runComposeUp
	defer func() { runComposeUp = original }()

	var services []string
	runComposeUp = func(ctx context.Context, repoRoot string, requested ...string) error {
		services = append([]string(nil), requested...)
		return nil
	}

	opts := Options{
		RepoRoot: "/repo",
		RuntimeReport: runtimeReportFor("database", status.ComponentDetails{
			Mode: "embedded",
		}, status.HealthLevelUnhealthy, "database down"),
	}

	applied, message, err := recoverEmbeddedDatabase(context.Background(), opts)
	if err != nil || !applied || message == "" {
		t.Fatalf("recoverEmbeddedDatabase() = %t, %q, %v", applied, message, err)
	}
	if len(services) != 1 || services[0] != "postgres" {
		t.Fatalf("unexpected services started: %v", services)
	}

	applied, message, err = (dbConnectableCheck{}).Fix(context.Background(), opts)
	if err != nil || !applied || message == "" {
		t.Fatalf("dbConnectableCheck.Fix() = %t, %q, %v", applied, message, err)
	}

	applied, message, err = (noFixCheck{}).Fix(context.Background(), Options{})
	if err != nil || applied || message != "" {
		t.Fatalf("noFixCheck.Fix() = %t, %q, %v", applied, message, err)
	}
}
