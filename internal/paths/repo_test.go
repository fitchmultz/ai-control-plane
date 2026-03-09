// Package paths provides canonical repository-relative path helpers.
//
// Purpose:
//   - Verify canonical repo path helper behavior.
//
// Responsibilities:
//   - Cover relative and absolute path resolution.
//   - Cover common demo path helpers used by commands.
//
// Scope:
//   - Repo path helper tests only.
//
// Usage:
//   - Run via `go test ./internal/paths`.
//
// Invariants/Assumptions:
//   - Helpers remain deterministic for equivalent inputs.
package paths

import "testing"

func TestResolveRepoPath(t *testing.T) {
	if got := ResolveRepoPath("/repo", "demo/logs/output"); got != "/repo/demo/logs/output" {
		t.Fatalf("ResolveRepoPath() = %q", got)
	}
	if got := ResolveRepoPath("/repo", "/tmp/output"); got != "/tmp/output" {
		t.Fatalf("ResolveRepoPath() absolute = %q", got)
	}
	if got := DemoEnvPath("/repo"); got != "/repo/demo/.env" {
		t.Fatalf("DemoEnvPath() = %q", got)
	}
	if got := ReleaseBundlesPath("/repo"); got != "/repo/demo/logs/release-bundles" {
		t.Fatalf("ReleaseBundlesPath() = %q", got)
	}
}
