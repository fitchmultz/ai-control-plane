// cmd_bridge.go - Bridge script execution helpers.
//
// Purpose:
//
//	Execute bridge-script-backed subcommands declared in the canonical command catalog.
//
// Responsibilities:
//   - Validate bridge script paths before execution.
//   - Run bridge scripts with repository-root context and stable exit codes.
//
// Scope:
//   - Low-level script invocation only; selection and help live in grouped dispatch.
//
// Usage:
//   - Called by command_spec.go for script-backed command leaves.
//
// Invariants/Assumptions:
//   - Bridge scripts are executable files rooted under the repository.
//   - Bridge execution remains timeout-bounded.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const bridgeScriptTimeout = 10 * time.Minute

func runBridgeScript(
	ctx context.Context,
	relativePath string,
	commandName string,
	scriptPrefixArgs []string,
	scriptArgs []string,
	stdout *os.File,
	stderr *os.File,
) int {
	repoRoot := detectRepoRootWithContext(ctx)
	if repoRoot == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}

	scriptPath := filepath.Join(repoRoot, relativePath)
	if err := proc.ValidateExecutable(scriptPath); err != nil {
		switch {
		case errors.Is(err, proc.ErrExecutableNotFound):
			fmt.Fprintf(stderr, "Error: bridge script not found: %s\n", scriptPath)
			return exitcodes.ACPExitPrereq
		case errors.Is(err, proc.ErrExecutableIsDirectory):
			fmt.Fprintf(stderr, "Error: bridge script path is a directory: %s\n", scriptPath)
			return exitcodes.ACPExitPrereq
		case errors.Is(err, proc.ErrExecutableNotExecutable):
			fmt.Fprintf(stderr, "Error: bridge script is not executable: %s\n", scriptPath)
			return exitcodes.ACPExitPrereq
		default:
			fmt.Fprintf(stderr, "Error: failed to validate bridge script %s: %v\n", scriptPath, err)
		}
		return exitcodes.ACPExitRuntime
	}

	res := proc.Run(ctx, proc.Request{
		Name:    "/bin/bash",
		Args:    append(append([]string{scriptPath}, scriptPrefixArgs...), scriptArgs...),
		Dir:     repoRoot,
		Env:     []string{"ACP_REPO_ROOT=" + repoRoot},
		Stdin:   os.Stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Timeout: bridgeScriptTimeout,
	})
	if res.Err != nil {
		message, code := classifyProcFailure(res.Err, procFailureMessages{
			NotFound: "bash executable not found",
			Timeout:  fmt.Sprintf("bridge script %q timed out", commandName),
			Canceled: fmt.Sprintf("bridge script %q canceled", commandName),
		})
		fmt.Fprintf(stderr, "Error: %s\n", message)
		return code
	}

	return exitcodes.ACPExitSuccess
}
