// cmd_onboard.go - Onboarding command implementation.
//
// Purpose:
//
//	Expose typed onboarding workflows for local tools and IDE integrations.
//
// Responsibilities:
//   - Parse onboarding arguments.
//   - Render onboarding help and tool notes.
//   - Delegate execution to internal/onboard.
//
// Scope:
//   - CLI integration only; workflow logic lives in internal/onboard.
//
// Usage:
//   - Invoked through `acpctl onboard`.
//
// Invariants/Assumptions:
//   - Exit codes follow the ACP contract.
//   - The command reads demo/.env as data, never as sourced shell.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/onboard"
)

func runOnboardCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	opts, err := onboard.ParseArgs(args, detectRepoRootWithContext(ctx), stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return int(onboard.ExitUsage)
	}
	result := onboard.Run(ctx, opts)
	return int(result.ExitCode)
}
