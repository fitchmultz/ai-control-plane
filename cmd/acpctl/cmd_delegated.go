// cmd_delegated.go - Grouped command dispatch implementation.
//
// Purpose:
//
//	Route grouped CLI subcommands through the canonical command catalog.
//
// Responsibilities:
//   - Render grouped command help from catalog metadata.
//   - Dispatch grouped subcommands to native handlers, Make targets, or bridge scripts.
//   - Preserve deterministic usage and exit-code behavior across grouped commands.
//
// Scope:
//   - Command selection, help rendering, and Make process invocation.
//
// Usage:
//   - Invoked by main.go for any root command that owns subcommands.
//
// Invariants/Assumptions:
//   - Grouped subcommands resolve through command_registry.go only.
//   - Native handlers remain responsible for their own detailed help text.
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

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const makeTargetTimeout = 30 * time.Minute

func runCommandGroup(ctx context.Context, root rootCommandDefinition, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printCommandGroupHelp(stdout, root)
		return exitcodes.ACPExitUsage
	}

	if isHelpToken(args[0]) {
		printCommandGroupHelp(stdout, root)
		return exitcodes.ACPExitSuccess
	}

	subcommand, ok := lookupSubcommand(root, args[0])
	if !ok {
		fmt.Fprintf(stderr, "Error: Unknown %s subcommand: %s\n", root.Name, args[0])
		printCommandGroupHelp(stderr, root)
		return exitcodes.ACPExitUsage
	}

	if subcommand.NativeRun != nil {
		return subcommand.NativeRun(ctx, args[1:], stdout, stderr)
	}

	if len(args) > 1 && isHelpToken(args[1]) {
		printGroupedSubcommandHelp(stdout, root, subcommand)
		return exitcodes.ACPExitSuccess
	}

	if subcommand.ScriptRelativePath != "" {
		return runBridgeScript(ctx, subcommand, args[1:], stdout, stderr)
	}

	return runMakeTarget(ctx, subcommand.MakeTarget, args[1:], stdout, stderr)
}

func lookupDelegatedGroup(name string) (rootCommandDefinition, bool) {
	command, ok := lookupRootCommand(name)
	if !ok || command.NativeRun != nil {
		return rootCommandDefinition{}, false
	}
	return command, true
}

func runDelegatedGroup(ctx context.Context, root rootCommandDefinition, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroup(ctx, root, args, stdout, stderr)
}

func printCommandGroupHelp(out *os.File, root rootCommandDefinition) {
	fmt.Fprintf(out, "Usage: acpctl %s <subcommand> [options or make args]\n\n", root.Name)
	fmt.Fprintf(out, "%s.\n\n", root.Description)
	fmt.Fprintln(out, "Subcommands:")
	for _, subcommand := range root.Subcommands {
		fmt.Fprintf(out, "  %-22s %s\n", subcommand.Name, subcommand.Description)
	}
	if len(root.Examples) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		for _, example := range root.Examples {
			fmt.Fprintf(out, "  %s\n", example)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Exit codes:")
	fmt.Fprintln(out, "  0   Success")
	fmt.Fprintln(out, "  1   Domain non-success")
	fmt.Fprintln(out, "  2   Prerequisites not ready")
	fmt.Fprintln(out, "  3   Runtime/internal error")
	fmt.Fprintln(out, "  64  Usage error")
}

func printGroupedSubcommandHelp(out *os.File, root rootCommandDefinition, subcommand subcommandDefinition) {
	usage := subcommand.Usage
	if usage == "" {
		usage = fmt.Sprintf("acpctl %s %s [make args]", root.Name, subcommand.Name)
	}
	fmt.Fprintf(out, "Usage: %s\n\n", usage)
	fmt.Fprintf(out, "%s\n\n", subcommand.Description)
	switch {
	case subcommand.MakeTarget != "":
		fmt.Fprintf(out, "Delegates to make target: %s\n\n", subcommand.MakeTarget)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintf(out, "  acpctl %s %s\n", root.Name, subcommand.Name)
		fmt.Fprintf(out, "  acpctl %s %s VERBOSE=1\n", root.Name, subcommand.Name)
	case subcommand.ScriptRelativePath != "":
		fmt.Fprintf(out, "Executes bridge script: %s\n", subcommand.ScriptRelativePath)
	}
}

func runMakeTarget(ctx context.Context, target string, makeArgs []string, stdout *os.File, stderr *os.File) int {
	makeBin := strings.TrimSpace(os.Getenv("ACPCTL_MAKE_BIN"))
	if makeBin == "" {
		makeBin = "make"
	}
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
