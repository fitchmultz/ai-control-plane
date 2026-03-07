// cmd_delegated.go - Make-delegated command implementation
//
// Purpose: Route grouped CLI subcommands to native handlers or Make targets.
// Responsibilities:
//   - Define grouped command metadata and Make target mappings.
//   - Dispatch native overrides where typed behavior exists.
//   - Preserve delegated execution for Make-owned workflows.
// Scope:
//   - Command selection, help rendering, and Make process invocation.
// Usage:
//   - Invoked by top-level CLI parsing for grouped commands such as validate/db.
// Invariants/Assumptions:
//   - Make-owned policy remains authoritative for delegated subcommands.
//   - Native handlers must own their own help text when explicitly routed here.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

// makeDelegatedSubcommand represents a subcommand that delegates to Make
type makeDelegatedSubcommand struct {
	Name        string
	MakeTarget  string
	Description string
}

// makeDelegatedGroup groups related delegated subcommands
type makeDelegatedGroup struct {
	Name        string
	Description string
	Examples    []string
	Subcommands []makeDelegatedSubcommand
}

// delegatedGroups defines all command groups that delegate to Make
var delegatedGroups = []makeDelegatedGroup{
	{
		Name:        "deploy",
		Description: "Service lifecycle, release, and deployment operations",
		Examples: []string{
			"acpctl deploy up",
			"acpctl deploy health",
			"acpctl deploy up-production",
			"acpctl deploy up-tls",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "up", MakeTarget: "up", Description: "Start default services"},
			{Name: "down", MakeTarget: "down", Description: "Stop default services"},
			{Name: "restart", MakeTarget: "restart", Description: "Restart default services"},
			{Name: "health", MakeTarget: "health", Description: "Run service health checks"},
			{Name: "logs", MakeTarget: "logs", Description: "Tail service logs"},
			{Name: "ps", MakeTarget: "ps", Description: "Show running services"},
			{Name: "up-production", MakeTarget: "up-production", Description: "Start production profile services"},
			{Name: "prod-smoke", MakeTarget: "prod-smoke", Description: "Run production smoke tests"},
			{Name: "up-offline", MakeTarget: "up-offline", Description: "Start offline mode services"},
			{Name: "down-offline", MakeTarget: "down-offline", Description: "Stop offline mode services"},
			{Name: "health-offline", MakeTarget: "health-offline", Description: "Run offline mode health checks"},
			{Name: "up-tls", MakeTarget: "up-tls", Description: "Start TLS mode services"},
			{Name: "down-tls", MakeTarget: "down-tls", Description: "Stop TLS mode services"},
			{Name: "tls-health", MakeTarget: "tls-health", Description: "Run TLS health checks"},
			{Name: "helm-validate", MakeTarget: "helm-validate", Description: "Validate Helm chart"},
			{Name: "release-bundle", MakeTarget: "release-bundle", Description: "Build deployment release bundle"},
			{Name: "readiness-evidence", MakeTarget: "readiness-evidence", Description: "Generate and verify dated readiness evidence"},
			{Name: "pilot-closeout-bundle", MakeTarget: "pilot-closeout-bundle", Description: "Assemble and verify a pilot closeout evidence bundle"},
			{Name: "artifact-retention", MakeTarget: "artifact-retention", Description: "Enforce document artifact retention policy"},
		},
	},
	{
		Name:        "validate",
		Description: "Configuration and policy validation operations",
		Examples: []string{
			"acpctl validate config",
			"acpctl validate lint",
			"acpctl validate detections",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "lint", MakeTarget: "lint", Description: "Run static validation/lint gate"},
			{Name: "config", MakeTarget: "validate-config", Description: "Validate demo deployment configuration"},
			{Name: "detections", MakeTarget: "validate-detections", Description: "Validate detection rule output"},
			{Name: "siem-queries", MakeTarget: "validate-siem-queries", Description: "Validate SIEM query sync"},
			{Name: "network-contract", MakeTarget: "network-contract", Description: "Render network contract artifacts"},
			{Name: "supply-chain", MakeTarget: "supply-chain-gate", Description: "Run supply-chain security gate"},
			{Name: "secrets-audit", MakeTarget: "secrets-audit", Description: "Run deterministic tracked-file secrets audit"},
			{Name: "compose-healthchecks", MakeTarget: "validate-compose-healthchecks", Description: "Validate Docker Compose healthchecks"},
			{Name: "security", MakeTarget: "security-gate", Description: "Run Make-composed security gate (hygiene, secrets, license, supply chain)"},
		},
	},
	{
		Name:        "db",
		Description: "Database backup, restore, and inspection operations",
		Examples: []string{
			"acpctl db status",
			"acpctl db backup",
			"acpctl db dr-drill",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "status", MakeTarget: "db-status", Description: "Show database status and statistics"},
			{Name: "backup", MakeTarget: "db-backup", Description: "Create database backup"},
			{Name: "restore", MakeTarget: "db-restore", Description: "Restore embedded database from backup"},
			{Name: "shell", MakeTarget: "db-shell", Description: "Open database shell"},
			{Name: "dr-drill", MakeTarget: "dr-drill", Description: "Run database DR restore drill"},
		},
	},
	{
		Name:        "key",
		Description: "Virtual key lifecycle operations",
		Examples: []string{
			"acpctl key gen alice --budget 10.00",
			"acpctl key revoke alice",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "gen", MakeTarget: "key-gen", Description: "Generate a standard virtual key"},
			{Name: "revoke", MakeTarget: "key-revoke", Description: "Revoke a virtual key by alias"},
			{Name: "gen-dev", MakeTarget: "key-gen-dev", Description: "Generate a developer key"},
			{Name: "gen-lead", MakeTarget: "key-gen-lead", Description: "Generate a team-lead key"},
		},
	},
	{
		Name:        "host",
		Description: "Host-first deployment and operations",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host check",
			"acpctl host apply",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "preflight", MakeTarget: "host-preflight", Description: "Validate host readiness"},
			{Name: "check", MakeTarget: "host-check", Description: "Run declarative host preflight/check mode"},
			{Name: "apply", MakeTarget: "host-apply", Description: "Run declarative host apply/converge"},
			{Name: "install", MakeTarget: "host-install", Description: "Install systemd service"},
			{Name: "service-status", MakeTarget: "host-service-status", Description: "Show service status"},
		},
	},
	{
		Name:        "demo",
		Description: "Demo scenario, preset, and snapshot operations",
		Examples: []string{
			"acpctl demo scenario SCENARIO=1",
			"acpctl demo preset PRESET=executive-demo",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "scenario", MakeTarget: "demo-scenario", Description: "Run a specific demo scenario"},
			{Name: "all", MakeTarget: "demo-all", Description: "Run all demo scenarios"},
			{Name: "preset", MakeTarget: "demo-preset", Description: "Run a named demo preset"},
			{Name: "snapshot", MakeTarget: "demo-snapshot", Description: "Create a named demo snapshot"},
			{Name: "restore", MakeTarget: "demo-restore", Description: "Restore a named demo snapshot"},
		},
	},
	{
		Name:        "terraform",
		Description: "Terraform provisioning workflow helpers",
		Examples: []string{
			"acpctl terraform init",
			"acpctl terraform plan",
			"acpctl terraform apply",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "init", MakeTarget: "tf-init", Description: "Initialize Terraform"},
			{Name: "plan", MakeTarget: "tf-plan", Description: "Run Terraform plan"},
			{Name: "apply", MakeTarget: "tf-apply", Description: "Run Terraform apply"},
			{Name: "destroy", MakeTarget: "tf-destroy", Description: "Run Terraform destroy"},
			{Name: "fmt", MakeTarget: "tf-fmt", Description: "Format Terraform files"},
			{Name: "validate", MakeTarget: "tf-validate", Description: "Validate Terraform configuration"},
		},
	},
	{
		Name:        "helm",
		Description: "Helm chart validation and smoke tests",
		Examples: []string{
			"acpctl helm validate",
			"acpctl helm smoke",
		},
		Subcommands: []makeDelegatedSubcommand{
			{Name: "validate", MakeTarget: "helm-validate", Description: "Validate Helm chart"},
			{Name: "smoke", MakeTarget: "prod-smoke-helm", Description: "Run Helm production smoke tests"},
		},
	},
}

