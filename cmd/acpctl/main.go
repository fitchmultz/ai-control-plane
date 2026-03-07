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
//   - Execute typed local filesystem workflows.
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
//	  - cmd_files.go    : File synchronization
//	  - cmd_bridge.go   : Bridge to legacy scripts
//	  - cmd_status.go   : System status collection
//	  - cmd_doctor.go   : Environment diagnostics
//	  - cmd_completion.go: Shell completions
//	  - cmd_delegated.go: Make-delegated commands
//	  - common.go       : Shared utilities
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
	if len(args) == 0 {
		printRootHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printRootHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	if command, ok := lookupNativeCommand(args[0]); ok {
		return command.Run(ctx, args[1:], stdout, stderr)
	}

	group, ok := lookupDelegatedGroup(args[0])
	if !ok {
		fmt.Fprintf(stderr, "Error: Unknown command: %s\n", args[0])
		printRootHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	return runDelegatedGroup(ctx, group, args[1:], stdout, stderr)
}

func printRootHelp(out *os.File) {
	registry := buildCommandRegistry()

	fmt.Fprint(out, `Usage: acpctl <command> [subcommand] [options or make args]

Typed control-plane CLI for AI Control Plane operations.

Commands:
`)
	for _, command := range registry.RootCommands {
		fmt.Fprintf(out, "  %-16s %s\n", command.Name, command.Description)
	}
	fmt.Fprint(out, `

Examples:
  acpctl ci should-run-runtime --quiet
  acpctl ci wait --timeout 120
  acpctl env get LITELLM_MASTER_KEY
  acpctl files sync-helm
  acpctl doctor
  acpctl benchmark baseline --requests 20 --concurrency 2
  acpctl doctor --json
  acpctl bridge host_preflight --help
  acpctl deploy up
  acpctl deploy readiness-evidence run
  acpctl validate config
  acpctl db status
  acpctl key gen alice --budget 10.00
  acpctl helm validate

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error

Environment:
  ACPCTL_MAKE_BIN   Override make executable used by delegated commands
                    (default: make)
  ACP_REPO_ROOT     Override repository root detection
`)
}
