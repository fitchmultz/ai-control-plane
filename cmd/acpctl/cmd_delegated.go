// cmd_delegated.go - Delegated command execution helpers.
//
// Purpose:
//
//	Execute Make-backed and bridge-backed command leaves selected by the typed
//	command-spec layer.
//
// Responsibilities:
//   - Run Make targets with repo-root context and bounded timeouts.
//   - Validate executable overrides before launching delegated processes.
//
// Scope:
//   - Low-level delegated process execution only.
//
// Usage:
//   - Invoked by command_spec.go after command parsing and help routing finish.
//
// Invariants/Assumptions:
//   - Typed command specs already resolved the concrete delegated command path.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const makeTargetTimeout = 30 * time.Minute

func runMakeTarget(ctx context.Context, target string, makeArgs []string, stdout *os.File, stderr *os.File) int {
	makeBin := config.NewLoader().Tooling().MakeBinary
	if err := proc.ValidateExecutable(makeBin); err != nil {
		fmt.Fprintf(stderr, "Error: make executable not found or not executable: %s\n", makeBin)
		return exitcodes.ACPExitPrereq
	}

	repoRoot := detectRepoRootWithContext(ctx)
	res := proc.Run(ctx, proc.Request{
		Name:    makeBin,
		Args:    append([]string{target}, makeArgs...),
		Dir:     repoRoot,
		Stdin:   os.Stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Timeout: makeTargetTimeout,
	})
	if res.Err != nil {
		message, code := classifyProcFailure(res.Err, procFailureMessages{
			NotFound: fmt.Sprintf("make executable not found: %s", makeBin),
			Timeout:  fmt.Sprintf("make target %q timed out", target),
			Canceled: fmt.Sprintf("make target %q canceled", target),
		})
		fmt.Fprintf(stderr, "Error: %s\n", message)
		return code
	}

	return exitcodes.ACPExitSuccess
}
