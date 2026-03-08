// Package ci implements CI-runtime scope decision logic.
//
// Purpose:
//
//	Decide whether runtime checks should run based on changed files.
//
// Responsibilities:
//   - Support explicit changed-path input for deterministic tests.
//   - Discover changed files from git when explicit paths are not supplied.
//   - Classify docs/test-only changes as safe to skip runtime checks.
//   - Default conservatively to running runtime checks on uncertainty.
//
// Non-scope:
//   - Does not execute runtime checks.
//   - Does not mutate git state or working tree.
//
// Invariants/Assumptions:
//   - Conservative default: unknown states result in ShouldRun=true.
//   - Exit code mapping is handled by caller.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package ci

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

// DecisionOptions controls runtime-scope decision behavior.
type DecisionOptions struct {
	RepoRoot string
	Paths    []string
	CIFull   string
}

// DecisionResult captures the outcome and operator-facing reason.
type DecisionResult struct {
	ShouldRun bool
	Reason    string
}

// IsTruthy returns true for the repository's accepted truthy values.
func IsTruthy(value string) bool {
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	default:
		return false
	}
}

// DecideRuntimeScope determines whether runtime checks should run.
func DecideRuntimeScope(ctx context.Context, options DecisionOptions) (DecisionResult, error) {
	if IsTruthy(options.CIFull) {
		return DecisionResult{ShouldRun: true, Reason: "CI_FULL is set; runtime checks should run."}, nil
	}

	changed := make(map[string]struct{})
	if len(options.Paths) > 0 {
		for _, raw := range options.Paths {
			clean := normalizePath(raw)
			if clean == "" {
				continue
			}
			changed[clean] = struct{}{}
		}
	} else {
		repoRoot := options.RepoRoot
		if repoRoot == "" {
			return DecisionResult{ShouldRun: true, Reason: "Repository root not provided; running runtime checks conservatively."}, nil
		}

		inside, err := runGit(ctx, repoRoot, "rev-parse", "--is-inside-work-tree")
		if err != nil || strings.TrimSpace(inside) != "true" {
			return DecisionResult{ShouldRun: true, Reason: "Not in a git work tree; running runtime checks conservatively."}, nil
		}

		gitCommands := [][]string{
			{"diff", "--name-only", "-z"},
			{"diff", "--cached", "--name-only", "-z"},
			{"ls-files", "--others", "--exclude-standard", "-z"},
		}
		for _, gitArgs := range gitCommands {
			out, runErr := runGit(ctx, repoRoot, gitArgs...)
			if runErr != nil {
				return DecisionResult{ShouldRun: true, Reason: "Unable to inspect git changes; running runtime checks conservatively."}, nil
			}
			for _, file := range parseNullDelimited(out) {
				changed[file] = struct{}{}
			}
		}
	}

	if len(changed) == 0 {
		return DecisionResult{ShouldRun: true, Reason: "No changed files detected; running runtime checks conservatively."}, nil
	}

	paths := make([]string, 0, len(changed))
	for file := range changed {
		paths = append(paths, file)
	}
	sort.Strings(paths)

	if slices.ContainsFunc(paths, isRuntimeImpacting) {
		return DecisionResult{ShouldRun: true, Reason: "Runtime-impacting changes detected; runtime checks should run."}, nil
	}

	return DecisionResult{ShouldRun: false, Reason: "Docs/test-only changes detected; runtime checks can be skipped."}, nil
}

func runGit(ctx context.Context, repoRoot string, args ...string) (string, error) {
	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    args,
		Dir:     repoRoot,
		Timeout: 10 * time.Second,
	})
	if res.Err != nil {
		return "", res.Err
	}
	return res.Stdout, nil
}

func parseNullDelimited(input string) []string {
	if input == "" {
		return nil
	}
	parts := bytes.Split([]byte(input), []byte{0})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		result = append(result, normalizePath(string(part)))
	}
	return result
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	trimmed = filepath.Clean(trimmed)
	if trimmed == "." {
		return ""
	}
	return trimmed
}

func isRuntimeImpacting(path string) bool {
	normalized := normalizePath(path)
	switch {
	case normalized == "AGENTS.md":
		return false
	case strings.HasPrefix(normalized, "docs/"):
		return false
	case strings.HasPrefix(normalized, ".ralph/"):
		return false
	case strings.HasPrefix(normalized, "demo/logs/"):
		return false
	case strings.HasPrefix(normalized, "demo/backups/"):
		return false
	case strings.HasPrefix(normalized, "demo/scripts/tests/"):
		return false
	case strings.HasPrefix(normalized, "scripts/tests/"):
		return false
	case strings.HasPrefix(normalized, "local/scripts/tests/"):
		return false
	case strings.HasSuffix(normalized, ".md"):
		return false
	default:
		return true
	}
}

// ValidateDecisionArgs performs CLI-level argument validation.
func ValidateDecisionArgs(paths []string) error {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("--path requires a non-empty argument")
		}
	}
	return nil
}
