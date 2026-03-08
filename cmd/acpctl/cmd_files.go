// cmd_files.go - Files subcommand implementation
//
// Purpose: Implement file synchronization commands
// Responsibilities:
//   - Parse files subcommand flags
//   - Execute Helm file synchronization
//
// Non-scope:
//   - Does not perform Git operations
//   - Does not delete extraneous files
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/filesync"
)

func runFilesSubcommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printFilesHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printFilesHelp(stdout)
		return exitcodes.ACPExitSuccess
	case "sync-helm":
		return runFilesSyncHelm(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: Unknown files subcommand: %s\n", args[0])
		printFilesHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func printFilesHelp(out *os.File) {
	command, err := lookupNativeRootCommand("files")
	if err != nil {
		fmt.Fprintf(out, "Error: %v\n", err)
		return
	}

	fmt.Fprint(out, `Usage: acpctl files <subcommand> [options]

Typed file synchronization workflows.

Subcommands:
`)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(out, "  %-12s %s\n", subcommand.Name, subcommand.Description)
	}
	fmt.Fprint(out, `

Examples:
  acpctl files sync-helm
  acpctl files sync-helm --help

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printFilesSyncHelmHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl files sync-helm

Synchronize canonical repository files into deploy/helm/ai-control-plane/files/.

Examples:
  acpctl files sync-helm

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runFilesSyncHelm(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 1 && isHelpToken(args[0]) {
		printFilesSyncHelmHelp(stdout)
		return exitcodes.ACPExitSuccess
	}
	if len(args) > 0 {
		fmt.Fprintf(stderr, "Error: Unknown argument(s): %s\n", strings.Join(args, " "))
		fmt.Fprintln(stderr, "Run `acpctl files sync-helm --help` for usage information")
		return exitcodes.ACPExitUsage
	}

	repoRoot := detectRepoRootWithContext(ctx)
	if strings.TrimSpace(repoRoot) == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}

	if err := filesync.SyncHelmFiles(filesync.SyncOptions{
		RepoRoot: repoRoot,
		Writer:   stdout,
	}); err != nil {
		fmt.Fprintf(stderr, "Error: helm file synchronization failed: %v\n", err)
		return exitcodes.ACPExitDomain
	}

	return exitcodes.ACPExitSuccess
}