func lookupDelegatedGroup(name string) (makeDelegatedGroup, bool) {
	for _, group := range delegatedGroups {
		if group.Name == name {
			return group, true
		}
	}
	return makeDelegatedGroup{}, false
}

func lookupSubcommand(group makeDelegatedGroup, name string) (makeDelegatedSubcommand, bool) {
	for _, sub := range group.Subcommands {
		if sub.Name == name {
			return sub, true
		}
	}
	return makeDelegatedSubcommand{}, false
}

func printDelegatedGroupHelp(out *os.File, group makeDelegatedGroup) {
	fmt.Fprintf(out, "Usage: acpctl %s <subcommand> [make args]\n\n", group.Name)
	fmt.Fprintf(out, "%s.\n\n", group.Description)
	fmt.Fprintln(out, "Subcommands:")
	for _, sub := range group.Subcommands {
		fmt.Fprintf(out, "  %-22s %s\n", sub.Name, sub.Description)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	for _, example := range group.Examples {
		fmt.Fprintf(out, "  %s\n", example)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Exit codes:")
	fmt.Fprintln(out, "  0   Success")
	fmt.Fprintln(out, "  1   Domain non-success")
	fmt.Fprintln(out, "  2   Prerequisites not ready")
	fmt.Fprintln(out, "  3   Runtime/internal error")
	fmt.Fprintln(out, "  64  Usage error")
}

func printDelegatedSubcommandHelp(out *os.File, group makeDelegatedGroup, sub makeDelegatedSubcommand) {
	fmt.Fprintf(out, "Usage: acpctl %s %s [make args]\n\n", group.Name, sub.Name)
	fmt.Fprintf(out, "%s\n\n", sub.Description)
	fmt.Fprintf(out, "Delegates to make target: %s\n\n", sub.MakeTarget)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintf(out, "  acpctl %s %s\n", group.Name, sub.Name)
	fmt.Fprintf(out, "  acpctl %s %s VERBOSE=1\n", group.Name, sub.Name)
}

func runDelegatedGroup(group makeDelegatedGroup, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printDelegatedGroupHelp(stdout, group)
		return exitcodes.ACPExitUsage
	}

	if isHelpToken(args[0]) {
		printDelegatedGroupHelp(stdout, group)
		return exitcodes.ACPExitSuccess
	}

	subcommand, ok := lookupSubcommand(group, args[0])
	if !ok {
		fmt.Fprintf(stderr, "Error: Unknown %s subcommand: %s\n", group.Name, args[0])
		printDelegatedGroupHelp(stderr, group)
		return exitcodes.ACPExitUsage
	}

	if group.Name == "validate" {
		if code, handled := runNativeValidateSubcommand(subcommand.Name, args[1:], stdout, stderr); handled {
			return code
		}
	}

	if len(args) > 1 && isHelpToken(args[1]) {
		printDelegatedSubcommandHelp(stdout, group, subcommand)
		return exitcodes.ACPExitSuccess
	}

	// Native Go implementations for specific commands
	if group.Name == "db" {
		switch subcommand.Name {
		case "backup":
			return runDBBackupCommand(args[1:], stdout, stderr)
		case "restore":
			return runDBRestoreCommand(args[1:], stdout, stderr)
		case "dr-drill":
			return runDBDRDrill(args[1:], stdout, stderr)
		}
	}

	if group.Name == "key" {
		switch subcommand.Name {
		case "gen":
			return runKeyGenCommand(args[1:], stdout, stderr)
		case "gen-dev":
			// Prepend --role developer if not already specified
			return runKeyGenCommand(append([]string{"--role", "developer"}, args[1:]...), stdout, stderr)
		case "gen-lead":
			// Prepend --role team-lead if not already specified
			return runKeyGenCommand(append([]string{"--role", "team-lead"}, args[1:]...), stdout, stderr)
		case "revoke":
			return runKeyRevokeCommand(args[1:], stdout, stderr)
		}
	}

	if group.Name == "deploy" {
		switch subcommand.Name {
		case "artifact-retention":
			return runArtifactRetentionCommand(args[1:], stdout, stderr)
		case "readiness-evidence":
			return runReadinessEvidenceCommand(args[1:], stdout, stderr)
		case "pilot-closeout-bundle":
			return runPilotCloseoutBundleCommand(args[1:], stdout, stderr)
		case "release-bundle":
			return runReleaseBundleCommand(args[1:], stdout, stderr)
		}
	}

	if group.Name == "helm" {
		switch subcommand.Name {
		case "validate":
			return runHelmValidateCommand(args[1:], stdout, stderr)
		case "smoke":
			return runHelmSmokeCommand(args[1:], stdout, stderr)
		}
	}

	return runMakeTarget(subcommand.MakeTarget, args[1:], stdout, stderr)
}

func runNativeValidateSubcommand(name string, args []string, stdout *os.File, stderr *os.File) (int, bool) {
	switch name {
	case "detections":
		return runValidateDetections(args, stdout, stderr), true
	case "siem-queries":
		return runValidateSiemQueries(args, stdout, stderr), true
	case "config":
		return runValidateConfig(args, stdout, stderr), true
	case "compose-healthchecks":
		return runValidateComposeHealthchecks(args, stdout, stderr), true
	case "secrets-audit":
		return runSecretsAudit(args, stdout, stderr), true
	default:
		return 0, false
	}
}

func runMakeTarget(target string, makeArgs []string, stdout *os.File, stderr *os.File) int {
	makeBin := strings.TrimSpace(os.Getenv("ACPCTL_MAKE_BIN"))
	if makeBin == "" {
		makeBin = "make"
	}
	if err := ensureExecutable(makeBin); err != nil {
		fmt.Fprintf(stderr, "Error: make executable not found or not executable: %s\n", makeBin)
		return exitcodes.ACPExitPrereq
	}

	repoRoot := detectRepoRoot()
	cmd := exec.Command(makeBin, append([]string{target}, makeArgs...)...)
	cmd.Dir = repoRoot
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintf(stderr, "Error: make executable not found: %s\n", makeBin)
			return exitcodes.ACPExitPrereq
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case exitcodes.ACPExitDomain, exitcodes.ACPExitPrereq,
				exitcodes.ACPExitRuntime, exitcodes.ACPExitUsage:
				return exitErr.ExitCode()
			default:
				return exitcodes.ACPExitRuntime
			}
		}

		fmt.Fprintf(stderr, "Error: make target execution failed: %v\n", err)
		return exitcodes.ACPExitRuntime
	}

	return exitcodes.ACPExitSuccess
}

func ensureExecutable(command string) error {
	if strings.ContainsRune(command, filepath.Separator) {
		info, err := os.Stat(command)
		if err != nil {
			return err
		}
		if info.Mode().IsDir() {
			return errors.New("command path is a directory")
		}
		if info.Mode()&0o111 == 0 {
			return errors.New("command path is not executable")
		}
		return nil
	}

	_, err := exec.LookPath(command)
	return err
}
