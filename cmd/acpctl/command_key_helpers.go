// command_key_helpers.go - Shared key command rendering helpers.
//
// Purpose:
//   - Keep key-management command output normalized across dry-run, execution,
//     and revoke paths.
//
// Responsibilities:
//   - Render canonical key generation summaries.
//   - Render generation progress details consistently.
//   - Render placeholder revoke messaging without duplicating strings.
//
// Scope:
//   - Command-layer output helpers for `acpctl key`.
//
// Usage:
//   - Used by key command handlers in cmd/acpctl.
//
// Invariants/Assumptions:
//   - Output remains human-oriented.
//   - Helpers preserve the existing stdout/stderr split.
package main

import (
	"fmt"
	"io"

	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

func printKeyGenerationPlan(out io.Writer, printer *output.Output, title string, budget float64, plan keygen.GenerateRequestPlan) {
	fmt.Fprintln(out, printer.Bold(title))
	fmt.Fprintf(out, "Alias: %s\n", plan.Request.KeyAlias)
	fmt.Fprintf(out, "Budget: $%.2f\n", budget)
	fmt.Fprintf(out, "Duration: %s\n", plan.Request.BudgetDuration)
	fmt.Fprintf(out, "Role: %s\n", plan.Role)
	fmt.Fprintf(out, "Models: %v\n", plan.Models)
	if plan.Request.RPMLimit > 0 {
		fmt.Fprintf(out, "RPM: %d\n", plan.Request.RPMLimit)
	}
	if plan.Request.TPMLimit > 0 {
		fmt.Fprintf(out, "TPM: %d\n", plan.Request.TPMLimit)
	}
	if plan.Request.MaxParallelRequests > 0 {
		fmt.Fprintf(out, "Max Parallel: %d\n", plan.Request.MaxParallelRequests)
	}
}

func printKeyGenerationProgress(out io.Writer, plan keygen.GenerateRequestPlan) {
	fmt.Fprintf(out, "Generating key '%s' with budget: $%.2f (role: %s)\n", plan.Request.KeyAlias, plan.Request.MaxBudget, plan.Role)
	if plan.Request.RPMLimit > 0 {
		fmt.Fprintf(out, "  RPM limit: %d\n", plan.Request.RPMLimit)
	}
	if plan.Request.TPMLimit > 0 {
		fmt.Fprintf(out, "  TPM limit: %d\n", plan.Request.TPMLimit)
	}
	if plan.Request.MaxParallelRequests > 0 {
		fmt.Fprintf(out, "  Max parallel: %d\n", plan.Request.MaxParallelRequests)
	}
	fmt.Fprintf(out, "  Budget duration: %s\n", plan.Request.BudgetDuration)
	fmt.Fprintln(out)
}

func printKeyRevokeProgress(out io.Writer, alias string) {
	fmt.Fprintf(out, "Revoking key: %s\n", alias)
}

func printKeyRevokeSuccess(out io.Writer, printer *output.Output, alias string) {
	printCommandSuccess(out, printer, "Key revocation complete")
	fmt.Fprintf(out, "Alias: %s\n", alias)
}
