// acpctl is the typed CLI core for AI Control Plane operational workflows.
//
// Purpose:
//
//	Provide a statically-typed command implementation for shell-facing
//	operational scripts and operator commands.
//
// Responsibilities:
//   - Parse command-line arguments and flags.
//   - Execute deterministic CI runtime-scope decisions.
//   - Execute typed chargeback, status, and onboarding workflows.
//   - Delegate operator workflows to stable Make targets.
//   - Emit stable exit codes aligned with the repository contract.
//
// Non-scope:
//   - Does not execute runtime checks directly.
//   - Does not replace Docker/Make orchestration.
//
// Invariants/Assumptions:
//   - Exit codes remain: 0/1/2/3/64.
//   - `ci should-run-runtime` behavior is stable and deterministic.
//   - Operator commands run Make from the repository root.
//
// Architecture:
//
//	This file is a thin orchestrator. Command implementations are in:
//	  - cmd_ci.go       : CI subcommands
//	  - cmd_bridge.go   : Bridge to legacy scripts
//	  - cmd_status.go   : System status collection
//	  - cmd_doctor.go   : Environment diagnostics
//	  - cmd_completion.go: Shell completions
//	  - cmd_chargeback.go: Chargeback rendering/payload helpers
//	  - cmd_delegated.go: Make-delegated commands
//	  - common.go       : Shared utilities
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if err := commandStartupError(); err != nil {
		fmt.Fprintf(stderr, "Error: invalid command spec: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	invocation, err := parseInvocation(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		rootSpec, specErr := loadCommandSpec()
		if specErr == nil {
			printCommandHelp(stderr, []*commandSpec{rootSpec.Root})
		}
		return exitcodes.ACPExitUsage
	}
	if len(args) == 0 {
		printCommandHelp(stdout, invocation.Path)
		return exitcodes.ACPExitUsage
	}
	return executeInvocation(ctx, invocation, stdout, stderr)
}
