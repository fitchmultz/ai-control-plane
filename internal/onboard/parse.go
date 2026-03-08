// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Parse onboarding CLI arguments into typed workflow options without mixing
//	argument handling into execution logic.
//
// Responsibilities:
//   - Parse user-supplied flags into Options.
//   - Apply CLI-level defaults for alias, host, port, and budget.
//   - Preserve help routing semantics used by the CLI command wrapper.
//
// Scope:
//   - Argument parsing only.
//
// Usage:
//   - Called by `cmd/acpctl/cmd_onboard.go`.
//
// Invariants/Assumptions:
//   - Unknown flags fail fast with usage errors.
//   - Tool-specific defaults are finalized later by resolveDefaults.
package onboard

import (
	"errors"
	"fmt"
	"io"
)

func ParseArgs(args []string, repoRoot string, stdout io.Writer, stderr io.Writer) (Options, error) {
	opts := Options{
		RepoRoot: repoRoot,
		Stdout:   stdout,
		Stderr:   stderr,
	}
	if len(args) == 0 {
		return opts, nil
	}
	opts.Tool = args[0]
	if isHelpToken(opts.Tool) {
		return opts, nil
	}
	args = args[1:]

	opts.Alias = opts.Tool + "-cli"
	opts.Budget = DefaultBudget
	opts.Host = DefaultHost
	opts.Port = DefaultPort

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--mode":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --mode")
			}
			opts.Mode = args[index]
		case "--alias":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --alias")
			}
			opts.Alias = args[index]
		case "--budget":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --budget")
			}
			opts.Budget = args[index]
		case "--model":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --model")
			}
			opts.Model = args[index]
		case "--host":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --host")
			}
			opts.Host = args[index]
		case "--port":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --port")
			}
			opts.Port = args[index]
		case "--tls":
			opts.UseTLS = true
		case "--verify":
			opts.Verify = true
		case "--write-config":
			opts.WriteConfig = true
		case "--show-key":
			opts.ShowKey = true
		case "--help", "-h":
			opts.Mode = "help"
		default:
			return Options{}, fmt.Errorf("unknown option: %s", args[index])
		}
	}

	return opts, nil
}
