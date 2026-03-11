// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Enumerate tracked repository files for security scanners.
//
// Responsibilities:
//   - Query git for the canonical tracked-file inventory.
//   - Normalize tracked paths for deterministic downstream scans.
//   - Keep command execution isolated from validator logic.
//
// Scope:
//   - Read-only tracked-file enumeration only.
//
// Usage:
//   - Called by secrets audit and public-hygiene checks.
//
// Invariants/Assumptions:
//   - Returned paths are repository-relative and stably sorted.
//   - Enumeration uses git, not filesystem heuristics.
package security

import (
	"bytes"
	"context"
	"path/filepath"
	"sort"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

func ListTrackedFiles(ctx context.Context, repoRoot string) ([]string, error) {
	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    []string{"ls-files", "-z"},
		Dir:     repoRoot,
		Timeout: 5_000_000_000,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	rawPaths := bytes.Split([]byte(res.Stdout), []byte{0})
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		if len(rawPath) == 0 {
			continue
		}
		relPath := filepath.ToSlash(filepath.Clean(string(rawPath)))
		if policy.MatchAnyGlob(relPath, []string{"deploy/incubating/**"}) {
			continue
		}
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	return paths, nil
}
