// cmd_ci.go - CI subcommand implementation
//
// Purpose: Implement CI-related commands for runtime scope decisions
// Responsibilities:
//   - Parse CI subcommand flags
//   - Execute should-run-runtime decision logic
//
// Non-scope:
//   - Does not execute runtime checks directly
//   - Does not modify git state

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/ci"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func runCISubcommand(args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printCIHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printCIHelp(stdout)
		return exitcodes.ACPExitSuccess
	case "should-run-runtime":
		return runCIShouldRunRuntime(args[1:], stdout, stderr)
	case "wait":
		return runCIWaitCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: Unknown ci subcommand: %s\n", args[0])
		printCIHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func printCIHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl ci <subcommand> [options]

CI subcommands:
  should-run-runtime   Decide whether runtime checks should run
  wait                 Wait for services to become healthy

Examples:
  acpctl ci should-run-runtime --quiet
  acpctl ci wait --timeout 120

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runCIShouldRunRuntime(args []string, stdout *os.File, stderr *os.File) int {
	var paths repeatedStringFlag
	var quiet bool

	fs := flag.NewFlagSet("should-run-runtime", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Var(&paths, "path", "Add a changed path explicitly (repeatable)")
	fs.BoolVar(&quiet, "quiet", false, "Print no informational output")
	fs.BoolVar(&quiet, "q", false, "Print no informational output")
	help := fs.Bool("help", false, "Show help")
	fs.BoolVar(help, "h", false, "Show help")

	fs.Usage = func() {
		fmt.Fprint(fs.Output(), `Usage: acpctl ci should-run-runtime [OPTIONS]

Determine whether CI runtime checks should run for the current change set.

Options:
  --path PATH   Add a changed path explicitly (repeatable)
  --quiet, -q   Print no informational output
  --help, -h    Show this help message

Exit codes:
  0   Runtime checks should run
  1   Runtime checks can be skipped
  2   Prerequisites not ready
  64  Usage error
`)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return exitcodes.ACPExitSuccess
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	if *help {
		fs.Usage()
		return exitcodes.ACPExitSuccess
	}

	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "Error: Unknown positional argument(s): %s\n", fs.Args())
		return exitcodes.ACPExitUsage
	}

	if err := ci.ValidateDecisionArgs(paths); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	repoRoot := detectRepoRoot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := ci.DecideRuntimeScope(ctx, ci.DecisionOptions{
		RepoRoot: repoRoot,
		Paths:    paths,
		CIFull:   os.Getenv("CI_FULL"),
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: runtime scope decision failed: %v\n", err)
		return exitcodes.ACPExitRuntime
	}

	if !quiet && result.Reason != "" {
		fmt.Fprintln(stdout, result.Reason)
	}

	if result.ShouldRun {
		return exitcodes.ACPExitSuccess
	}
	return exitcodes.ACPExitDomain
}
