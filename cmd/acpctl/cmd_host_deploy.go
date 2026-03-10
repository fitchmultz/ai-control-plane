// cmd_host_deploy.go - Native host check/apply command implementation.
//
// Purpose:
//   - Own the typed declarative Ansible-backed host deployment surface.
//
// Responsibilities:
//   - Define the `host check` and `host apply` commands.
//   - Validate local Ansible prerequisites and tracked repository surfaces.
//   - Execute syntax-checked playbook runs with explicit extra-vars wiring.
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
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const hostDeployTimeout = 60 * time.Minute

type hostDeployOptions struct {
	Mode                 string
	Inventory            string
	Limit                string
	RepoPath             string
	EnvFile              string
	TLSMode              string
	PublicURL            string
	WaitForStabilization bool
	RunSmokeTests        bool
	StabilizationSeconds string
	ExtraVars            []string
}

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
		Bind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (hostDeployOptions, error) {
			repoRoot, err := requireCommandRepoRoot(bindCtx)
			if err != nil {
				return hostDeployOptions{}, err
			}
			tlsMode := input.NormalizedString("tls-mode")
			if tlsMode != "" && tlsMode != "plain" && tlsMode != "tls" {
				return hostDeployOptions{}, fmt.Errorf("--tls-mode must be plain or tls")
			}
			stabilizationSeconds := input.NormalizedString("stabilization-seconds")
			if stabilizationSeconds != "" {
				if _, err := strconv.Atoi(stabilizationSeconds); err != nil {
					return hostDeployOptions{}, fmt.Errorf("invalid --stabilization-seconds value: %q", stabilizationSeconds)
				}
			}
			return hostDeployOptions{
				Mode:                 mode,
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
	opts := raw.(hostDeployOptions)
	playbookPath := filepath.Join(runCtx.RepoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml")
	ansibleCfg := filepath.Join(runCtx.RepoRoot, "deploy", "ansible", "ansible.cfg")

	if err := proc.ValidateExecutable("ansible-playbook"); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: ansible-playbook not found or not executable: %v\n", err)
		return exitcodes.ACPExitPrereq
	}
	if _, err := os.Stat(opts.Inventory); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: inventory file not found: %s\n", opts.Inventory)
		return exitcodes.ACPExitUsage
	}
	if _, err := os.Stat(playbookPath); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: playbook not found: %s\n", playbookPath)
		return exitcodes.ACPExitRuntime
	}
	if _, err := os.Stat(ansibleCfg); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: ansible config not found: %s\n", ansibleCfg)
		return exitcodes.ACPExitRuntime
	}

	ansibleArgs := []string{"-i", opts.Inventory, playbookPath}
	if opts.Limit != "" {
		ansibleArgs = append(ansibleArgs, "--limit", opts.Limit)
	}
	if opts.Mode == "check" {
		ansibleArgs = append(ansibleArgs, "--check")
	}
	for _, item := range hostDeployExtraVars(opts) {
		ansibleArgs = append(ansibleArgs, "--extra-vars", item)
	}

	request := proc.Request{
		Name:    "ansible-playbook",
		Dir:     runCtx.RepoRoot,
		Env:     []string{"ANSIBLE_CONFIG=" + ansibleCfg},
		Stdin:   os.Stdin,
		Stdout:  runCtx.Stdout,
		Stderr:  runCtx.Stderr,
		Timeout: hostDeployTimeout,
	}
	if code := runHostDeployProc(ctx, request, runCtx.Stderr, append([]string{"--syntax-check"}, ansibleArgs...), "host "+opts.Mode+" syntax-check"); code != exitcodes.ACPExitSuccess {
		return code
	}
	return runHostDeployProc(ctx, request, runCtx.Stderr, ansibleArgs, "host "+opts.Mode)
}

func hostDeployExtraVars(opts hostDeployOptions) []string {
	extraVars := append([]string(nil), opts.ExtraVars...)
	if opts.RepoPath != "" {
		extraVars = append(extraVars, "acp_repo_path="+opts.RepoPath)
	}
	if opts.EnvFile != "" {
		extraVars = append(extraVars, "acp_env_file="+opts.EnvFile)
	}
	if opts.TLSMode != "" {
		extraVars = append(extraVars, "acp_tls_mode="+opts.TLSMode)
	}
	if opts.PublicURL != "" {
		extraVars = append(extraVars, "acp_public_url="+opts.PublicURL)
	}
	extraVars = append(extraVars, "acp_wait_for_stabilization="+strconv.FormatBool(opts.WaitForStabilization))
	extraVars = append(extraVars, "acp_run_smoke_tests="+strconv.FormatBool(opts.RunSmokeTests))
	if opts.StabilizationSeconds != "" {
		extraVars = append(extraVars, "acp_stabilization_seconds="+opts.StabilizationSeconds)
	}
	return extraVars
}

func runHostDeployProc(ctx context.Context, request proc.Request, stderr io.Writer, args []string, commandName string) int {
	request.Args = args
	res := proc.Run(ctx, request)
	if res.Err == nil {
		return exitcodes.ACPExitSuccess
	}
	message, code := classifyProcFailure(res.Err, procFailureMessages{
		NotFound:         "ansible-playbook not found",
		Timeout:          fmt.Sprintf("%s timed out", commandName),
		Canceled:         fmt.Sprintf("%s canceled", commandName),
		Exit:             fmt.Sprintf("%s failed", commandName),
		ExitCodeOverride: exitcodes.ACPExitRuntime,
		Fallback:         fmt.Sprintf("%s failed: %v", commandName, res.Err),
	})
	if proc.IsExit(res.Err) && strings.Contains(commandName, "syntax-check") {
		code = exitcodes.ACPExitDomain
	}
	fmt.Fprintf(stderr, "Error: %s\n", message)
	return code
}
