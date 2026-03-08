// cmd_env.go - Strict environment-file access commands.
//
// Purpose:
//   - Provide typed, non-executing access to repository .env values.
//
// Responsibilities:
//   - Parse env subcommands and flags.
//   - Read specific keys from env files via the shared strict parser.
//   - Emit deterministic exit codes for missing keys and invalid files.
//
// Non-scope:
//   - Does not export shell snippets or evaluate shell syntax.
//   - Does not mutate env files.
//
// Invariants/Assumptions:
//   - `.env` content is treated as data only.
//   - Command output is the raw requested value on stdout.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func runEnvCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printEnvHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printEnvHelp(stdout)
		return exitcodes.ACPExitSuccess
	case "get":
		return runEnvGetCommand(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: Unknown env subcommand: %s\n", args[0])
		printEnvHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func printEnvHelp(out *os.File) {
	command, err := lookupNativeRootCommand("env")
	if err != nil {
		fmt.Fprintf(out, "Error: %v\n", err)
		return
	}

	fmt.Fprint(out, `Usage: acpctl env <subcommand> [options]

Typed .env access helpers that never execute shell code.

Subcommands:
`)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(out, "  %-12s %s\n", subcommand.Name, subcommand.Description)
	}
	fmt.Fprint(out, `

Examples:
  acpctl env get LITELLM_MASTER_KEY
  acpctl env get --file demo/.env DATABASE_URL

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printEnvGetHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl env get [--file path] KEY

Read a single KEY from an env file as data only.

Examples:
  acpctl env get LITELLM_MASTER_KEY
  acpctl env get --file demo/.env DATABASE_URL

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runEnvGetCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 1 && isHelpToken(args[0]) {
		printEnvGetHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	flags := flag.NewFlagSet("env get", flag.ContinueOnError)
	flags.SetOutput(stderr)

	defaultEnvPath := filepath.Join(detectRepoRootWithContext(ctx), "demo", ".env")
	envPath := flags.String("file", defaultEnvPath, "Path to env file")
	if err := flags.Parse(args); err != nil {
		return exitcodes.ACPExitUsage
	}

	remaining := flags.Args()
	if len(remaining) != 1 {
		fmt.Fprintln(stderr, "Error: env get requires exactly one KEY argument")
		printEnvGetHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	key := strings.TrimSpace(remaining[0])
	if key == "" {
		fmt.Fprintln(stderr, "Error: env key must not be empty")
		return exitcodes.ACPExitUsage
	}

	value, ok, err := config.LookupEnvFile(*envPath, key)
	if err != nil {
		fmt.Fprintf(stderr, "Error: failed to read env file: %v\n", err)
		return exitcodes.ACPExitPrereq
	}
	if !ok {
		fmt.Fprintf(stderr, "Error: %s not found in %s\n", key, *envPath)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, value)
	return exitcodes.ACPExitSuccess
}
