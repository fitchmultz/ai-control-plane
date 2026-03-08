// check_ports_test.go - Focused coverage for port occupancy checks.
//
// Purpose:
//   - Verify doctor port checks classify free, occupied, and ACP-owned ports correctly.
//
// Responsibilities:
//   - Cover port helper behavior and occupied-port message paths.
//
// Scope:
//   - TCP port diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use ephemeral loopback ports for determinism.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestPortsFreeCheckID(t *testing.T) {
	t.Parallel()
	if (portsFreeCheck{}).ID() != "ports_free" {
		t.Fatalf("expected ID ports_free")
	}
}

func TestPortsFreeCheckRunWithFreePort(t *testing.T) {
	t.Parallel()

	freePort, release := reserveLocalPort(t)
	release()

	result := (portsFreeCheck{}).Run(context.Background(), Options{
		RequiredPorts: []int{freePort},
	})
	if result.ID != "ports_free" {
		t.Fatalf("unexpected ID %s", result.ID)
	}
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy result, got %v", result.Level)
	}
}

func TestPortsFreeCheckWithOccupiedPort(t *testing.T) {
	t.Parallel()

	port, release := reserveLocalPort(t)
	defer release()

	result := (portsFreeCheck{}).Run(context.Background(), Options{
		RequiredPorts: []int{port},
	})
	if result.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning, got %v", result.Level)
	}
	if result.Severity != SeverityDomain {
		t.Fatalf("expected domain severity, got %v", result.Severity)
	}
}

func TestOccupiedPortsBelongToRunningACP(t *testing.T) {
	t.Parallel()

	port, release := reserveLocalPort(t)
	defer release()

	opts := Options{
		Gateway: config.GatewaySettings{Host: "127.0.0.1", PortInt: port},
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			HealthReachable: true,
		}, status.HealthLevelHealthy, "Gateway is responding"),
	}

	if !occupiedPortsBelongToRunningACP(context.Background(), []int{port}, opts) {
		t.Fatal("expected gateway port to be recognized as ACP-owned")
	}
	if occupiedPortsBelongToRunningACP(context.Background(), []int{port + 1}, opts) {
		t.Fatal("did not expect unrelated port to be recognized as ACP-owned")
	}
}

func TestIsPortOccupied(t *testing.T) {
	t.Parallel()

	freePort, release := reserveLocalPort(t)
	release()
	if isPortOccupied(context.Background(), "127.0.0.1", freePort) {
		t.Fatalf("expected port %d to be free", freePort)
	}

	occupiedPort, releaseOccupied := reserveLocalPort(t)
	defer releaseOccupied()
	if !isPortOccupied(context.Background(), "127.0.0.1", occupiedPort) {
		t.Fatalf("expected port %d to be occupied", occupiedPort)
	}
}

func TestJoinPorts(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"4000":           joinPorts([]int{4000}),
		"4000|5432":      joinPorts([]int{4000, 5432}),
		"4000|5432|8080": joinPorts([]int{4000, 5432, 8080}),
		"":               joinPorts(nil),
	}

	for expected, got := range tests {
		if got != expected {
			t.Fatalf("joinPorts() = %q, want %q", got, expected)
		}
	}
}
