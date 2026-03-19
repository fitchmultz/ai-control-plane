// cmd_validate_policy_rules.go - Custom policy rule validation command adapter.
//
// Purpose:
//   - Own the typed validation surface for the tracked ACP custom policy rules.
//
// Responsibilities:
//   - Define `acpctl validate policy-rules`.
//   - Load the tracked rule file and repository-backed RBAC/model context.
//   - Render deterministic validation summaries for operators.
//
// Scope:
//   - Policy-rule validation command binding and output only.
//
// Usage:
//   - Invoked through `acpctl validate policy-rules`.
//
// Invariants/Assumptions:
//   - Rule validation logic remains in `internal/policyengine`.
package main

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/policyengine"
)

type validatePolicyRulesOptions struct {
	RepoRoot  string
	RulesPath string
	Verbose   bool
}

func validatePolicyRulesCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    "policy-rules",
		Summary: "Validate the tracked ACP custom policy rule contract",
		Options: []commandOptionSpec{
			{Name: "file", ValueName: "PATH", Summary: "Policy rule file to validate", Type: optionValueString, DefaultText: policyengine.DefaultRulesPath},
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Bind: bindRepoParsed(bindValidatePolicyRulesOptions),
		Run:  runValidatePolicyRulesTyped,
	})
}

func bindValidatePolicyRulesOptions(bindCtx commandBindContext, input parsedCommandInput) (validatePolicyRulesOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return validatePolicyRulesOptions{}, err
	}
	options := validatePolicyRulesOptions{
		RepoRoot:  repoRoot,
		RulesPath: resolveRepoInput(repoRoot, policyengine.DefaultRulesPath),
		Verbose:   input.Bool("verbose"),
	}
	if input.Has("file") {
		options.RulesPath = resolveRepoInput(repoRoot, input.NormalizedString("file"))
	}
	return options, nil
}

func runValidatePolicyRulesTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	opts := raw.(validatePolicyRulesOptions)
	printCommandSection(runCtx.Stdout, out, "=== Custom Policy Rule Validation ===")

	rulesDoc, err := policyengine.LoadRulesFile(opts.RulesPath)
	if err != nil {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "custom policy rule validation failed")
	}
	validationCtx, err := policyengine.LoadValidationContext(opts.RepoRoot)
	if err != nil {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "custom policy rule validation failed")
	}
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stdout, "Policy rules: %s\n", opts.RulesPath)
		fmt.Fprintf(runCtx.Stdout, "Approved models: %d\n", len(validationCtx.ApprovedModels))
		fmt.Fprintf(runCtx.Stdout, "Tracked roles: %d\n", len(validationCtx.Roles))
	}
	issues := policyengine.ValidateRulesFile(rulesDoc, validationCtx)
	if len(issues) > 0 {
		return failValidation(runCtx.Stderr, out, issues, "Custom policy rule validation failed")
	}
	fmt.Fprintf(runCtx.Stdout, "Validated %d custom policy rule(s)\n", len(rulesDoc.Rules))
	fmt.Fprintln(runCtx.Stdout, out.Green("Custom policy rule validation passed"))
	return exitcodes.ACPExitSuccess
}
