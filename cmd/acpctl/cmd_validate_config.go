// cmd_validate_config.go - Deployment configuration validation commands.
//
// Purpose:
//   - Own the typed deployment configuration validation command surface.
//
// Responsibilities:
//   - Define `acpctl validate config`.
//   - Adapt parsed CLI options into typed validation options.
//   - Render consistent configuration validation output.
//
// Scope:
//   - Deployment configuration validation adapters only.
//
// Usage:
//   - Invoked through `acpctl validate config`.
//
// Invariants/Assumptions:
//   - Deployment config validation ownership remains in `internal/validation`.
package main

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

type validateConfigOptions struct {
	Production     bool
	SecretsEnvFile string
}

func validateConfigCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "config",
		Summary:     "Validate deployment configuration (use --production for host contract checks)",
		Description: "Validate deployment configuration, including production host contract checks when --production is set.",
		Options: []commandOptionSpec{
			{Name: "production", Summary: "Enforce the production deployment contract", Type: optionValueBool},
			{Name: "secrets-env-file", ValueName: "PATH", Summary: "Canonical production secrets file", Type: optionValueString},
		},
		Bind: bindParsedValue(func(input parsedCommandInput) validateConfigOptions {
			return validateConfigOptions{
				Production:     input.Bool("production"),
				SecretsEnvFile: input.NormalizedString("secrets-env-file"),
			}
		}),
		Run: runValidateConfigTyped,
	})
}

func runValidateConfigTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateConfigOptions)
	options := validation.ConfigValidationOptions{}
	if opts.Production {
		options.Profile = validation.ConfigValidationProfileProduction
	}
	options.SecretsEnvFile = opts.SecretsEnvFile

	return runIssueValidationWithPrelude(runCtx, nil, func(out *output.Output, runCtx commandRunContext) {
		printCommandSection(runCtx.Stdout, out, "=== Deployment Configuration Validation ===")
		if options.Profile == validation.ConfigValidationProfileProduction {
			fmt.Fprintf(runCtx.Stdout, "Profile: %s\n", options.Profile)
			if !textutil.IsBlank(options.SecretsEnvFile) {
				fmt.Fprintf(runCtx.Stdout, "Secrets file: %s\n", options.SecretsEnvFile)
			}
		}
	}, issueValidationConfig{
		SuccessMessage:  "Configuration validation passed",
		FailureMessage:  "Configuration validation failed",
		RuntimeErrorMsg: "Configuration validation failed",
		ColorSuccess:    true,
	}, func() ([]string, error) {
		return validation.ValidateDeploymentConfig(runCtx.RepoRoot, options)
	})
}
