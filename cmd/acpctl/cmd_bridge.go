// cmd_bridge.go - Bridge subcommand implementation
//
// Purpose: Bridge to legacy shell script implementations
// Responsibilities:
//   - Map bridge commands to shell scripts
//   - Execute bridge scripts with proper error handling
//
// Non-scope:
//   - Does not reimplement script logic in Go
//   - Does not validate script arguments

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

// bridgeScript defines a bridge command mapping
type bridgeScript struct {
	Name               string
	ScriptRelativePath string
	Description        string
	Usage              string
}

// bridgedScripts is the registry of available bridge commands
var bridgedScripts = []bridgeScript{
	{
		Name:               "host_deploy",
		ScriptRelativePath: "scripts/libexec/host_deploy_impl.sh",
		Description:        "Host declarative deployment orchestration",
		Usage:              "acpctl bridge host_deploy [check|apply] [options]",
	},
	{
		Name:               "host_install",
		ScriptRelativePath: "scripts/libexec/host_install_impl.sh",
		Description:        "Systemd host service installation/management",
		Usage:              "acpctl bridge host_install [command] [options]",
	},
	{
		Name:               "host_preflight",
		ScriptRelativePath: "scripts/libexec/host_preflight_impl.sh",
		Description:        "Host readiness preflight checks",
		Usage:              "acpctl bridge host_preflight [options]",
	},
	{
		Name:               "host_upgrade_slots",
		ScriptRelativePath: "scripts/libexec/host_upgrade_slots_impl.sh",
		Description:        "Slot-based host upgrade orchestration",
		Usage:              "acpctl bridge host_upgrade_slots [command] [options]",
	},
	{
		Name:               "onboard",
		ScriptRelativePath: "scripts/libexec/onboard_impl.sh",
		Description:        "Tool onboarding workflows",
		Usage:              "acpctl bridge onboard <tool> [options]",
	},
	{
		Name:               "prepare_secrets_env",
		ScriptRelativePath: "scripts/libexec/prepare_secrets_env_impl.sh",
		Description:        "Host secrets contract refresh/sync",
		Usage:              "acpctl bridge prepare_secrets_env [options]",
	},
	{
		Name:               "prod_smoke_helm",
		ScriptRelativePath: "scripts/libexec/prod_smoke_helm_impl.sh",
		Description:        "Helm production smoke workflow",
		Usage:              "acpctl bridge prod_smoke_helm [options]",
	},
	{
		Name:               "prod_smoke_test",
		ScriptRelativePath: "scripts/libexec/prod_smoke_test_impl.sh",
		Description:        "Runtime production smoke checks",
		Usage:              "acpctl bridge prod_smoke_test [options]",
	},
	{
		Name:               "release_bundle",
		ScriptRelativePath: "scripts/libexec/release_bundle_impl.sh",
		Description:        "Deployment release bundle build/verify",
		Usage:              "acpctl bridge release_bundle <build|verify> [options]",
	},
	{
		Name:               "switch_claude_mode",
		ScriptRelativePath: "scripts/libexec/switch_claude_mode_impl.sh",
		Description:        "Claude mode switching helper",
		Usage:              "acpctl bridge switch_claude_mode <mode|status> [options]",
	},
}

func lookupBridgeScript(name string) (bridgeScript, bool) {
	for _, script := range bridgedScripts {
		if script.Name == name {
			return script, true
		}
	}
	return bridgeScript{}, false
}

func printBridgeHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl bridge <name> [args...]

Execute mapped legacy script implementations directly (no make delegation).

Scripts:
`)
	for _, script := range bridgedScripts {
		fmt.Fprintf(out, "  %-22s %s\n", script.Name, script.Description)
		fmt.Fprintf(out, "                         Usage: %s\n", script.Usage)
	}
	fmt.Fprint(out, `
Examples:
  acpctl bridge host_preflight --profile production
  acpctl bridge host_deploy check --inventory deploy/ansible/inventory/hosts.yml
  acpctl bridge release_bundle build --version v1.2.3

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runBridgeSubcommand(args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printBridgeHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	if isHelpToken(args[0]) {
		printBridgeHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	script, ok := lookupBridgeScript(args[0])
	if !ok {
		fmt.Fprintf(stderr, "Error: Unknown bridge script: %s\n", args[0])
		printBridgeHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	return runBridgeScript(script, args[1:], stdout, stderr)
}

func runBridgeScript(script bridgeScript, scriptArgs []string, stdout *os.File, stderr *os.File) int {
	repoRoot := detectRepoRoot()
	if repoRoot == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}

	scriptPath := filepath.Join(repoRoot, script.ScriptRelativePath)
	info, err := os.Stat(scriptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "Error: bridge script not found: %s\n", scriptPath)
			return exitcodes.ACPExitPrereq
		}
		fmt.Fprintf(stderr, "Error: failed to stat bridge script %s: %v\n", scriptPath, err)
		return exitcodes.ACPExitRuntime
	}
	if info.IsDir() {
		fmt.Fprintf(stderr, "Error: bridge script path is a directory: %s\n", scriptPath)
		return exitcodes.ACPExitPrereq
	}
	if info.Mode()&0o111 == 0 {
		fmt.Fprintf(stderr, "Error: bridge script is not executable: %s\n", scriptPath)
		return exitcodes.ACPExitPrereq
	}

	cmd := exec.Command("/bin/bash", append([]string{scriptPath}, scriptArgs...)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintf(stderr, "Error: bash executable not found\n")
			return exitcodes.ACPExitPrereq
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case exitcodes.ACPExitDomain, exitcodes.ACPExitPrereq,
				exitcodes.ACPExitRuntime, exitcodes.ACPExitUsage:
				return exitErr.ExitCode()
			case 127:
				fmt.Fprintf(stderr, "Error: bridge script reported command not found\n")
				return exitcodes.ACPExitPrereq
			default:
				return exitcodes.ACPExitRuntime
			}
		}

		fmt.Fprintf(stderr, "Error: bridge script execution failed: %v\n", err)
		return exitcodes.ACPExitRuntime
	}

	return exitcodes.ACPExitSuccess
}
