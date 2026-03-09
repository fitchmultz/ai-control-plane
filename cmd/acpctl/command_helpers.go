// command_helpers.go - Shared command helper primitives.
//
// Purpose:
//   - Centralize repeated command failure formatting and repo-aware helpers.
//
// Responsibilities:
//   - Provide nil-safe command logger access.
//   - Provide shared failure and validation reporting helpers.
//   - Resolve repository-relative operator inputs consistently.
//
// Scope:
//   - Shared command-layer helper functions only.
//
// Usage:
//   - Used by binders and handlers across cmd/acpctl.
//
// Invariants/Assumptions:
//   - Error messages remain stderr-oriented.
//   - Relative paths resolve from the detected repository root, not process cwd.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

func ensureWorkflowLogger(runCtx commandRunContext) *slog.Logger {
	if runCtx.Logger != nil {
		return runCtx.Logger
	}
	return logging.Nop()
}

func requireCommandRepoRoot(bindCtx commandBindContext) (string, error) {
	repoRoot := strings.TrimSpace(bindCtx.RepoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("repository root could not be determined; run inside the repository or set ACP_REPO_ROOT")
	}
	return repoRoot, nil
}

func resolveRepoInput(repoRoot string, value string) string {
	return repopath.ResolveRepoPath(repoRoot, value)
}

func failCommand(stderr *os.File, out *output.Output, code int, err error, message string) int {
	switch {
	case err != nil && strings.TrimSpace(message) != "":
		fmt.Fprintf(stderr, out.Fail("%s: %v\n"), message, err)
	case err != nil:
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
	case strings.TrimSpace(message) != "":
		fmt.Fprintf(stderr, out.Fail("%s\n"), message)
	}
	return code
}

func failValidation(stderr *os.File, out *output.Output, issues []string, failureMessage string) int {
	for _, issue := range issues {
		fmt.Fprintf(stderr, "- %s\n", issue)
	}
	if strings.TrimSpace(failureMessage) != "" {
		fmt.Fprintln(stderr, out.Fail(failureMessage))
	}
	return exitcodes.ACPExitDomain
}

func runCommandPath(ctx context.Context, prefix []string, args []string, stdout *os.File, stderr *os.File) int {
	combined := append(append([]string(nil), prefix...), args...)
	invocation, err := parseInvocation(combined)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		if path, pathErr := findCommandPath(prefix); pathErr == nil {
			printCommandHelp(stderr, path)
		}
		return exitcodes.ACPExitUsage
	}
	return executeInvocation(ctx, invocation, stdout, stderr)
}

func runCommandGroupPath(ctx context.Context, prefix []string, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		if path, err := findCommandPath(prefix); err == nil {
			printCommandHelp(stdout, path)
		}
		return exitcodes.ACPExitUsage
	}
	return runCommandPath(ctx, prefix, args, stdout, stderr)
}
