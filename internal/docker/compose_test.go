// compose_test.go - Tests for Docker Compose argument construction.
//
// Purpose:
//
//	Verify project/file scoping stays deterministic for typed Docker helpers.
//
// Responsibilities:
//   - Confirm default compose invocations target the tracked main file.
//   - Confirm explicit compose options add project-name and alternate files.
//
// Scope:
//   - Argument construction only.
//
// Usage:
//   - Run via `go test ./internal/docker`.
//
// Invariants/Assumptions:
//   - Tests do not require Docker to be installed.
package docker

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestComposeBuildArgs_DefaultFile(t *testing.T) {
	t.Parallel()

	compose := &Compose{
		cmd:        "docker",
		argsPrefix: []string{"compose"},
		projectDir: "/repo/demo",
	}

	got := compose.buildArgs("ps")
	want := []string{
		"compose",
		"-f", filepath.Join("/repo/demo", "docker-compose.yml"),
		"--project-directory", "/repo/demo",
		"ps",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildArgs() = %#v, want %#v", got, want)
	}
}

func TestComposeBuildArgs_WithOptions(t *testing.T) {
	t.Parallel()

	compose := &Compose{
		cmd:         "docker",
		argsPrefix:  []string{"compose"},
		projectDir:  "/repo/demo",
		projectName: "ai-control-plane-ci-runtime",
		files:       []string{"docker-compose.offline.yml"},
	}

	got := compose.buildArgs("ps")
	want := []string{
		"compose",
		"--project-name", "ai-control-plane-ci-runtime",
		"-f", filepath.Join("/repo/demo", "docker-compose.offline.yml"),
		"--project-directory", "/repo/demo",
		"ps",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildArgs() = %#v, want %#v", got, want)
	}
}
