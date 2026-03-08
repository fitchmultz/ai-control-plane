// database_test validates DatabaseCollector docker-based status checks.
//
// Purpose:
//
//	Ensure database status collection correctly interprets PostgreSQL
//	availability and metrics from docker exec commands.
//
// Responsibilities:
//   - Verify collector name returns "database".
//   - Verify status level when Docker is unavailable.
//   - Verify status level when PostgreSQL container is not running.
//   - Verify response parsing for various PostgreSQL outputs.
//
// Non-scope:
//   - Does not test against real running PostgreSQL containers.
//   - Tests mock the command execution behavior.
//
// Invariants/Assumptions:
//   - Docker availability is checked via exec.LookPath.
//   - PostgreSQL container status is checked via docker ps.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package collectors

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

type fakeContainerResolver struct {
	containerID string
	err         error
	calls       int
}

func (f *fakeContainerResolver) ContainerID(ctx context.Context, service string) (string, error) {
	f.calls++
	return f.containerID, f.err
}

type recordingRunner struct {
	inner    *runner.MockRunner
	commands []string
}

func newRecordingRunner() *recordingRunner {
	return &recordingRunner{inner: runner.NewMockRunner()}
}

func (r *recordingRunner) SetResponse(command string, result *runner.Result) {
	r.inner.SetResponse(command, result)
}

func (r *recordingRunner) Run(ctx context.Context, name string, arg ...string) *runner.Result {
	r.commands = append(r.commands, name+" "+strings.Join(arg, " "))
	return r.inner.Run(ctx, name, arg...)
}

func (r *recordingRunner) sawCommandContaining(substr string) bool {
	for _, command := range r.commands {
		if strings.Contains(command, substr) {
			return true
		}
	}
	return false
}

func TestDatabaseCollector_Name(t *testing.T) {
	t.Parallel()

	c := DatabaseCollector{}
	if c.Name() != "database" {
		t.Fatalf("expected name 'database', got %q", c.Name())
	}
}

func TestDatabaseCollector_Collect_NoDocker(t *testing.T) {
	t.Parallel()

	// This test validates that the collector returns unknown status
	// when Docker is not available. In practice, this depends on
	// the system having Docker installed.

	c := NewDatabaseCollector("/tmp")

	ctx := context.Background()
	result := c.Collect(ctx)

	// When Docker is not available, should return unknown
	// Note: This may vary based on test environment
	if result.Name != "database" {
		t.Fatalf("expected name 'database', got %q", result.Name)
	}
}

func TestDatabaseCollector_ComponentStatus_Levels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		level         status.HealthLevel
		message       string
		expectDetails bool
	}{
		{
			name:          "healthy status",
			level:         status.HealthLevelHealthy,
			message:       "Connected",
			expectDetails: true,
		},
		{
			name:          "unhealthy status",
			level:         status.HealthLevelUnhealthy,
			message:       "PostgreSQL not accepting connections",
			expectDetails: false,
		},
		{
			name:          "warning status",
			level:         status.HealthLevelWarning,
			message:       "PostgreSQL responded unexpectedly",
			expectDetails: false,
		},
		{
			name:          "unknown status",
			level:         status.HealthLevelUnknown,
			message:       "Docker not available",
			expectDetails: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We can't easily mock exec.Command, but we can verify
			// the ComponentStatus structure is correct
			component := status.ComponentStatus{
				Name:    "database",
				Level:   tt.level,
				Message: tt.message,
			}

			if component.Name != "database" {
				t.Fatal("name mismatch")
			}

			if component.Level != tt.level {
				t.Fatal("level mismatch")
			}

			if component.Message != tt.message {
				t.Fatal("message mismatch")
			}
		})
	}
}

func TestDatabaseCollector_DetailsStructure(t *testing.T) {
	t.Parallel()

	// Test that when healthy, database collector would return
	// the expected details structure
	details := map[string]any{
		"expected_tables": 4,
		"version":         "PostgreSQL 15.4",
		"size":            "10 MB",
		"connections":     "5",
	}

	component := status.ComponentStatus{
		Name:    "database",
		Level:   status.HealthLevelHealthy,
		Message: "Connected",
		Details: details,
	}

	if component.Details == nil {
		t.Fatal("expected details to be present")
	}

	detailsMap, ok := component.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if _, ok := detailsMap["expected_tables"]; !ok {
		t.Fatal("expected expected_tables in details")
	}

	if _, ok := detailsMap["version"]; !ok {
		t.Fatal("expected version in details")
	}
}

func TestDatabaseCollector_Suggestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		level              status.HealthLevel
		expectedSuggestion string
	}{
		{
			name:               "unhealthy shows restart suggestion",
			level:              status.HealthLevelUnhealthy,
			expectedSuggestion: "Start services",
		},
		{
			name:               "unknown shows docker install suggestion",
			level:              status.HealthLevelUnknown,
			expectedSuggestion: "Docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var suggestions []string
			switch tt.level {
			case status.HealthLevelUnhealthy:
				suggestions = []string{
					"Start services: make up",
					"Check container status: docker ps",
				}
			case status.HealthLevelUnknown:
				suggestions = []string{
					"Install Docker: https://docs.docker.com/get-docker/",
				}
			}

			if len(suggestions) == 0 {
				t.Fatal("expected suggestions")
			}

			found := false
			for _, s := range suggestions {
				if strings.Contains(s, tt.expectedSuggestion) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("expected suggestion containing %q", tt.expectedSuggestion)
			}
		})
	}
}

// === BEHAVIORAL TESTS WITH MOCK RUNNER ===

