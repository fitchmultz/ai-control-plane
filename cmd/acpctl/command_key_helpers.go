// command_key_helpers.go - Shared key command rendering helpers.
//
// Purpose:
//   - Keep key-management command output normalized across dry-run, execution,
//     and revoke paths.
//
// Responsibilities:
//   - Render canonical key generation summaries.
//   - Render key inventory, inspection, rotation, and revoke output consistently.
//   - Keep lifecycle messaging out of individual command handlers.
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
	"text/tabwriter"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
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

func printKeyList(out io.Writer, keys []gateway.KeyInfo) {
	if len(keys) == 0 {
		fmt.Fprintln(out, "No virtual keys found.")
		return
	}

	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ALIAS\tBUDGET\tSPEND\tRPM\tTPM\tPARALLEL")
	for _, key := range keys {
		fmt.Fprintf(writer, "%s\t$%.2f\t$%.2f\t%d\t%d\t%d\n",
			key.Alias(),
			key.MaxBudget,
			key.Spend,
			key.RPMLimit,
			key.TPMLimit,
			key.MaxParallelRequests,
		)
	}
	_ = writer.Flush()
}

func printKeyInspection(out io.Writer, inspection keygen.Inspection) {
	key := inspection.Key
	usage := inspection.Usage

	fmt.Fprintf(out, "Alias: %s\n", key.Alias())
	fmt.Fprintf(out, "Budget: $%.2f\n", key.MaxBudget)
	fmt.Fprintf(out, "Budget Duration: %s\n", key.BudgetDuration)
	fmt.Fprintf(out, "Report Month: %s\n", usage.ReportMonth)
	fmt.Fprintf(out, "Usage Spend: $%.2f\n", usage.TotalSpend)
	fmt.Fprintf(out, "Usage Requests: %d\n", usage.TotalRequests)
	fmt.Fprintf(out, "Usage Tokens: %d\n", usage.TotalTokens)
	if usage.LastSeen != "" {
		fmt.Fprintf(out, "Last Seen: %s\n", usage.LastSeen)
	}
	if len(usage.ByModel) > 0 {
		fmt.Fprintln(out, "Top Models:")
		for _, row := range usage.ByModel {
			fmt.Fprintf(out, "  - %s: $%.2f (%d req / %d tok)\n", row.Model, row.SpendAmount, row.RequestCount, row.TokenCount)
		}
	}
}

func printKeyRotation(out io.Writer, result keygen.RotationResult) {
	fmt.Fprintf(out, "Original Alias: %s\n", result.Original.Key.Alias())
	fmt.Fprintf(out, "Replacement Alias: %s\n", result.ReplacementPlan.Request.KeyAlias)
	fmt.Fprintf(out, "Replacement Budget: $%.2f\n", result.ReplacementPlan.Request.MaxBudget)
	fmt.Fprintf(out, "Replacement Role: %s\n", result.ReplacementPlan.Role)

	if result.Replacement != nil {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Replacement Secret: %s\n", result.Replacement.Key)
	}
	if result.RevokedOld {
		fmt.Fprintln(out, "Old key revoked: yes")
	} else {
		fmt.Fprintln(out, "Old key revoked: no (staged cutover)")
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Cutover Plan:")
	for _, line := range result.StageInstructions {
		fmt.Fprintf(out, "  - %s\n", line)
	}
}

func printKeyRevokeProgress(out io.Writer, alias string) {
	fmt.Fprintf(out, "Revoking key: %s\n", alias)
}

func printKeyRevokeSuccess(out io.Writer, printer *output.Output, alias string) {
	printCommandSuccess(out, printer, "Key revocation complete")
	fmt.Fprintf(out, "Alias: %s\n", alias)
}
