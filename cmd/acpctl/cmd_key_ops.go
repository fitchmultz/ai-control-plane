// cmd_key_ops.go - Key operations command implementation
//
// Purpose: Provide native Go implementation of key generation and management
//
// Responsibilities:
//   - Parse arguments using internal/keygen/parser
//   - Validate using internal/keygen/validator
//   - Execute key generation via gateway client
//
// Non-scope:
//   - Argument parsing logic (see internal/keygen/parser.go)
//   - Validation logic (see internal/keygen/validator.go)
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

	acpconfig "github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

func runKeyGenCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	// Handle help
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		printKeyGenHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	out := output.New()

	// Parse arguments using the parser module
	config, err := keygen.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		printKeyGenHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	// Validate alias
	if err := keygen.ValidateAlias(config.Alias); err != nil {
		fmt.Fprintf(stderr, "Invalid alias: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	// Resolve and validate role
	role := keygen.ResolveRole(config.Role)
	if err := keygen.ValidateRole(role); err != nil {
		fmt.Fprintf(stderr, "Invalid role: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	// Check prerequisites
	if err := keygen.CheckPrerequisites(!config.DryRun); err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		if !config.DryRun {
			fmt.Fprintln(stderr, "Set it in your environment: export LITELLM_MASTER_KEY=...")
		}
		return exitcodes.ACPExitPrereq
	}

	// Get models for role
	models := keygen.GetModelsForRole(role)

	// Build request
	req := &gateway.GenerateKeyRequest{
		KeyAlias:       config.Alias,
		MaxBudget:      config.Budget,
		BudgetDuration: config.Duration,
		Models:         models,
	}

	if config.RPM > 0 {
		req.RPMLimit = config.RPM
	}
	if config.TPM > 0 {
		req.TPMLimit = config.TPM
	}
	if config.Parallel > 0 {
		req.MaxParallelRequests = config.Parallel
	}

	// Dry run mode
	if config.DryRun {
		return runDryRun(config, role, models, stdout, out)
	}

	// Check curl prerequisite
	if !prereq.CommandExists("curl") {
		fmt.Fprintln(stderr, out.Fail("curl is required"))
		return exitcodes.ACPExitPrereq
	}

	// Create gateway client and generate key
	return generateKey(ctx, config, req, stdout, stderr, out)
}

func runDryRun(config *keygen.Config, role string, models []string, stdout *os.File, out *output.Output) int {
	fmt.Fprintln(stdout, out.Bold("=== Key Generation (Dry Run) ==="))
	fmt.Fprintf(stdout, "Alias: %s\n", config.Alias)
	fmt.Fprintf(stdout, "Budget: $%.2f\n", config.Budget)
	fmt.Fprintf(stdout, "Duration: %s\n", config.Duration)
	fmt.Fprintf(stdout, "Role: %s\n", role)
	fmt.Fprintf(stdout, "Models: %v\n", models)
	if config.RPM > 0 {
		fmt.Fprintf(stdout, "RPM: %d\n", config.RPM)
	}
	if config.TPM > 0 {
		fmt.Fprintf(stdout, "TPM: %d\n", config.TPM)
	}
	if config.Parallel > 0 {
		fmt.Fprintf(stdout, "Max Parallel: %d\n", config.Parallel)
	}
	return exitcodes.ACPExitSuccess
}

func generateKey(ctx context.Context, config *keygen.Config, req *gateway.GenerateKeyRequest, stdout *os.File, stderr *os.File, out *output.Output) int {
	masterKey := acpconfig.NewLoader().Gateway(true).MasterKey
	client := gateway.NewClient(gateway.WithMasterKey(masterKey))

	role := keygen.ResolveRole(config.Role)
	fmt.Fprintf(stderr, "Generating key '%s' with budget: $%.2f (role: %s)\n", config.Alias, config.Budget, role)
	if config.RPM > 0 {
		fmt.Fprintf(stderr, "  RPM limit: %d\n", config.RPM)
	}
	if config.TPM > 0 {
		fmt.Fprintf(stderr, "  TPM limit: %d\n", config.TPM)
	}
	if config.Parallel > 0 {
		fmt.Fprintf(stderr, "  Max parallel: %d\n", config.Parallel)
	}
	fmt.Fprintf(stderr, "  Budget duration: %s\n", config.Duration)
	fmt.Fprintln(stderr, "")

	resp, err := client.GenerateKey(ctx, req)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Key generation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	// Output key to stdout ONLY
	fmt.Fprintln(stdout, resp.Key)
	return exitcodes.ACPExitSuccess
}

func runKeyRevokeCommand(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		printKeyRevokeHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	// Parse alias
	config, err := keygen.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		printKeyRevokeHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	if config.Alias == "" {
		fmt.Fprintln(stderr, "Error: alias is required")
		printKeyRevokeHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	out := output.New()

	// Check prerequisites (master key required)
	if err := keygen.CheckPrerequisites(true); err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	fmt.Fprintf(stdout, "Revoking key: %s\n", config.Alias)
	fmt.Fprintln(stdout, out.Yellow("Note: Key revocation requires LiteLLM admin API"))
	fmt.Fprintln(stdout, "This would call DELETE /key/{alias} if supported by LiteLLM")

	return exitcodes.ACPExitSuccess
}

func printKeyGenHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl key gen <alias> [OPTIONS]

Generate a LiteLLM virtual key with optional budget and rate limits.

Arguments:
  alias              Key alias/name (required)
                     Allowed: A-Z, a-z, 0-9, ., _, - (1-64 chars)

Options:
  --budget BUDGET    Maximum budget in USD (default: 10.00)
  --rpm RPM          Requests per minute limit (optional)
  --tpm TPM          Tokens per minute limit (optional)
  --parallel N       Max parallel requests (optional)
  --duration DUR     Budget reset duration (default: 30d)
  --role ROLE        Role for key generation (default: developer)
                     Valid: admin, team-lead, developer, auditor
  --dry-run          Preview the request without executing
  --help, -h         Show this help message

Environment Variables:
  LITELLM_MASTER_KEY  Master key for key generation (required)
  GATEWAY_HOST        Gateway host (default: 127.0.0.1)
  LITELLM_PORT        LiteLLM port (default: 4000)

Examples:
  acpctl key gen my-key                    # Generate with $10 budget
  acpctl key gen my-key --budget 5.00      # Generate with $5 budget
  acpctl key gen my-key --role developer   # Generate with specific role
  acpctl key gen my-key --dry-run          # Preview only

Exit codes:
  0   Success: key generated
  2   Prerequisites not ready
  3   Runtime error
  64  Usage error
`)
}

func printKeyRevokeHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl key revoke <alias>

Revoke a virtual key by alias.

Arguments:
  alias              Key alias to revoke (required)

Environment Variables:
  LITELLM_MASTER_KEY  Master key (required)

Exit codes:
  0   Success
  2   Prerequisites not ready
  64  Usage error
`)
}
