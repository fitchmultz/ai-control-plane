// cmd_upgrade.go - Typed upgrade framework command implementation.
//
// Purpose:
//   - Own the typed host-first upgrade planning, validation, execution, and
//   - rollback command surface.
//
// Responsibilities:
//   - Define the public `upgrade` command tree and options.
//   - Adapt CLI input into typed upgrade workflow options.
//   - Render concise operator-facing results and exit codes.
//
// Scope:
//   - Upgrade command metadata and CLI adaptation only.
//
// Usage:
//   - Invoked through `acpctl upgrade <plan|check|execute|rollback>`.
//
// Invariants/Assumptions:
//   - Host-first upgrade execution runs from the target release checkout.
//   - Rollback runs from the previous release checkout.
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/hostdeploy"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/upgrade"
)

const defaultUpgradeSecretsEnvFile = "/etc/ai-control-plane/secrets.env"

func upgradeCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "upgrade",
		Summary:     "Plan, validate, execute, and roll back host-first upgrades",
		Description: "Plan, validate, execute, and roll back host-first upgrades.",
		Examples: []string{
			"acpctl upgrade plan --from 0.0.9",
			"acpctl upgrade check --from 0.0.9 --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env",
			"acpctl upgrade execute --from 0.0.9 --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env",
			"acpctl upgrade rollback --run-dir demo/logs/upgrades/upgrade-20260317T120000.000000000Z --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env",
		},
		Children: []*commandSpec{
			upgradeWorkflowCommandSpec("plan", runUpgradePlan),
			upgradeWorkflowCommandSpec("check", runUpgradeCheck),
			upgradeWorkflowCommandSpec("execute", runUpgradeExecute),
			upgradeRollbackCommandSpec(),
		},
	}
}

func upgradeWorkflowCommandSpec(name string, run func(context.Context, commandRunContext, any) int) *commandSpec {
	summary := map[string]string{
		"plan":    "Show the explicit supported upgrade plan",
		"check":   "Validate the upgrade path, config migrations, and host convergence",
		"execute": "Execute the supported host-first upgrade workflow",
	}[name]
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        name,
		Summary:     summary,
		Description: summary + ".",
		Options: []commandOptionSpec{
			{Name: "from", ValueName: "VERSION", Summary: "Current deployed version", Type: optionValueString, Required: true},
			{Name: "to", ValueName: "VERSION", Summary: "Target version (defaults to the checked-out VERSION file)", Type: optionValueString, DefaultText: "VERSION file"},
			{Name: "inventory", ValueName: "PATH", Summary: "Inventory file", Type: optionValueString, DefaultText: "deploy/ansible/inventory/hosts.yml"},
			{Name: "limit", ValueName: "TARGET", Summary: "Optional Ansible --limit selector", Type: optionValueString},
			{Name: "repo-path", ValueName: "PATH", Summary: "Override acp_repo_path", Type: optionValueString},
			{Name: "env-file", ValueName: "PATH", Summary: "Canonical secrets/env file", Type: optionValueString, DefaultText: defaultUpgradeSecretsEnvFile},
			{Name: "no-wait", Summary: "Set acp_wait_for_stabilization=false", Type: optionValueBool},
			{Name: "skip-smoke-tests", Summary: "Set acp_run_smoke_tests=false", Type: optionValueBool},
			{Name: "stabilization-seconds", ValueName: "N", Summary: "Override acp_stabilization_seconds", Type: optionValueString},
			{Name: "extra-var", ValueName: "KEY=VALUE", Summary: "Additional Ansible extra var", Type: optionValueString, Repeatable: true},
			{Name: "output-dir", ValueName: "DIR", Summary: "Upgrade artifact output root", Type: optionValueString, DefaultText: "demo/logs/upgrades"},
		},
		Bind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (upgrade.Options, error) {
			repoRoot, err := requireCommandRepoRoot(bindCtx)
			if err != nil {
				return upgrade.Options{}, err
			}
			return upgrade.Options{
				RepoRoot:             repoRoot,
				FromVersion:          input.NormalizedString("from"),
				ToVersion:            input.NormalizedString("to"),
				Inventory:            resolveRepoInput(repoRoot, input.StringDefault("inventory", "deploy/ansible/inventory/hosts.yml")),
				Limit:                input.NormalizedString("limit"),
				RepoPath:             input.NormalizedString("repo-path"),
				EnvFile:              resolveRepoInput(repoRoot, input.StringDefault("env-file", defaultUpgradeSecretsEnvFile)),
				OutputRoot:           resolveRepoInput(repoRoot, input.StringDefault("output-dir", "demo/logs/upgrades")),
				ExtraVars:            input.Strings("extra-var"),
				WaitForStabilization: !input.Bool("no-wait"),
				RunSmokeTests:        !input.Bool("skip-smoke-tests"),
				StabilizationSeconds: input.NormalizedString("stabilization-seconds"),
			}, nil
		}),
		Run: run,
	})
}

func upgradeRollbackCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "rollback",
		Summary:     "Restore the pre-upgrade snapshots from a recorded upgrade run",
		Description: "Restore the pre-upgrade snapshots from a recorded upgrade run.",
		Options: []commandOptionSpec{
			{Name: "run-dir", ValueName: "DIR", Summary: "Upgrade run directory", Type: optionValueString, Required: true},
			{Name: "inventory", ValueName: "PATH", Summary: "Inventory file", Type: optionValueString, DefaultText: "deploy/ansible/inventory/hosts.yml"},
			{Name: "limit", ValueName: "TARGET", Summary: "Optional Ansible --limit selector", Type: optionValueString},
			{Name: "repo-path", ValueName: "PATH", Summary: "Override acp_repo_path", Type: optionValueString},
			{Name: "env-file", ValueName: "PATH", Summary: "Canonical secrets/env file", Type: optionValueString, DefaultText: defaultUpgradeSecretsEnvFile},
			{Name: "no-wait", Summary: "Set acp_wait_for_stabilization=false", Type: optionValueBool},
			{Name: "skip-smoke-tests", Summary: "Set acp_run_smoke_tests=false", Type: optionValueBool},
			{Name: "stabilization-seconds", ValueName: "N", Summary: "Override acp_stabilization_seconds", Type: optionValueString},
			{Name: "extra-var", ValueName: "KEY=VALUE", Summary: "Additional Ansible extra var", Type: optionValueString, Repeatable: true},
		},
		Bind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (upgrade.RollbackOptions, error) {
			repoRoot, err := requireCommandRepoRoot(bindCtx)
			if err != nil {
				return upgrade.RollbackOptions{}, err
			}
			return upgrade.RollbackOptions{
				RepoRoot:             repoRoot,
				RunDir:               resolveRepoInput(repoRoot, input.NormalizedString("run-dir")),
				Inventory:            resolveRepoInput(repoRoot, input.StringDefault("inventory", "deploy/ansible/inventory/hosts.yml")),
				Limit:                input.NormalizedString("limit"),
				RepoPath:             input.NormalizedString("repo-path"),
				EnvFile:              resolveRepoInput(repoRoot, input.StringDefault("env-file", defaultUpgradeSecretsEnvFile)),
				ExtraVars:            input.Strings("extra-var"),
				WaitForStabilization: !input.Bool("no-wait"),
				RunSmokeTests:        !input.Bool("skip-smoke-tests"),
				StabilizationSeconds: input.NormalizedString("stabilization-seconds"),
			}, nil
		}),
		Run: runUpgradeRollback,
	})
}

func runUpgradePlan(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(upgrade.Options)
	out := output.New()
	plan, err := upgrade.PlanForOptions(opts)
	if err != nil {
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(runCtx.Stdout, out.Bold("Upgrade plan"))
	fmt.Fprintf(runCtx.Stdout, "  From: %s\n", plan.FromVersion)
	fmt.Fprintf(runCtx.Stdout, "  To:   %s\n", plan.ToVersion)
	fmt.Fprintf(runCtx.Stdout, "  Path: %s\n", strings.Join(plan.Path, " -> "))
	for index, step := range plan.Steps {
		fmt.Fprintf(runCtx.Stdout, "  %d. %s\n", index+1, step)
	}
	if len(plan.Compatibility) > 0 {
		fmt.Fprintln(runCtx.Stdout, "Compatibility:")
		for _, item := range plan.Compatibility {
			fmt.Fprintf(runCtx.Stdout, "  - %s\n", item)
		}
	}
	if len(plan.Rollback) > 0 {
		fmt.Fprintln(runCtx.Stdout, "Rollback notes:")
		for _, item := range plan.Rollback {
			fmt.Fprintf(runCtx.Stdout, "  - %s\n", item)
		}
	}
	return exitcodes.ACPExitSuccess
}

func runUpgradeCheck(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(upgrade.Options)
	opts.Stdout = runCtx.Stdout
	opts.Stderr = runCtx.Stderr
	out := output.New()
	plan, err := upgrade.Check(ctx, opts)
	if err != nil {
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitCodeForUpgradeError(err, exitcodes.ACPExitDomain)
	}
	fmt.Fprintln(runCtx.Stdout, out.Green("Upgrade check passed"))
	fmt.Fprintf(runCtx.Stdout, "Path: %s\n", strings.Join(plan.Path, " -> "))
	return exitcodes.ACPExitSuccess
}

func runUpgradeExecute(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(upgrade.Options)
	opts.Stdout = runCtx.Stdout
	opts.Stderr = runCtx.Stderr
	out := output.New()
	summary, err := upgrade.Execute(ctx, opts)
	if err != nil {
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitCodeForUpgradeError(err, exitcodes.ACPExitRuntime)
	}
	fmt.Fprintln(runCtx.Stdout, out.Green("Upgrade completed"))
	fmt.Fprintf(runCtx.Stdout, "Run directory: %s\n", summary.RunDirectory)
	fmt.Fprintf(runCtx.Stdout, "Rollback: acpctl upgrade rollback --run-dir %s --inventory %s --env-file %s\n", summary.RunDirectory, summary.Inventory, summary.EnvFile)
	return exitcodes.ACPExitSuccess
}

func runUpgradeRollback(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(upgrade.RollbackOptions)
	opts.Stdout = runCtx.Stdout
	opts.Stderr = runCtx.Stderr
	out := output.New()
	summary, err := upgrade.Rollback(ctx, opts)
	if err != nil {
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitCodeForUpgradeError(err, exitcodes.ACPExitRuntime)
	}
	fmt.Fprintln(runCtx.Stdout, out.Green("Rollback completed"))
	fmt.Fprintf(runCtx.Stdout, "Restored version: %s\n", summary.FromVersion)
	return exitcodes.ACPExitSuccess
}

func exitCodeForUpgradeError(err error, fallback int) int {
	var deployErr *hostdeploy.Error
	if errors.As(err, &deployErr) {
		switch deployErr.Kind {
		case hostdeploy.ErrorKindPrereq:
			return exitcodes.ACPExitPrereq
		case hostdeploy.ErrorKindUsage, hostdeploy.ErrorKindDomain:
			return exitcodes.ACPExitDomain
		default:
			return fallback
		}
	}
	switch {
	case upgrade.IsKind(err, upgrade.ErrorKindPrereq):
		return exitcodes.ACPExitPrereq
	case upgrade.IsKind(err, upgrade.ErrorKindDomain):
		return exitcodes.ACPExitDomain
	default:
		return fallback
	}
}
