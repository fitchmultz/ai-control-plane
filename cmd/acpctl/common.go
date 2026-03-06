// common.go - Shared utilities for acpctl commands
//
// Purpose: Provide common helper functions used across command implementations
// Responsibilities:
//   - Repository root detection
//   - Terminal capability detection
//   - String utility functions
//
// Non-scope:
//   - Does not implement command logic
//   - Does not handle I/O directly

package main

import (
	"os"
	"os/exec"
	"strings"
)

// repeatedStringFlag is a flag that can be specified multiple times
type repeatedStringFlag []string

func (r *repeatedStringFlag) String() string {
	return strings.Join(*r, ",")
}

func (r *repeatedStringFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

// isHelpToken checks if the argument is a help flag
func isHelpToken(arg string) bool {
	switch arg {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}

// detectRepoRoot finds the repository root using git or environment variable
func detectRepoRoot() string {
	if explicit := strings.TrimSpace(os.Getenv("ACP_REPO_ROOT")); explicit != "" {
		return explicit
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return ""
		}
		return wd
	}
	return strings.TrimSpace(string(out))
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
