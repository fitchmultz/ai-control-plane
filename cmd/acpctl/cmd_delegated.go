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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const makeTargetTimeout = 30 * time.Minute

func runMakeTarget(ctx context.Context, target string, makeArgs []string, stdout *os.File, stderr *os.File) int {
	makeBin := config.NewLoader().Tooling().MakeBinary
	if err := ensureExecutable(makeBin); err != nil {
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
		switch {
		case proc.IsNotFound(res.Err):
			fmt.Fprintf(stderr, "Error: make executable not found: %s\n", makeBin)
		case proc.IsTimeout(res.Err):
			fmt.Fprintf(stderr, "Error: make target %q timed out\n", target)
		case proc.IsCanceled(res.Err):
			fmt.Fprintf(stderr, "Error: make target %q canceled\n", target)
		}
		return proc.ACPExitCode(res.Err)
	}

	return exitcodes.ACPExitSuccess
}

func ensureExecutable(command string) error {
	if strings.ContainsRune(command, filepath.Separator) {
		info, err := os.Stat(command)
		if err != nil {
			return err
		}
		if info.Mode().IsDir() {
			return errors.New("command path is a directory")
		}
		if info.Mode()&0o111 == 0 {
			return errors.New("command path is not executable")
		}
		return nil
	}

	_, err := exec.LookPath(command)
	return err
}
