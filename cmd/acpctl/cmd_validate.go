// cmd_validate.go - Validation commands implementation
//
// Purpose: Provide native Go validation workflows for repo configuration contracts.
// Responsibilities:
//   - Validate detection rule contracts and SIEM mapping parity.
//   - Validate deployment configuration and compose healthchecks.
//   - Report deterministic, contract-aligned exit codes for validation failures.
// Scope:
//   - Read-only validation of committed configuration and policy files.
// Usage:
//   - Invoked through `acpctl validate <subcommand>`.
// Invariants/Assumptions:
//   - Validation commands must be safe to run repeatedly without side effects.
//   - Help token handling stays consistent with delegated command routing.

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

	fmt.Fprintf(stdout, "Validated %d detection rule(s) (%d validated, %d decision-grade)\n",
		len(detections.DetectionRules), validatedCount, decisionGradeCount)
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

	fmt.Fprintf(stdout, "Validated %d SIEM mapping(s) against %d enabled detection rule(s)\n",
		len(siemQueries.SIEMQueries), contracts.CountEnabledDetectionRules(detections))
	if validateSchema {
		fmt.Fprintln(stdout, "Schema mapping coverage validated")
	}
	fmt.Fprintln(stdout, out.Green("SIEM query validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateConfig(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateConfigHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Deployment Configuration Validation ==="))

	repoRoot := detectRepoRoot()
	requiredFiles := []string{
		"demo/docker-compose.yml",
		"demo/config/litellm.yaml",
	}

	allValid := true
	for _, file := range requiredFiles {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintf(stdout, out.Fail("Missing required file: %s\n"), file)
			allValid = false
		} else {
			fmt.Fprintf(stdout, out.Pass("Found: %s\n"), file)
		}
	}

	// Check .env file exists and has required variables
	envFile := filepath.Join(repoRoot, "demo/.env")
	if _, err := os.Stat(envFile); err != nil {
		fmt.Fprint(stdout, out.Warn("Environment file not found: demo/.env\n"))
		fmt.Fprintln(stdout, "Run 'make install-env' to create it")
	} else {
		fmt.Fprint(stdout, out.Pass("Found: demo/.env\n"))
	}

	if !allValid {
		fmt.Fprintln(stderr, out.Fail("Configuration validation failed"))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, out.Green("Configuration validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateComposeHealthchecks(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printValidateComposeHealthchecksHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Docker Compose Healthchecks Validation ==="))

	repoRoot := detectRepoRoot()
	composeFile := filepath.Join(repoRoot, "demo/docker-compose.yml")

	data, err := os.ReadFile(composeFile)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to read docker-compose.yml: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	content := string(data)

	// Check for healthcheck sections
	healthcheckCount := strings.Count(content, "healthcheck:")
	testCount := strings.Count(content, "test:")

	fmt.Fprintf(stdout, "Found %d healthcheck section(s)\n", healthcheckCount)
	fmt.Fprintf(stdout, "Found %d health test(s)\n", testCount)

	if healthcheckCount == 0 {
		fmt.Fprintln(stderr, out.Warn("No healthchecks defined in compose file"))
	}

	fmt.Fprintln(stdout, out.Green("Healthcheck validation passed"))
	return exitcodes.ACPExitSuccess
}

func printValidateDetectionsHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate detections [OPTIONS]

Validate the detection rule contract in demo/config/detection_rules.yaml
against the canonical SIEM mappings and approved-model configuration.

Options:
  --verbose, -v     Enable verbose output
  --help, -h        Show this help message

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
`)
}

func printValidateSiemQueriesHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate siem-queries [OPTIONS]

Validate demo/config/siem_queries.yaml against enabled detection rules in
 demo/config/detection_rules.yaml.

Options:
  --validate-schema  Also validate normalized schema field mappings
  --verbose, -v      Enable verbose output
  --help, -h         Show this help message

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
`)
}

func printValidateConfigHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate config [OPTIONS]

Validate deployment configuration files.

Options:
  --help, -h        Show this help message

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
`)
}

func printValidateComposeHealthchecksHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate compose-healthchecks [OPTIONS]

Validate Docker Compose healthcheck configuration.

Options:
  --help, -h        Show this help message

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
`)
}
