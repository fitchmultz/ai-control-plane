// check_docker_test.go - Focused coverage for Docker prerequisite checks.
//
// Purpose:
//   - Verify docker availability check metadata and result shaping.
//
// Responsibilities:
//   - Cover stable IDs and result severity envelopes.
//
// Scope:
//   - Docker prerequisite checks only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests do not require Docker to be installed.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestDockerAvailableCheckID(t *testing.T) {
	t.Parallel()
	check := dockerAvailableCheck{}
	if check.ID() != "docker_available" {
		t.Fatalf("expected ID docker_available, got %s", check.ID())
	}
}

func TestDockerAvailableCheckRun(t *testing.T) {
	t.Parallel()

	result := dockerAvailableCheck{}.Run(context.Background(), Options{})
	if result.ID != "docker_available" {
		t.Fatalf("expected ID docker_available, got %s", result.ID)
	}
	if result.Name != "Docker Available" {
		t.Fatalf("expected Docker Available name, got %s", result.Name)
	}
	if result.Level != status.HealthLevelHealthy && result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("unexpected level: %v", result.Level)
	}
	if result.Severity != SeverityPrereq && result.Severity != SeverityDomain {
		t.Fatalf("unexpected severity: %v", result.Severity)
	}
}