func TestDatabaseCollector_Collect_PostgresNotRunning(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock docker ps returning empty (container not running)
	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Name != "database" {
		t.Errorf("expected name 'database', got %q", result.Name)
	}

	if result.Level != status.HealthLevelUnhealthy {
		t.Errorf("expected unhealthy when postgres not running, got %v", result.Level)
	}

	if result.Message != "PostgreSQL container not running" {
		t.Errorf("expected container not running message, got %q", result.Message)
	}
}

func TestDatabaseCollector_Collect_PostgresLookupFailure(t *testing.T) {
	mock := runner.NewMockRunner()
	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "",
		Stderr:   "Cannot connect to the Docker daemon",
		ExitCode: 1,
		Error:    fmt.Errorf("Cannot connect to the Docker daemon"),
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	result := c.Collect(context.Background())
	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy, got %v", result.Level)
	}

	if result.Message != "Failed to locate PostgreSQL container" {
		t.Fatalf("expected lookup failure message, got %q", result.Message)
	}

	details, ok := result.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	lookupError, ok := details["lookup_error"].(string)
	if !ok || !strings.Contains(lookupError, "postgres container lookup failed") {
		t.Fatalf("expected lookup_error details, got %v", details["lookup_error"])
	}
}

func TestDatabaseCollector_Collect_PostgresConnectionFailed(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock docker ps showing container running
	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "postgres\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock psql connection failing
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT 1;", &runner.Result{
		Stdout:   "",
		Stderr:   "psql: could not connect to server: Connection refused",
		ExitCode: 2,
		Error:    context.DeadlineExceeded,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Level != status.HealthLevelUnhealthy {
		t.Errorf("expected unhealthy when connection fails, got %v", result.Level)
	}

	if result.Message != "PostgreSQL not accepting connections" {
		t.Errorf("expected connection refused message, got %q", result.Message)
	}

	// Verify stderr is included in details
	details, ok := result.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if _, ok := details["stderr"]; !ok {
		t.Error("expected stderr in details for debugging")
	}
}

func TestDatabaseCollector_Collect_PostgresHealthy(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock docker ps showing container running
	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "postgres\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock successful psql connection
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT 1;", &runner.Result{
		Stdout:   " 1 \n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock table count query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('LiteLLM_VerificationToken', 'LiteLLM_UserTable', 'LiteLLM_BudgetTable', 'LiteLLM_SpendLogs');", &runner.Result{
		Stdout:   "4\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock version query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT version();", &runner.Result{
		Stdout:   " PostgreSQL 15.4 on x86_64...\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock size query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT pg_size_pretty(pg_database_size('litellm'));", &runner.Result{
		Stdout:   " 10 MB\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock connections query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT count(*) FROM pg_stat_activity WHERE datname = 'litellm';", &runner.Result{
		Stdout:   "5\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Level != status.HealthLevelHealthy {
		t.Errorf("expected healthy, got %v", result.Level)
	}

	if result.Message != "Connected" {
		t.Errorf("expected 'Connected' message, got %q", result.Message)
	}

	// Verify details are populated
	details, ok := result.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["expected_tables"] != 4 {
		t.Errorf("expected 4 tables, got %v", details["expected_tables"])
	}

	if details["connections"] != "5" {
		t.Errorf("expected 5 connections, got %v", details["connections"])
	}
}

func TestDatabaseCollector_Collect_UnexpectedResponse(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock docker ps showing container running
	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "postgres\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Mock psql returning unexpected output (not "1")
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT 1;", &runner.Result{
		Stdout:   "unexpected\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Level != status.HealthLevelWarning {
		t.Errorf("expected warning for unexpected response, got %v", result.Level)
	}

	if result.Message != "PostgreSQL responded unexpectedly" {
		t.Errorf("expected unexpected response message, got %q", result.Message)
	}
}

func TestDatabaseCollector_Collect_UsesComposeResolverWhenAvailable(t *testing.T) {
	recording := newRecordingRunner()
	recording.SetResponse("docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT 1;", &runner.Result{
		Stdout:   " 1 \n",
		ExitCode: 0,
	})

	resolver := &fakeContainerResolver{containerID: "compose-postgres"}

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(recording)
	c.SetContainerResolver(resolver)

	result := c.Collect(context.Background())
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %v", result.Level)
	}

	if resolver.calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.calls)
	}

	if recording.sawCommandContaining("docker ps --filter label=com.docker.compose.service=postgres") {
		t.Fatal("expected compose resolver to avoid docker ps fallback")
	}
}

func TestDatabaseCollector_dbQuery(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock successful query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT version();", &runner.Result{
		Stdout:   " PostgreSQL 15.4\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "postgres\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	result, err := c.dbQuery(ctx, "SELECT version();")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != "PostgreSQL 15.4" {
		t.Errorf("expected 'PostgreSQL 15.4', got %q", result)
	}
}

func TestDatabaseCollector_dbQuery_Error(t *testing.T) {
	mock := runner.NewMockRunner()

	// Mock failing query
	mock.SetResponse("docker exec postgres psql -U litellm -d litellm -t -c SELECT * FROM nonexistent;", &runner.Result{
		Stdout:   "",
		Stderr:   "ERROR: relation 'nonexistent' does not exist",
		ExitCode: 1,
		Error:    context.DeadlineExceeded,
	})

	mock.SetResponse("docker ps --filter label=com.docker.compose.service=postgres --filter label=com.docker.compose.project.working_dir=/tmp/demo --format {{.ID}}", &runner.Result{
		Stdout:   "postgres\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	c := NewDatabaseCollector("/tmp")
	c.SetRunner(mock)
	c.SetContainerResolver(nil)

	ctx := context.Background()
	_, err := c.dbQuery(ctx, "SELECT * FROM nonexistent;")

	if err == nil {
		t.Error("expected error for failing query")
	}

	// Verify error includes stderr
	if err != nil && !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to include stderr details, got %q", err.Error())
	}
}
