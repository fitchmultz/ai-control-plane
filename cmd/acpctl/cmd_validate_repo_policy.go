// cmd_validate_repo_policy.go - Repository policy validation commands.
//
// Purpose:
//   - Own repo-policy-oriented validation commands that operate on tracked
//     sources and Compose surfaces.
//
// Responsibilities:
//   - Define typed Compose healthcheck, header, and env-access validators.
//   - Reuse the shared issue-rendering contract for policy failures.
//
// Scope:
//   - Repo-policy validation adapters only.
//
// Usage:
//   - Invoked through `acpctl validate <policy-command>`.
//
// Invariants/Assumptions:
//   - Validation logic remains in `internal/validation`.
package main

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

func validateComposeHealthchecksCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("compose-healthchecks", "Validate Docker Compose healthchecks", runValidateComposeHealthchecksTyped)
}

func validateHeadersCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("headers", "Validate Go source file header policy", runValidateHeadersTyped)
}

func validateEnvAccessCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("env-access", "Fail on direct environment access outside internal/config", runValidateEnvAccessTyped)
}

func runValidateComposeHealthchecksTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	return runIssueValidation(runCtx, nil, issueValidationConfig{
		Title:           "=== Docker Compose Healthchecks Validation ===",
		SuccessMessage:  "Healthcheck validation passed",
		FailureMessage:  "Healthcheck validation failed",
		RuntimeErrorMsg: "Healthcheck validation failed",
		ColorSuccess:    true,
	}, func() ([]string, error) {
		return validation.ValidateComposeHealthchecks(runCtx.RepoRoot)
	})
}

func runValidateHeadersTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	return runIssueValidation(runCtx, nil, issueValidationConfig{
		SuccessMessage:  "Go header policy validation passed",
		FailureMessage:  "Go header policy validation failed",
		RuntimeErrorMsg: "Go header policy validation failed",
	}, func() ([]string, error) {
		return validation.ValidateGoHeaders(runCtx.RepoRoot)
	})
}

func runValidateEnvAccessTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	return runIssueValidation(runCtx, nil, issueValidationConfig{
		SuccessMessage:  "Direct environment access policy passed",
		FailureMessage:  "Direct environment access policy failed",
		RuntimeErrorMsg: "Direct environment access policy failed",
	}, func() ([]string, error) {
		return validation.ValidateDirectEnvAccess(runCtx.RepoRoot)
	})
}
