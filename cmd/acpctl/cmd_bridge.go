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

func runBridgeScript(ctx context.Context, relativePath string, commandName string, scriptArgs []string, stdout *os.File, stderr *os.File) int {
	repoRoot := detectRepoRootWithContext(ctx)
	if repoRoot == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}

	scriptPath := filepath.Join(repoRoot, relativePath)
	info, err := os.Stat(scriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "Error: bridge script not found: %s\n", scriptPath)
			return exitcodes.ACPExitPrereq
		}
		fmt.Fprintf(stderr, "Error: failed to stat bridge script %s: %v\n", scriptPath, err)
		return exitcodes.ACPExitRuntime
	}
	if info.IsDir() {
		fmt.Fprintf(stderr, "Error: bridge script path is a directory: %s\n", scriptPath)
		return exitcodes.ACPExitPrereq
	}
	if info.Mode()&0o111 == 0 {
		fmt.Fprintf(stderr, "Error: bridge script is not executable: %s\n", scriptPath)
		return exitcodes.ACPExitPrereq
	}

	res := proc.Run(ctx, proc.Request{
		Name:    "/bin/bash",
		Args:    append([]string{scriptPath}, scriptArgs...),
		Dir:     repoRoot,
		Env:     []string{"ACP_REPO_ROOT=" + repoRoot},
		Stdin:   os.Stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Timeout: bridgeScriptTimeout,
	})
	if res.Err != nil {
		switch {
		case proc.IsNotFound(res.Err):
			fmt.Fprintln(stderr, "Error: bash executable not found")
		case proc.IsTimeout(res.Err):
			fmt.Fprintf(stderr, "Error: bridge script %q timed out\n", commandName)
		case proc.IsCanceled(res.Err):
			fmt.Fprintf(stderr, "Error: bridge script %q canceled\n", commandName)
		}
		return proc.ACPExitCode(res.Err)
	}

	return exitcodes.ACPExitSuccess
}
