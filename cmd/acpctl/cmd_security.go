// cmd_security.go - Security validation command adapters.
//
// Purpose:
//   - Expose typed security validators through spec-owned bind/run flows.
//
// Responsibilities:
//   - Define validate subcommand specs for security validators.
//   - Route security validation requests into `internal/security`.
//   - Render deterministic findings for operators and CI.
//
// Scope:
//   - Command-layer adapters only; policy logic lives in internal packages.
//
// Usage:
//   - Invoked via `acpctl validate <public-hygiene|license|supply-chain|secrets-audit>`.
//
// Invariants/Assumptions:
//   - Security logic must not live in the CLI package.
//   - Findings remain sorted and machine-scannable.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/security"
)

type securityValidatorCommandConfig struct {
	Name        string
	Summary     string
	Description string
	Run         func(context.Context, commandRunContext, any) int
}

func newSecurityValidatorCommandSpec(config securityValidatorCommandConfig) *commandSpec {
	return &commandSpec{
		Name:        config.Name,
		Summary:     config.Summary,
		Description: config.Description,
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindNoOptions,
			NativeRun:  config.Run,
		},
	}
}

func validateSecretsAuditCommandSpec() *commandSpec {
	return newSecurityValidatorCommandSpec(securityValidatorCommandConfig{
		Name:        "secrets-audit",
		Summary:     "Run deterministic tracked-file secrets audit",
		Description: "Run deterministic tracked-file secrets audit.",
		Run:         runSecretsAuditTyped,
	})
}

func validatePublicHygieneCommandSpec() *commandSpec {
	return newSecurityValidatorCommandSpec(securityValidatorCommandConfig{
		Name:        "public-hygiene",
		Summary:     "Fail when local-only files are tracked by git",
		Description: "Fail when local-only files are tracked by git.",
		Run:         runValidatePublicHygieneTyped,
	})
}

func validateLicenseCommandSpec() *commandSpec {
	return newSecurityValidatorCommandSpec(securityValidatorCommandConfig{
		Name:        "license",
		Summary:     "Validate license policy structure and restricted references",
		Description: "Validate the third-party license policy contract and restricted reference boundary.",
		Run:         runValidateLicenseTyped,
	})
}

func validateSupplyChainCommandSpec() *commandSpec {
	return newSecurityValidatorCommandSpec(securityValidatorCommandConfig{
		Name:        "supply-chain",
		Summary:     "Run supply-chain policy and digest validation",
		Description: "Validate the typed supply-chain policy contract and digest pinning across canonical deployment surfaces.",
		Run:         runValidateSupplyChainTyped,
	})
}

func runSecretsAuditTyped(ctx context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	logger := workflowLogger(runCtx, "validate_secrets_audit")
	workflowStart(logger)
	trackedFiles, err := security.ListTrackedFiles(ctx, runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Secrets audit could not enumerate tracked files: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}
	findings, err := security.AuditTrackedSecrets(runCtx.RepoRoot, trackedFiles)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Secrets audit failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Secrets Audit ==="))
	fmt.Fprintln(runCtx.Stdout, "Scanning tracked files for likely public-repo secret leaks...")
	if len(findings) == 0 {
		workflowComplete(logger, "tracked_files", len(trackedFiles), "findings", 0)
		fmt.Fprintln(runCtx.Stdout, out.Green("Secrets audit passed"))
		return exitcodes.ACPExitSuccess
	}
	workflowWarn(logger, "tracked_files", len(trackedFiles), "findings", len(findings))
	for _, finding := range findings {
		if finding.Line > 0 {
			fmt.Fprintf(runCtx.Stdout, "%s:%d [%s] %s\n", finding.Path, finding.Line, finding.RuleID, finding.Message)
			continue
		}
		fmt.Fprintf(runCtx.Stdout, "%s [%s] %s\n", finding.Path, finding.RuleID, finding.Message)
	}
	fmt.Fprintln(runCtx.Stderr, out.Fail("Secrets audit found tracked-file security issues"))
	return exitcodes.ACPExitDomain
}

func runValidatePublicHygieneTyped(ctx context.Context, runCtx commandRunContext, _ any) int {
	logger := workflowLogger(runCtx, "validate_public_hygiene")
	workflowStart(logger)
	trackedFiles, err := security.ListTrackedFiles(ctx, runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitPrereq
	}
	violations := security.ValidatePublicHygiene(trackedFiles)
	if len(violations) == 0 {
		workflowComplete(logger, "tracked_files", len(trackedFiles), "violations", 0)
		fmt.Fprintln(runCtx.Stdout, "Public-release tracked file hygiene passed")
		return exitcodes.ACPExitSuccess
	}
	workflowWarn(logger, "tracked_files", len(trackedFiles), "violations", len(violations))
	fmt.Fprintln(runCtx.Stderr, "Local-only files are tracked and block public release:")
	for _, violation := range violations {
		fmt.Fprintln(runCtx.Stderr, violation)
	}
	fmt.Fprintln(runCtx.Stderr, "Remove from git index (git rm --cached ...) and keep in .gitignore.")
	return exitcodes.ACPExitDomain
}

func runValidateLicenseTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	logger := workflowLogger(runCtx, "validate_license_policy")
	workflowStart(logger)
	findings, err := security.ValidateLicensePolicy(runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(findings) == 0 {
		workflowComplete(logger, "findings", 0)
		fmt.Fprintln(runCtx.Stdout, "License boundary check passed")
		return exitcodes.ACPExitSuccess
	}
	workflowWarn(logger, "findings", len(findings))
	fmt.Fprintln(runCtx.Stderr, "Restricted LiteLLM enterprise references detected outside docs:")
	for _, finding := range findings {
		fmt.Fprintln(runCtx.Stderr, finding)
	}
	return exitcodes.ACPExitDomain
}

func runValidateSupplyChainTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	logger := workflowLogger(runCtx, "validate_supply_chain")
	workflowStart(logger)
	findings, err := security.ValidateSupplyChainPolicy(runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(findings) == 0 {
		workflowComplete(logger, "findings", 0)
		fmt.Fprintln(runCtx.Stdout, "Supply-chain policy and digest pinning baseline passed")
		return exitcodes.ACPExitSuccess
	}
	workflowWarn(logger, "findings", len(findings))
	fmt.Fprintln(runCtx.Stderr, "Supply-chain policy violations detected:")
	for _, finding := range findings {
		fmt.Fprintln(runCtx.Stderr, finding)
	}
	return exitcodes.ACPExitDomain
}

func runSecretsAudit(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "secrets-audit"}, args, stdout, stderr)
}

func runValidatePublicHygiene(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "public-hygiene"}, args, stdout, stderr)
}

func runValidateLicense(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "license"}, args, stdout, stderr)
}

func runValidateSupplyChain(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "supply-chain"}, args, stdout, stderr)
}
