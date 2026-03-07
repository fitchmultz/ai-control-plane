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
	"context"
	"os"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const repoRootDetectTimeout = 5 * time.Second

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
	return detectRepoRootWithContext(context.Background())
}

func detectRepoRootWithContext(ctx context.Context) string {
	if explicit := strings.TrimSpace(os.Getenv("ACP_REPO_ROOT")); explicit != "" {
		return explicit
	}
	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    []string{"rev-parse", "--show-toplevel"},
		Timeout: repoRootDetectTimeout,
	})
	if res.Err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return ""
		}
		return wd
	}
	return strings.TrimSpace(res.Stdout)
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}
