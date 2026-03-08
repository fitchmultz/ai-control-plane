// command_registry.go - Canonical acpctl command catalog
//
// Purpose:
//
//	Provide the single source of truth for visible acpctl command metadata,
//	dispatch ownership, and help/completion structure.
//
// Responsibilities:
//   - Define native root commands and grouped subcommands in one catalog.
//   - Encode whether a subcommand is native, Make-backed, or bridge-script-backed.
//   - Keep user-facing command ordering deterministic for help and completion.
//
// Scope:
//   - Metadata and lookup helpers only; concrete business logic stays in command handlers.
//
// Usage:
//   - Consumed by main.go, completion generation, and grouped command dispatch.
//
// Invariants/Assumptions:
//   - Hidden/internal commands stay out of visible command lists.
//   - Every grouped subcommand resolves to exactly one execution owner.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
)

type commandRunner func(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int

type commandDescriptor struct {
	Name        string
	Description string
}

type subcommandDefinition struct {
	commandDescriptor
	MakeTarget         string
	NativeRun          commandRunner
	ScriptRelativePath string
	Usage              string
}

type rootCommandDefinition struct {
	commandDescriptor
	NativeRun   commandRunner
	Subcommands []subcommandDefinition
	Examples    []string
	Hidden      bool
}

type commandCatalog struct {
	RootCommands []rootCommandDefinition
}

type commandRegistry struct {
	RootCommands     []commandDescriptor
	GroupSubcommands map[string][]commandDescriptor
}

type compiledCommandState struct {
	Catalog           commandCatalog
	Registry          commandRegistry
	RootByName        map[string]rootCommandDefinition
	SubcommandsByRoot map[string]map[string]subcommandDefinition
}

type commandLookupError struct {
	Kind string
	Root string
	Name string
}

func (e *commandLookupError) Error() string {
	switch e.Kind {
	case "root":
		return fmt.Sprintf("unknown root command: %s", e.Name)
	case "native-root":
		return fmt.Sprintf("command is not a native root command: %s", e.Name)
	case "subcommand":
		return fmt.Sprintf("unknown subcommand %s for root %s", e.Name, e.Root)
	default:
		return "invalid command lookup"
	}
}

var (
	commandStateOnce sync.Once
	commandState     compiledCommandState
	commandStateErr  error
)

func buildCommandCatalog() commandCatalog {
	return commandCatalog{
		RootCommands: []rootCommandDefinition{
			{
				commandDescriptor: commandDescriptor{
					Name:        "ci",
					Description: "CI and local gate helpers",
				},
				NativeRun: runCISubcommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "should-run-runtime", Description: "Decide whether runtime checks should run"}},
					{commandDescriptor: commandDescriptor{Name: "wait", Description: "Wait for services to become healthy"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "files",
					Description: "Typed local file synchronization helpers",
				},
				NativeRun: runFilesSubcommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "sync-helm", Description: "Synchronize canonical repository files into Helm chart files/"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "env",
					Description: "Strict .env access helpers",
				},
				NativeRun: runEnvCommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "get", Description: "Read a single env key without shell execution"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "chargeback",
					Description: "Typed chargeback rendering helpers",
				},
				NativeRun: runChargebackCommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "render", Description: "Render canonical chargeback JSON or CSV"}},
					{commandDescriptor: commandDescriptor{Name: "payload", Description: "Render canonical chargeback webhook payload JSON"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "status",
					Description: "Aggregated system health overview",
				},
				NativeRun: runStatusCommand,
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "health",
					Description: "Run service health checks",
				},
				NativeRun: runHealthCommand,
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "doctor",
					Description: "Environment preflight diagnostics",
				},
				NativeRun: runDoctorCommand,
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "benchmark",
					Description: "Lightweight local performance baseline",
				},
				NativeRun: runBenchmarkCommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "baseline", Description: "Run the local gateway performance baseline"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "completion",
					Description: "Generate shell completion scripts",
				},
				NativeRun: runCompletionSubcommand,
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "bash", Description: "Generate Bash completion script"}},
					{commandDescriptor: commandDescriptor{Name: "zsh", Description: "Generate Zsh completion script"}},
					{commandDescriptor: commandDescriptor{Name: "fish", Description: "Generate Fish completion script"}},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "onboard",
					Description: "Configure local tools to route through the gateway",
				},
				NativeRun: runOnboardCommand,
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "deploy",
					Description: "Service lifecycle, release, and deployment operations",
				},
				Examples: []string{
					"acpctl deploy up",
					"acpctl deploy health",
					"acpctl deploy up-production",
					"acpctl deploy up-tls",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "up", Description: "Start default services"}, MakeTarget: "up"},
					{commandDescriptor: commandDescriptor{Name: "down", Description: "Stop default services"}, MakeTarget: "down"},
					{commandDescriptor: commandDescriptor{Name: "restart", Description: "Restart default services"}, MakeTarget: "restart"},
					{commandDescriptor: commandDescriptor{Name: "health", Description: "Run service health checks"}, MakeTarget: "health"},
					{commandDescriptor: commandDescriptor{Name: "logs", Description: "Tail service logs"}, MakeTarget: "logs"},
					{commandDescriptor: commandDescriptor{Name: "ps", Description: "Show running services"}, MakeTarget: "ps"},
					{commandDescriptor: commandDescriptor{Name: "up-production", Description: "Start production profile services"}, MakeTarget: "up-production"},
					{commandDescriptor: commandDescriptor{Name: "prod-smoke", Description: "Run production smoke tests"}, MakeTarget: "prod-smoke"},
					{commandDescriptor: commandDescriptor{Name: "up-offline", Description: "Start offline mode services"}, MakeTarget: "up-offline"},
					{commandDescriptor: commandDescriptor{Name: "down-offline", Description: "Stop offline mode services"}, MakeTarget: "down-offline"},
					{commandDescriptor: commandDescriptor{Name: "health-offline", Description: "Run offline mode health checks"}, MakeTarget: "health-offline"},
					{commandDescriptor: commandDescriptor{Name: "up-tls", Description: "Start TLS mode services"}, MakeTarget: "up-tls"},
					{commandDescriptor: commandDescriptor{Name: "down-tls", Description: "Stop TLS mode services"}, MakeTarget: "down-tls"},
					{commandDescriptor: commandDescriptor{Name: "tls-health", Description: "Run TLS health checks"}, MakeTarget: "tls-health"},
					{commandDescriptor: commandDescriptor{Name: "helm-validate", Description: "Validate Helm chart"}, MakeTarget: "helm-validate"},
					{commandDescriptor: commandDescriptor{Name: "release-bundle", Description: "Build deployment release bundle"}, NativeRun: runReleaseBundleCommand},
					{commandDescriptor: commandDescriptor{Name: "readiness-evidence", Description: "Generate and verify dated readiness evidence"}, NativeRun: runReadinessEvidenceCommand},
					{commandDescriptor: commandDescriptor{Name: "pilot-closeout-bundle", Description: "Assemble and verify a pilot closeout evidence bundle"}, NativeRun: runPilotCloseoutBundleCommand},
					{commandDescriptor: commandDescriptor{Name: "artifact-retention", Description: "Enforce document artifact retention policy"}, NativeRun: runArtifactRetentionCommand},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "validate",
					Description: "Configuration and policy validation operations",
				},
				Examples: []string{
					"acpctl validate config",
					"acpctl validate lint",
					"acpctl validate detections",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "lint", Description: "Run static validation/lint gate"}, MakeTarget: "lint"},
					{commandDescriptor: commandDescriptor{Name: "config", Description: "Validate demo deployment configuration"}, NativeRun: runValidateConfig},
					{commandDescriptor: commandDescriptor{Name: "detections", Description: "Validate detection rule output"}, NativeRun: runValidateDetections},
					{commandDescriptor: commandDescriptor{Name: "siem-queries", Description: "Validate SIEM query sync"}, NativeRun: runValidateSiemQueries},
					{commandDescriptor: commandDescriptor{Name: "network-contract", Description: "Render network contract artifacts"}, MakeTarget: "network-contract"},
					{commandDescriptor: commandDescriptor{Name: "public-hygiene", Description: "Fail when local-only files are tracked by git"}, NativeRun: runValidatePublicHygiene},
					{commandDescriptor: commandDescriptor{Name: "license", Description: "Validate license policy structure and restricted references"}, NativeRun: runValidateLicense},
					{commandDescriptor: commandDescriptor{Name: "supply-chain", Description: "Run supply-chain policy and digest validation"}, NativeRun: runValidateSupplyChain},
					{commandDescriptor: commandDescriptor{Name: "secrets-audit", Description: "Run deterministic tracked-file secrets audit"}, NativeRun: runSecretsAudit},
					{commandDescriptor: commandDescriptor{Name: "compose-healthchecks", Description: "Validate Docker Compose healthchecks"}, NativeRun: runValidateComposeHealthchecks},
					{commandDescriptor: commandDescriptor{Name: "headers", Description: "Validate Go source file header policy"}, NativeRun: runValidateHeaders},
					{commandDescriptor: commandDescriptor{Name: "env-access", Description: "Fail on direct environment access outside internal/config"}, NativeRun: runValidateEnvAccess},
					{commandDescriptor: commandDescriptor{Name: "security", Description: "Run Make-composed security gate (hygiene, secrets, license, supply chain)"}, MakeTarget: "security-gate"},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "db",
					Description: "Database backup, restore, and inspection operations",
				},
				Examples: []string{
					"acpctl db status",
					"acpctl db backup",
					"acpctl db dr-drill",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "status", Description: "Show database status and statistics"}, MakeTarget: "db-status"},
					{commandDescriptor: commandDescriptor{Name: "backup", Description: "Create database backup"}, NativeRun: runDBBackupCommand},
					{commandDescriptor: commandDescriptor{Name: "restore", Description: "Restore embedded database from backup"}, NativeRun: runDBRestoreCommand},
					{commandDescriptor: commandDescriptor{Name: "shell", Description: "Open database shell"}, MakeTarget: "db-shell"},
					{commandDescriptor: commandDescriptor{Name: "dr-drill", Description: "Run database DR restore drill"}, NativeRun: runDBDRDrill},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "key",
					Description: "Virtual key lifecycle operations",
				},
				Examples: []string{
					"acpctl key gen alice --budget 10.00",
					"acpctl key revoke alice",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "gen", Description: "Generate a standard virtual key"}, NativeRun: runKeyGenCommand},
					{commandDescriptor: commandDescriptor{Name: "revoke", Description: "Revoke a virtual key by alias"}, NativeRun: runKeyRevokeCommand},
					{commandDescriptor: commandDescriptor{Name: "gen-dev", Description: "Generate a developer key"}, NativeRun: runDeveloperKeyGenCommand},
					{commandDescriptor: commandDescriptor{Name: "gen-lead", Description: "Generate a team-lead key"}, NativeRun: runLeadKeyGenCommand},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "host",
					Description: "Host-first deployment and operations",
				},
				Examples: []string{
					"acpctl host preflight",
					"acpctl host check",
					"acpctl host apply",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "preflight", Description: "Validate host readiness"}, MakeTarget: "host-preflight"},
					{commandDescriptor: commandDescriptor{Name: "check", Description: "Run declarative host preflight/check mode"}, MakeTarget: "host-check"},
					{commandDescriptor: commandDescriptor{Name: "apply", Description: "Run declarative host apply/converge"}, MakeTarget: "host-apply"},
					{commandDescriptor: commandDescriptor{Name: "install", Description: "Install systemd service"}, MakeTarget: "host-install"},
					{commandDescriptor: commandDescriptor{Name: "service-status", Description: "Show service status"}, MakeTarget: "host-service-status"},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "demo",
					Description: "Demo scenario, preset, and snapshot operations",
				},
				Examples: []string{
					"acpctl demo scenario SCENARIO=1",
					"acpctl demo preset PRESET=executive-demo",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "scenario", Description: "Run a specific demo scenario"}, MakeTarget: "demo-scenario"},
					{commandDescriptor: commandDescriptor{Name: "all", Description: "Run all demo scenarios"}, MakeTarget: "demo-all"},
					{commandDescriptor: commandDescriptor{Name: "preset", Description: "Run a named demo preset"}, MakeTarget: "demo-preset"},
					{commandDescriptor: commandDescriptor{Name: "snapshot", Description: "Create a named demo snapshot"}, MakeTarget: "demo-snapshot"},
					{commandDescriptor: commandDescriptor{Name: "restore", Description: "Restore a named demo snapshot"}, MakeTarget: "demo-restore"},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "terraform",
					Description: "Terraform provisioning workflow helpers",
				},
				Examples: []string{
					"acpctl terraform init",
					"acpctl terraform plan",
					"acpctl terraform apply",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "init", Description: "Initialize Terraform"}, MakeTarget: "tf-init"},
					{commandDescriptor: commandDescriptor{Name: "plan", Description: "Run Terraform plan"}, MakeTarget: "tf-plan"},
					{commandDescriptor: commandDescriptor{Name: "apply", Description: "Run Terraform apply"}, MakeTarget: "tf-apply"},
					{commandDescriptor: commandDescriptor{Name: "destroy", Description: "Run Terraform destroy"}, MakeTarget: "tf-destroy"},
					{commandDescriptor: commandDescriptor{Name: "fmt", Description: "Format Terraform files"}, MakeTarget: "tf-fmt"},
					{commandDescriptor: commandDescriptor{Name: "validate", Description: "Validate Terraform configuration"}, MakeTarget: "tf-validate"},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "helm",
					Description: "Helm chart validation and smoke tests",
				},
				Examples: []string{
					"acpctl helm validate",
					"acpctl helm smoke",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "validate", Description: "Validate Helm chart"}, NativeRun: runHelmValidateCommand},
					{commandDescriptor: commandDescriptor{Name: "smoke", Description: "Run Helm production smoke tests"}, NativeRun: runHelmSmokeCommand},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name:        "bridge",
					Description: "Execute mapped legacy script implementations directly",
				},
				Examples: []string{
					"acpctl bridge host_preflight --profile production",
					"acpctl bridge host_deploy check --inventory deploy/ansible/inventory/hosts.yml",
					"acpctl bridge release_bundle build --version v1.2.3",
				},
				Subcommands: []subcommandDefinition{
					{commandDescriptor: commandDescriptor{Name: "host_deploy", Description: "Host declarative deployment orchestration"}, ScriptRelativePath: "scripts/libexec/host_deploy_impl.sh", Usage: "acpctl bridge host_deploy [check|apply] [options]"},
					{commandDescriptor: commandDescriptor{Name: "host_install", Description: "Systemd host service installation/management"}, ScriptRelativePath: "scripts/libexec/host_install_impl.sh", Usage: "acpctl bridge host_install [command] [options]"},
					{commandDescriptor: commandDescriptor{Name: "host_preflight", Description: "Host readiness preflight checks"}, ScriptRelativePath: "scripts/libexec/host_preflight_impl.sh", Usage: "acpctl bridge host_preflight [options]"},
					{commandDescriptor: commandDescriptor{Name: "host_upgrade_slots", Description: "Slot-based host upgrade orchestration"}, ScriptRelativePath: "scripts/libexec/host_upgrade_slots_impl.sh", Usage: "acpctl bridge host_upgrade_slots [command] [options]"},
					{commandDescriptor: commandDescriptor{Name: "onboard", Description: "Tool onboarding workflows"}, ScriptRelativePath: "scripts/libexec/onboard_impl.sh", Usage: "acpctl bridge onboard <tool> [options]"},
					{commandDescriptor: commandDescriptor{Name: "prepare_secrets_env", Description: "Host secrets contract refresh/sync"}, ScriptRelativePath: "scripts/libexec/prepare_secrets_env_impl.sh", Usage: "acpctl bridge prepare_secrets_env [options]"},
					{commandDescriptor: commandDescriptor{Name: "prod_smoke_helm", Description: "Helm production smoke workflow"}, ScriptRelativePath: "scripts/libexec/prod_smoke_helm_impl.sh", Usage: "acpctl bridge prod_smoke_helm [options]"},
					{commandDescriptor: commandDescriptor{Name: "prod_smoke_test", Description: "Runtime production smoke checks"}, ScriptRelativePath: "scripts/libexec/prod_smoke_test_impl.sh", Usage: "acpctl bridge prod_smoke_test [options]"},
					{commandDescriptor: commandDescriptor{Name: "release_bundle", Description: "Deployment release bundle build/verify"}, ScriptRelativePath: "scripts/libexec/release_bundle_impl.sh", Usage: "acpctl bridge release_bundle <build|verify> [options]"},
					{commandDescriptor: commandDescriptor{Name: "switch_claude_mode", Description: "Claude mode switching helper"}, ScriptRelativePath: "scripts/libexec/switch_claude_mode_impl.sh", Usage: "acpctl bridge switch_claude_mode <mode|status> [options]"},
				},
			},
			{
				commandDescriptor: commandDescriptor{
					Name: "__complete",
				},
				NativeRun: runHiddenComplete,
				Hidden:    true,
			},
		},
	}
}

