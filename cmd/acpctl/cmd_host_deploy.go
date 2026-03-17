// cmd_host_deploy.go - Native host check/apply command implementation.
//
// Purpose:
//   - Own the typed declarative Ansible-backed host deployment surface.
//
// Responsibilities:
//   - Define the `host check` and `host apply` commands.
//   - Validate local CLI input before handing execution to the shared hostdeploy package.
//   - Map typed host deployment failures onto the ACP exit-code contract.
//
// Scope:
//   - Local command adaptation for host deployment only.
//
// Usage:
//   - Invoked through `acpctl host check` and `acpctl host apply`.
//
// Invariants/Assumptions:
//   - `deploy/ansible/playbooks/gateway_host.yml` remains the source of truth.
//   - Remote-path flags are passed through unchanged to Ansible extra vars.
package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/hostdeploy"
)

func hostDeployCommandSpec(mode string) *commandSpec {
	summary := "Run declarative host apply/converge"
	examples := []string{
		"acpctl host apply --inventory deploy/ansible/inventory/hosts.yml",
		"acpctl host apply --inventory deploy/ansible/inventory/hosts.yml --limit gateway",
		"acpctl host apply --repo-path /opt/ai-control-plane --tls-mode tls --public-url https://gateway.example.com",
	}
	if mode == "check" {
		summary = "Run declarative host preflight/check mode"
		examples = []string{
			"acpctl host check --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl host check --inventory deploy/ansible/inventory/hosts.yml --limit gateway",
		}
	}
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        mode,
		Summary:     summary,
		Description: summary + ".",
		Examples:    examples,
		Options: []commandOptionSpec{
			{Name: "inventory", ValueName: "PATH", Summary: "Inventory file", Type: optionValueString, DefaultText: "deploy/ansible/inventory/hosts.yml"},
			{Name: "limit", ValueName: "TARGET", Summary: "Optional Ansible --limit selector", Type: optionValueString},
			{Name: "repo-path", ValueName: "PATH", Summary: "Override acp_repo_path", Type: optionValueString},
			{Name: "env-file", ValueName: "PATH", Summary: "Override acp_env_file", Type: optionValueString},
			{Name: "tls-mode", ValueName: "plain|tls", Summary: "Override acp_tls_mode", Type: optionValueString},
			{Name: "public-url", ValueName: "URL", Summary: "Override acp_public_url", Type: optionValueString},
			{Name: "no-wait", Summary: "Set acp_wait_for_stabilization=false", Type: optionValueBool},
			{Name: "skip-smoke-tests", Summary: "Set acp_run_smoke_tests=false", Type: optionValueBool},
			{Name: "stabilization-seconds", ValueName: "N", Summary: "Override acp_stabilization_seconds", Type: optionValueString},
			{Name: "extra-var", ValueName: "KEY=VALUE", Summary: "Additional Ansible extra var", Type: optionValueString, Repeatable: true},
		},
		Bind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (hostdeploy.Options, error) {
			repoRoot, err := requireCommandRepoRoot(bindCtx)
			if err != nil {
				return hostdeploy.Options{}, err
			}
			tlsMode := input.NormalizedString("tls-mode")
			if tlsMode != "" && tlsMode != "plain" && tlsMode != "tls" {
				return hostdeploy.Options{}, fmt.Errorf("--tls-mode must be plain or tls")
			}
			stabilizationSeconds := input.NormalizedString("stabilization-seconds")
			if stabilizationSeconds != "" {
				if _, err := strconv.Atoi(stabilizationSeconds); err != nil {
					return hostdeploy.Options{}, fmt.Errorf("invalid --stabilization-seconds value: %q", stabilizationSeconds)
				}
			}
			return hostdeploy.Options{
				Mode:                 mode,
				RepoRoot:             repoRoot,
				Inventory:            resolveRepoInput(repoRoot, input.StringDefault("inventory", "deploy/ansible/inventory/hosts.yml")),
				Limit:                input.NormalizedString("limit"),
				RepoPath:             input.NormalizedString("repo-path"),
				EnvFile:              input.NormalizedString("env-file"),
				TLSMode:              tlsMode,
				PublicURL:            input.NormalizedString("public-url"),
				WaitForStabilization: !input.Bool("no-wait"),
				RunSmokeTests:        !input.Bool("skip-smoke-tests"),
				StabilizationSeconds: stabilizationSeconds,
				ExtraVars:            input.Strings("extra-var"),
			}, nil
		}),
		Run: runHostDeploy,
	})
}

func runHostDeploy(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(hostdeploy.Options)
	opts.Stdout = runCtx.Stdout
	opts.Stderr = runCtx.Stderr
	if err := hostdeploy.Execute(ctx, opts); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %s\n", err)
		switch err := err.(type) {
		case *hostdeploy.Error:
			switch err.Kind {
			case hostdeploy.ErrorKindPrereq:
				return exitcodes.ACPExitPrereq
			case hostdeploy.ErrorKindUsage:
				return exitcodes.ACPExitUsage
			case hostdeploy.ErrorKindDomain:
				return exitcodes.ACPExitDomain
			default:
				return exitcodes.ACPExitRuntime
			}
		default:
			return exitcodes.ACPExitRuntime
		}
	}
	return exitcodes.ACPExitSuccess
}
