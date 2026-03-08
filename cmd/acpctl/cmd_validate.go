// cmd_validate.go - Validation command adapters.
//
// Purpose:
//   - Expose typed repository validation workflows through thin CLI wrappers.
//
// Responsibilities:
//   - Delegate validation logic to internal packages.
//   - Keep help text and exit-code behavior stable.
//   - Render deterministic findings for local CI and operators.
//
// Scope:
//   - Command-layer adapters only; validators live in internal packages.
//
// Usage:
//   - Invoked through `acpctl validate <subcommand>`.
//
// Invariants/Assumptions:
//   - Validators are side-effect free.
//   - The CLI package does not own validation policy.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/contracts"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

func runValidateDetections(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	verbose := false
	for _, arg := range args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		}
		if isHelpToken(arg) {
			printValidateDetectionsHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Detection Rules Validation ==="))
	repoRoot := detectRepoRoot()
	rulesPath := filepath.Join(repoRoot, detectionRulesRelativePath)
	siemPath := filepath.Join(repoRoot, siemQueriesRelativePath)
	litellmPath := filepath.Join(repoRoot, litellmConfigRelativePath)
	detections, err := contracts.LoadDetectionRulesFile(rulesPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load detection rules: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	siemQueries, err := contracts.LoadSIEMQueriesFile(siemPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load SIEM query mappings: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	litellmConfig, err := catalog.LoadLiteLLMConfig(litellmPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load LiteLLM config: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if verbose {
		fmt.Fprintf(stdout, "Detection rules: %s\n", rulesPath)
		fmt.Fprintf(stdout, "SIEM query mappings: %s\n", siemPath)
		fmt.Fprintf(stdout, "Approved models source: %s\n", litellmPath)
	}
	issues := contracts.ValidateDetectionContracts(detections, siemQueries, litellmConfig)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(stderr, "- %s\n", issue)
		}
		fmt.Fprintln(stderr, out.Fail("Detection validation failed"))
		return exitcodes.ACPExitDomain
	}
	validatedCount := 0
	decisionGradeCount := 0
	for _, rule := range detections.DetectionRules {
		if strings.EqualFold(rule.OperationalStatus, "validated") {
			validatedCount++
		}
		if strings.EqualFold(rule.CoverageTier, "decision-grade") {
			decisionGradeCount++
		}
	}
	fmt.Fprintf(stdout, "Validated %d detection rule(s) (%d validated, %d decision-grade)\n", len(detections.DetectionRules), validatedCount, decisionGradeCount)
	fmt.Fprintln(stdout, out.Green("Detection rules validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateSiemQueries(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	validateSchema := false
	verbose := false
	for _, arg := range args {
		if arg == "--validate-schema" {
			validateSchema = true
		}
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		}
		if isHelpToken(arg) {
			printValidateSiemQueriesHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== SIEM Queries Validation ==="))
	repoRoot := detectRepoRoot()
	rulesPath := filepath.Join(repoRoot, detectionRulesRelativePath)
	siemPath := filepath.Join(repoRoot, siemQueriesRelativePath)
	litellmPath := filepath.Join(repoRoot, litellmConfigRelativePath)
	detections, err := contracts.LoadDetectionRulesFile(rulesPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load detection rules: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	siemQueries, err := contracts.LoadSIEMQueriesFile(siemPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load SIEM query mappings: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	litellmConfig, err := catalog.LoadLiteLLMConfig(litellmPath)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to load LiteLLM config: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if verbose {
		fmt.Fprintf(stdout, "Detection rules: %s\n", rulesPath)
		fmt.Fprintf(stdout, "SIEM query mappings: %s\n", siemPath)
		fmt.Fprintf(stdout, "Approved models source: %s\n", litellmPath)
	}
	issues := contracts.ValidateSIEMContracts(detections, siemQueries, litellmConfig, validateSchema)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(stderr, "- %s\n", issue)
		}
		fmt.Fprintln(stderr, out.Fail("SIEM query validation failed"))
		return exitcodes.ACPExitDomain
	}
	fmt.Fprintf(stdout, "Validated %d SIEM mapping(s) against %d enabled detection rule(s)\n", len(siemQueries.SIEMQueries), contracts.CountEnabledDetectionRules(detections))
	if validateSchema {
		fmt.Fprintln(stdout, "Schema mapping coverage validated")
	}
	fmt.Fprintln(stdout, out.Green("SIEM query validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateConfig(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateConfigHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Deployment Configuration Validation ==="))
	issues, err := validation.ValidateDeploymentSurfaces(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Configuration validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, out.Green("Configuration validation passed"))
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintf(stderr, "- %s\n", issue)
	}
	fmt.Fprintln(stderr, out.Fail("Configuration validation failed"))
	return exitcodes.ACPExitDomain
}

func runValidateComposeHealthchecks(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateComposeHealthchecksHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Docker Compose Healthchecks Validation ==="))
	issues, err := validation.ValidateComposeHealthchecks(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Healthcheck validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, out.Green("Healthcheck validation passed"))
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintf(stderr, "- %s\n", issue)
	}
	fmt.Fprintln(stderr, out.Fail("Healthcheck validation failed"))
	return exitcodes.ACPExitDomain
}

func runValidateHeaders(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateHeadersHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	issues, err := validation.ValidateGoHeaders(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, "Go header policy validation passed")
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintln(stderr, issue)
	}
	return exitcodes.ACPExitDomain
}

func runValidateEnvAccess(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateEnvAccessHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}
	issues, err := validation.ValidateDirectEnvAccess(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(stdout, "Direct environment access policy passed")
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintln(stderr, issue)
	}
	return exitcodes.ACPExitDomain
}

func printValidateDetectionsHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate detections [OPTIONS]\n")
}
func printValidateSiemQueriesHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate siem-queries [OPTIONS]\n")
}
func printValidateConfigHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate config [OPTIONS]\n")
}
func printValidateComposeHealthchecksHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate compose-healthchecks [OPTIONS]\n")
}
func printValidateHeadersHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate headers [OPTIONS]\n\nValidate Go source file header policy.\n")
}
func printValidateEnvAccessHelp(out *os.File) {
	fmt.Fprint(out, "Usage: acpctl validate env-access [OPTIONS]\n\nFail on direct environment access outside internal/config.\n")
}