func loadCommandState() (compiledCommandState, error) {
	commandStateOnce.Do(func() {
		commandState, commandStateErr = compileCommandState(buildCommandCatalog())
	})
	return commandState, commandStateErr
}

func compileCommandState(catalog commandCatalog) (compiledCommandState, error) {
	state := compiledCommandState{
		Catalog:           catalog,
		Registry:          commandRegistry{RootCommands: make([]commandDescriptor, 0, len(catalog.RootCommands)+1), GroupSubcommands: make(map[string][]commandDescriptor, len(catalog.RootCommands))},
		RootByName:        make(map[string]rootCommandDefinition, len(catalog.RootCommands)),
		SubcommandsByRoot: make(map[string]map[string]subcommandDefinition, len(catalog.RootCommands)),
	}
	for _, root := range catalog.RootCommands {
		if _, exists := state.RootByName[root.Name]; exists {
			return compiledCommandState{}, fmt.Errorf("duplicate root command definition: %s", root.Name)
		}
		state.RootByName[root.Name] = root
		if len(root.Subcommands) > 0 {
			subcommands := make([]commandDescriptor, 0, len(root.Subcommands))
			subByName := make(map[string]subcommandDefinition, len(root.Subcommands))
			for _, subcommand := range root.Subcommands {
				if _, exists := subByName[subcommand.Name]; exists {
					return compiledCommandState{}, fmt.Errorf("duplicate subcommand definition for %s: %s", root.Name, subcommand.Name)
				}
				owners := 0
				if subcommand.NativeRun != nil {
					owners++
				}
				if subcommand.MakeTarget != "" {
					owners++
				}
				if subcommand.ScriptRelativePath != "" {
					owners++
				}
				if root.NativeRun == nil && owners != 1 {
					return compiledCommandState{}, fmt.Errorf("%s %s must have exactly one owner", root.Name, subcommand.Name)
				}
				subByName[subcommand.Name] = subcommand
				subcommands = append(subcommands, subcommand.commandDescriptor)
			}
			state.SubcommandsByRoot[root.Name] = subByName
			if !root.Hidden {
				state.Registry.GroupSubcommands[root.Name] = subcommands
			}
		}
		if !root.Hidden {
			state.Registry.RootCommands = append(state.Registry.RootCommands, root.commandDescriptor)
		}
	}
	state.Registry.RootCommands = append(state.Registry.RootCommands, commandDescriptor{
		Name:        "help",
		Description: "Show this help message",
	})
	return state, nil
}

