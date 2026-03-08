// cmd_security.go - Security validation command adapters.
//
// Purpose:
//   - Expose typed security validators through thin CLI wrappers.
//
// Responsibilities:
//   - Route security validation requests into `internal/security`.
//   - Keep CLI help and exit-code behavior stable.
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

func runSecretsAudit(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printSecretsAuditHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printSecretsAuditHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}

	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	trackedFiles, err := security.ListTrackedFiles(ctx, repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Secrets audit could not enumerate tracked files: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}
	findings, err := security.AuditTrackedSecrets(repoRoot, trackedFiles)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Secrets audit failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	fmt.Fprintln(stdout, out.Bold("=== Secrets Audit ==="))
	fmt.Fprintln(stdout, "Scanning tracked files for likely public-repo secret leaks...")
	if len(findings) == 0 {
		fmt.Fprintln(stdout, out.Green("Secrets audit passed"))
		return exitcodes.ACPExitSuccess
	}
	for _, finding := range findings {
		if finding.Line > 0 {
			fmt.Fprintf(stdout, "%s:%d [%s] %s\n", finding.Path, finding.Line, finding.RuleID, finding.Message)
			continue
		}
		fmt.Fprintf(stdout, "%s [%s] %s\n", finding.Path, finding.RuleID, finding.Message)
	}
	fmt.Fprintln(stderr, out.Fail("Secrets audit found tracked-file security issues"))
	return exitcodes.ACPExitDomain
}

func runValidatePublicHygiene(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printPublicHygieneHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printPublicHygieneHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}
	trackedFiles, err := security.ListTrackedFiles(ctx, detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitPrereq
	}
	violations := security.ValidatePublicHygiene(trackedFiles)
	if len(violations) == 0 {
		fmt.Fprintln(stdout, "Public-release tracked file hygiene passed")
		return exitcodes.ACPExitSuccess
	}
	fmt.Fprintln(stderr, "Local-only files are tracked and block public release:")
	for _, violation := range violations {
		fmt.Fprintln(stderr, violation)
	}
	fmt.Fprintln(stderr, "Remove from git index (git rm --cached ...) and keep in .gitignore.")
	return exitcodes.ACPExitDomain
}

func runValidateLicense(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printLicenseHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printLicenseHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}
	findings, err := security.ValidateLicensePolicy(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "License boundary check passed")
		return exitcodes.ACPExitSuccess
	}
	fmt.Fprintln(stderr, "Restricted LiteLLM enterprise references detected outside docs:")
	for _, finding := range findings {
		fmt.Fprintln(stderr, finding)
	}
	return exitcodes.ACPExitDomain
}

func runValidateSupplyChain(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printSupplyChainHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printSupplyChainHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}
	findings, err := security.ValidateSupplyChainPolicy(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "Supply-chain policy and digest pinning baseline passed")
		return exitcodes.ACPExitSuccess
	}
	fmt.Fprintln(stderr, "Supply-chain policy violations detected:")
	for _, finding := range findings {
		fmt.Fprintln(stderr, finding)
	}
	return exitcodes.ACPExitDomain
}

func printPublicHygieneHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate public-hygiene\n\nFail when local-only files are tracked by git.\n")
}

func printLicenseHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate license\n\nValidate the third-party license policy contract and restricted reference boundary.\n")
}

func printSupplyChainHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate supply-chain\n\nValidate the typed supply-chain policy contract and digest pinning across canonical deployment surfaces.\n")
}

func printSecretsAuditHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate secrets-audit\n\nRun deterministic tracked-file secrets audit.\n")
}
