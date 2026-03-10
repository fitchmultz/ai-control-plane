// cmd_host_preflight.go - Native host preflight command implementation.
//
// Purpose:
//   - Own the typed host-first preflight gate without routing through an
//     internal bridge script.
//
// Responsibilities:
//   - Validate required local binaries for host-first workflows.
//   - Verify the tracked systemd unit template and compose env parent path.
//   - Run the canonical production deployment-contract validation.
//
// Scope:
//   - Local host preflight checks only.
//
// Usage:
//   - Invoked through `acpctl host preflight`.
//
// Invariants/Assumptions:
//   - Only the production profile is supported.
//   - Relative paths resolve from the repository root.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

const defaultHostSecretsEnvFile = "/etc/ai-control-plane/secrets.env"

type hostPreflightOptions struct {
	Profile        string
	SecretsEnvFile string
	ComposeEnvFile string
}

func hostPreflightCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "preflight",
		Summary:     "Validate host readiness",
		Description: "Validate the local host-first runtime contract before installation or startup.",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host preflight --secrets-env-file /etc/ai-control-plane/secrets.env",
		},
		Options: []commandOptionSpec{
			{Name: "profile", ValueName: "NAME", Summary: "Deployment profile (only production is supported)", Type: optionValueString, DefaultText: "production"},
			{Name: "secrets-env-file", ValueName: "PATH", Summary: "Canonical production secrets file", Type: optionValueString, DefaultText: defaultHostSecretsEnvFile},
			{Name: "compose-env-file", ValueName: "PATH", Summary: "Compose runtime env file", Type: optionValueString, DefaultText: "demo/.env"},
		},
		Bind: bindRepoParsed(bindHostPreflightOptions),
		Run:  runHostPreflight,
	})
}

func bindHostPreflightOptions(bindCtx commandBindContext, input parsedCommandInput) (hostPreflightOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return hostPreflightOptions{}, err
	}
	profile := input.StringDefault("profile", "production")
	if profile != "production" {
		return hostPreflightOptions{}, fmt.Errorf("unsupported profile: %s", profile)
	}
	return hostPreflightOptions{
		Profile:        profile,
		SecretsEnvFile: resolveRepoInput(repoRoot, input.StringDefault("secrets-env-file", defaultHostSecretsEnvFile)),
		ComposeEnvFile: resolveRepoInput(repoRoot, input.StringDefault("compose-env-file", "demo/.env")),
	}, nil
}

func runHostPreflight(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(hostPreflightOptions)
	out := output.New()
	logger := workflowLogger(runCtx, "host_preflight", "profile", opts.Profile)
	workflowStart(logger)
	printCommandSection(runCtx.Stdout, out, "=== Host Preflight ===")
	fmt.Fprintf(runCtx.Stdout, "Profile: %s\n", opts.Profile)
	fmt.Fprintf(runCtx.Stdout, "Secrets file: %s\n", opts.SecretsEnvFile)
	fmt.Fprintf(runCtx.Stdout, "Compose env file: %s\n", opts.ComposeEnvFile)

	if err := requireHostPreflightPrereqs(runCtx.RepoRoot); err != nil {
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitPrereq, err, "Host preflight prerequisites failed")
	}

	templatePath := filepath.Join(runCtx.RepoRoot, "deploy", "systemd", "ai-control-plane.service.tmpl")
	if _, err := os.Stat(templatePath); err != nil {
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "Missing systemd template")
	}
	composeEnvDir := filepath.Dir(opts.ComposeEnvFile)
	if info, err := os.Stat(composeEnvDir); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "Compose env parent directory not found")
	}

	issues, err := validation.ValidateDeploymentConfig(runCtx.RepoRoot, validation.ConfigValidationOptions{
		Profile:        validation.ConfigValidationProfileProduction,
		SecretsEnvFile: opts.SecretsEnvFile,
	})
	if err != nil {
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "Host preflight validation failed")
	}
	if len(issues) > 0 {
		workflowWarn(logger, "issues", len(issues))
		return failValidation(runCtx.Stderr, out, issues, "Host preflight failed")
	}

	fmt.Fprintln(runCtx.Stdout, out.Green("Host preflight passed"))
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func requireHostPreflightPrereqs(repoRoot string) error {
	if !prereq.CommandExists("docker") {
		return fmt.Errorf("required command not found: docker")
	}
	makeBin := config.NewLoader().WithRepoRoot(repoRoot).Tooling().MakeBinary
	if err := proc.ValidateExecutable(makeBin); err != nil {
		return fmt.Errorf("required command not found or not executable: %s", makeBin)
	}
	if !prereq.CommandExists("systemctl") {
		return fmt.Errorf("required command not found: systemctl")
	}
	if _, err := docker.DetectCompose(); err != nil {
		return fmt.Errorf("docker compose not available: %w", err)
	}
	return nil
}