func commandStartupError() error {
	_, err := loadCommandState()
	return err
}

func buildCommandRegistry() commandRegistry {
	state, err := loadCommandState()
	if err != nil {
		return commandRegistry{}
	}
	return state.Registry
}

func lookupRootCommand(name string) (rootCommandDefinition, error) {
	state, err := loadCommandState()
	if err != nil {
		return rootCommandDefinition{}, err
	}
	command, ok := state.RootByName[name]
	if !ok {
		return rootCommandDefinition{}, &commandLookupError{Kind: "root", Name: name}
	}
	return command, nil
}

func lookupNativeRootCommand(name string) (rootCommandDefinition, error) {
	command, err := lookupRootCommand(name)
	if err != nil {
		return rootCommandDefinition{}, err
	}
	if command.NativeRun == nil {
		return rootCommandDefinition{}, &commandLookupError{Kind: "native-root", Name: name}
	}
	return command, nil
}

func lookupSubcommand(root rootCommandDefinition, name string) (subcommandDefinition, error) {
	state, err := loadCommandState()
	if err != nil {
		return subcommandDefinition{}, err
	}
	subcommands, ok := state.SubcommandsByRoot[root.Name]
	if !ok {
		return subcommandDefinition{}, &commandLookupError{Kind: "subcommand", Root: root.Name, Name: name}
	}
	subcommand, ok := subcommands[name]
	if !ok {
		return subcommandDefinition{}, &commandLookupError{Kind: "subcommand", Root: root.Name, Name: name}
	}
	return subcommand, nil
}

func runDeveloperKeyGenCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runKeyGenCommand(ctx, append([]string{"--role", "developer"}, args...), stdout, stderr)
}

func runLeadKeyGenCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runKeyGenCommand(ctx, append([]string{"--role", "team-lead"}, args...), stdout, stderr)
}
