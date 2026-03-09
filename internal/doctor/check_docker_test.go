// check_docker_test.go - Focused coverage for Docker prerequisite checks.
//
// Purpose:
//   - Verify docker availability checks stay deterministic across environments.
//
// Responsibilities:
//   - Cover docker-not-found, permission, daemon, and success branches.
//   - Verify stable IDs and severity mappings.
//
// Scope:
//   - Docker prerequisite checks only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests do not require Docker to be installed on the host.
package doctor

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestDockerAvailableCheckID(t *testing.T) {
	t.Parallel()
	check := dockerAvailableCheck{}
	if check.ID() != "docker_available" {
		t.Fatalf("expected ID docker_available, got %s", check.ID())
	}
}

func TestDockerAvailableCheckRun_CoversDeterministicBranches(t *testing.T) {
	tests := []struct {
		name        string
		result      proc.Result
		wantLevel   status.HealthLevel
		wantMessage string
		wantError   string
		wantSuggest string
	}{
		{
			name: "docker not found",
			result: proc.Result{
				Err: &proc.ExecError{Name: "docker", Kind: proc.KindNotFound, Err: exec.ErrNotFound},
			},
			wantLevel:   status.HealthLevelUnhealthy,
			wantMessage: "Docker not found in PATH",
			wantSuggest: "Install Docker: https://docs.docker.com/get-docker/",
		},
		{
			name: "permission denied",
			result: proc.Result{
				Err:    errors.New("exit status 1"),
				Stderr: "permission denied while trying to connect to the Docker daemon socket",
			},
			wantLevel:   status.HealthLevelUnhealthy,
			wantMessage: "Docker daemon requires permissions",
			wantError:   "permission denied while trying to connect to the Docker daemon socket",
			wantSuggest: "Add your user to the docker group if required",
		},
		{
			name: "daemon unavailable",
			result: proc.Result{
				Err:    errors.New("exit status 1"),
				Stderr: "Cannot connect to the Docker daemon",
			},
			wantLevel:   status.HealthLevelUnhealthy,
			wantMessage: "Docker daemon not accessible",
			wantError:   "Cannot connect to the Docker daemon",
			wantSuggest: "Ensure Docker service is running",
		},
		{
			name:        "healthy",
			result:      proc.Result{Stdout: "Server Version: 28.0.0"},
			wantLevel:   status.HealthLevelHealthy,
			wantMessage: "Docker is available and daemon is accessible",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			previous := runDockerInfo
			runDockerInfo = func(context.Context) proc.Result { return tt.result }
			t.Cleanup(func() { runDockerInfo = previous })

			result := dockerAvailableCheck{}.Run(context.Background(), Options{})
			if result.Level != tt.wantLevel {
				t.Fatalf("Level = %v, want %v", result.Level, tt.wantLevel)
			}
			if result.Message != tt.wantMessage {
				t.Fatalf("Message = %q, want %q", result.Message, tt.wantMessage)
			}
			if tt.wantError != "" && result.Details.Error != tt.wantError {
				t.Fatalf("Details.Error = %q, want %q", result.Details.Error, tt.wantError)
			}
			if tt.wantSuggest != "" && (len(result.Suggestions) == 0 || result.Suggestions[0] == "") {
				t.Fatalf("expected suggestions, got %+v", result.Suggestions)
			}
		})
	}
}
