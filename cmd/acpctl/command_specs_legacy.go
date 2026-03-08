// command_specs_legacy.go - Transitional command spec wiring for existing handlers.
//
// Purpose:
//
//	Attach existing acpctl command handlers and delegated commands to the new
//	typed command-spec tree while keeping ownership metadata in one place.
//
// Responsibilities:
//   - Build command specs for domains not fully migrated in this refactor.
//   - Adapt legacy native handlers to the typed command backend contract.
//   - Preserve existing Make and bridge delegation paths under the spec layer.
//
// Scope:
//   - Command metadata and backend wiring only.
//
// Usage:
//   - Consumed by command_registry.go when composing the root command tree.
//
// Invariants/Assumptions:
//   - Even transitional commands resolve only through the typed command-spec tree.
package main

import (
	"context"
	"os"
)

type rawArgsOptions struct {
	Args []string
}

func legacyNativeBackend(runner func(context.Context, []string, *os.File, *os.File) int) commandBackend {
	return commandBackend{
		Kind: commandBackendNative,
		NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
			args := append([]string(nil), input.arguments...)
			args = append(args, input.trailing...)
			return rawArgsOptions{Args: args}, nil
		},
		NativeRun: func(ctx context.Context, runCtx commandRunContext, raw any) int {
			return runner(ctx, raw.(rawArgsOptions).Args, runCtx.Stdout, runCtx.Stderr)
		},
	}
}

func makeLeafSpec(name string, summary string, target string) *commandSpec {
	return &commandSpec{
		Name:              name,
		Summary:           summary,
		Description:       summary + ".",
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind:       commandBackendMake,
			MakeTarget: target,
		},
	}
}

func bridgeLeafSpec(name string, summary string, relativePath string) *commandSpec {
	return &commandSpec{
		Name:              name,
		Summary:           summary,
		Description:       summary + ".",
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind:               commandBackendBridge,
			BridgeRelativePath: relativePath,
		},
	}
}

func envCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "env",
		Summary:     "Strict .env access helpers",
		Description: "Strict .env access helpers.",
		Examples: []string{
			"acpctl env get LITELLM_MASTER_KEY",
			"acpctl env get --file demo/.env DATABASE_URL",
		},
		Children: []*commandSpec{
			{
				Name:              "get",
				Summary:           "Read a single env key without shell execution",
				Description:       "Read a single env key without shell execution.",
				AllowTrailingArgs: true,
				Backend:           legacyNativeBackend(runEnvGetCommand),
			},
		},
	}
}

func statusCommandSpec() *commandSpec {
	return &commandSpec{
		Name:              "status",
		Summary:           "Aggregated system health overview",
		Description:       "Aggregated system health overview.",
		AllowTrailingArgs: true,
		Backend:           legacyNativeBackend(runStatusCommand),
	}
}

func healthCommandSpec() *commandSpec {
	return &commandSpec{
		Name:              "health",
		Summary:           "Run service health checks",
		Description:       "Run service health checks.",
		AllowTrailingArgs: true,
		Backend:           legacyNativeBackend(runHealthCommand),
	}
}

func doctorCommandSpec() *commandSpec {
	return &commandSpec{
		Name:              "doctor",
		Summary:           "Environment preflight diagnostics",
		Description:       "Environment preflight diagnostics.",
		AllowTrailingArgs: true,
		Backend:           legacyNativeBackend(runDoctorCommand),
	}
}

func onboardCommandSpec() *commandSpec {
	return &commandSpec{
		Name:              "onboard",
		Summary:           "Configure local tools to route through the gateway",
		Description:       "Configure local tools to route through the gateway.",
		AllowTrailingArgs: true,
		Backend:           legacyNativeBackend(runOnboardCommand),
	}
}

func deployCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "deploy",
		Summary:     "Service lifecycle, release, and deployment operations",
		Description: "Service lifecycle, release, and deployment operations.",
		Examples: []string{
			"acpctl deploy up",
			"acpctl deploy health",
			"acpctl deploy up-production",
			"acpctl deploy up-tls",
		},
		Children: []*commandSpec{
			makeLeafSpec("up", "Start default services", "up"),
			makeLeafSpec("down", "Stop default services", "down"),
			makeLeafSpec("restart", "Restart default services", "restart"),
			makeLeafSpec("health", "Run service health checks", "health"),
			makeLeafSpec("logs", "Tail service logs", "logs"),
			makeLeafSpec("ps", "Show running services", "ps"),
			makeLeafSpec("up-production", "Start production profile services", "up-production"),
			makeLeafSpec("prod-smoke", "Run production smoke tests", "prod-smoke"),
			makeLeafSpec("up-offline", "Start offline mode services", "up-offline"),
			makeLeafSpec("down-offline", "Stop offline mode services", "down-offline"),
			makeLeafSpec("health-offline", "Run offline mode health checks", "health-offline"),
			makeLeafSpec("up-tls", "Start TLS mode services", "up-tls"),
			makeLeafSpec("down-tls", "Stop TLS mode services", "down-tls"),
			makeLeafSpec("tls-health", "Run TLS health checks", "tls-health"),
			makeLeafSpec("helm-validate", "Validate Helm chart", "helm-validate"),
			{Name: "release-bundle", Summary: "Build deployment release bundle", Description: "Build deployment release bundle.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runReleaseBundleCommand)},
			{Name: "readiness-evidence", Summary: "Generate and verify dated readiness evidence", Description: "Generate and verify dated readiness evidence.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runReadinessEvidenceCommand)},
			{Name: "pilot-closeout-bundle", Summary: "Assemble and verify a pilot closeout evidence bundle", Description: "Assemble and verify a pilot closeout evidence bundle.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runPilotCloseoutBundleCommand)},
			{Name: "artifact-retention", Summary: "Enforce document artifact retention policy", Description: "Enforce document artifact retention policy.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runArtifactRetentionCommand)},
		},
	}
}

func validateCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "validate",
		Summary:     "Configuration and policy validation operations",
		Description: "Configuration and policy validation operations.",
		Examples: []string{
			"acpctl validate config",
			"acpctl validate config --production --secrets-env-file /etc/ai-control-plane/secrets.env",
			"acpctl validate lint",
			"acpctl validate detections",
		},
		Children: []*commandSpec{
			makeLeafSpec("lint", "Run static validation/lint gate", "lint"),
			{Name: "config", Summary: "Validate deployment configuration (use --production for host contract checks)", Description: "Validate deployment configuration, including production host contract checks when --production is set.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateConfig)},
			{Name: "detections", Summary: "Validate detection rule output", Description: "Validate detection rule output.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateDetections)},
			{Name: "siem-queries", Summary: "Validate SIEM query sync", Description: "Validate SIEM query sync.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateSiemQueries)},
			makeLeafSpec("network-contract", "Render network contract artifacts", "network-contract"),
			{Name: "public-hygiene", Summary: "Fail when local-only files are tracked by git", Description: "Fail when local-only files are tracked by git.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidatePublicHygiene)},
			{Name: "license", Summary: "Validate license policy structure and restricted references", Description: "Validate license policy structure and restricted references.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateLicense)},
			{Name: "supply-chain", Summary: "Run supply-chain policy and digest validation", Description: "Run supply-chain policy and digest validation.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateSupplyChain)},
			{Name: "secrets-audit", Summary: "Run deterministic tracked-file secrets audit", Description: "Run deterministic tracked-file secrets audit.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runSecretsAudit)},
			{Name: "compose-healthchecks", Summary: "Validate Docker Compose healthchecks", Description: "Validate Docker Compose healthchecks.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateComposeHealthchecks)},
			{Name: "headers", Summary: "Validate Go source file header policy", Description: "Validate Go source file header policy.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateHeaders)},
			{Name: "env-access", Summary: "Fail on direct environment access outside internal/config", Description: "Fail on direct environment access outside internal/config.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runValidateEnvAccess)},
			makeLeafSpec("security", "Run Make-composed security gate (hygiene, secrets, license, supply chain)", "security-gate"),
		},
	}
}

func dbCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "db",
		Summary:     "Database backup, restore, and inspection operations",
		Description: "Database backup, restore, and inspection operations.",
		Examples: []string{
			"acpctl db status",
			"acpctl db backup",
			"acpctl db dr-drill",
		},
		Children: []*commandSpec{
			makeLeafSpec("status", "Show database status and statistics", "db-status"),
			{Name: "backup", Summary: "Create database backup", Description: "Create database backup.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runDBBackupCommand)},
			{Name: "restore", Summary: "Restore embedded database from backup", Description: "Restore embedded database from backup.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runDBRestoreCommand)},
			makeLeafSpec("shell", "Open database shell", "db-shell"),
			{Name: "dr-drill", Summary: "Run database DR restore drill", Description: "Run database DR restore drill.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runDBDRDrill)},
		},
	}
}

func keyCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "key",
		Summary:     "Virtual key lifecycle operations",
		Description: "Virtual key lifecycle operations.",
		Examples: []string{
			"acpctl key gen alice --budget 10.00",
			"acpctl key revoke alice",
		},
		Children: []*commandSpec{
			{Name: "gen", Summary: "Generate a standard virtual key", Description: "Generate a standard virtual key.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runKeyGenCommand)},
			{Name: "revoke", Summary: "Revoke a virtual key by alias", Description: "Revoke a virtual key by alias.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runKeyRevokeCommand)},
			{Name: "gen-dev", Summary: "Generate a developer key", Description: "Generate a developer key.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runDeveloperKeyGenLegacy)},
			{Name: "gen-lead", Summary: "Generate a team-lead key", Description: "Generate a team-lead key.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runLeadKeyGenLegacy)},
		},
	}
}

func runDeveloperKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runKeyGenCommand(ctx, append([]string{"--role", "developer"}, args...), stdout, stderr)
}

func runLeadKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runKeyGenCommand(ctx, append([]string{"--role", "team-lead"}, args...), stdout, stderr)
}

func hostCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "host",
		Summary:     "Host-first deployment and operations",
		Description: "Host-first deployment and operations.",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host check",
			"acpctl host apply",
		},
		Children: []*commandSpec{
			makeLeafSpec("preflight", "Validate host readiness", "host-preflight"),
			makeLeafSpec("check", "Run declarative host preflight/check mode", "host-check"),
			makeLeafSpec("apply", "Run declarative host apply/converge", "host-apply"),
			makeLeafSpec("install", "Install systemd service", "host-install"),
			makeLeafSpec("service-status", "Show service status", "host-service-status"),
		},
	}
}

func demoCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "demo",
		Summary:     "Demo scenario, preset, and snapshot operations",
		Description: "Demo scenario, preset, and snapshot operations.",
		Examples: []string{
			"acpctl demo scenario SCENARIO=1",
			"acpctl demo preset PRESET=executive-demo",
		},
		Children: []*commandSpec{
			makeLeafSpec("scenario", "Run a specific demo scenario", "demo-scenario"),
			makeLeafSpec("all", "Run all demo scenarios", "demo-all"),
			makeLeafSpec("preset", "Run a named demo preset", "demo-preset"),
			makeLeafSpec("snapshot", "Create a named demo snapshot", "demo-snapshot"),
			makeLeafSpec("restore", "Restore a named demo snapshot", "demo-restore"),
		},
	}
}

func terraformCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "terraform",
		Summary:     "Terraform provisioning workflow helpers",
		Description: "Terraform provisioning workflow helpers.",
		Examples: []string{
			"acpctl terraform init",
			"acpctl terraform plan",
			"acpctl terraform apply",
		},
		Children: []*commandSpec{
			makeLeafSpec("init", "Initialize Terraform", "tf-init"),
			makeLeafSpec("plan", "Run Terraform plan", "tf-plan"),
			makeLeafSpec("apply", "Run Terraform apply", "tf-apply"),
			makeLeafSpec("destroy", "Run Terraform destroy", "tf-destroy"),
			makeLeafSpec("fmt", "Format Terraform files", "tf-fmt"),
			makeLeafSpec("validate", "Validate Terraform configuration", "tf-validate"),
		},
	}
}

func helmCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "helm",
		Summary:     "Helm chart validation and smoke tests",
		Description: "Helm chart validation and smoke tests.",
		Examples: []string{
			"acpctl helm validate",
			"acpctl helm smoke",
		},
		Children: []*commandSpec{
			{Name: "validate", Summary: "Validate Helm chart", Description: "Validate Helm chart.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runHelmValidateCommand)},
			{Name: "smoke", Summary: "Run Helm production smoke tests", Description: "Run Helm production smoke tests.", AllowTrailingArgs: true, Backend: legacyNativeBackend(runHelmSmokeCommand)},
		},
	}
}

func bridgeCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "bridge",
		Summary:     "Execute mapped legacy script implementations directly",
		Description: "Execute mapped legacy script implementations directly.",
		Examples: []string{
			"acpctl bridge host_preflight --profile production",
			"acpctl bridge host_deploy check --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl bridge release_bundle build --version v1.2.3",
		},
		Children: []*commandSpec{
			bridgeLeafSpec("host_deploy", "Host declarative deployment orchestration", "scripts/libexec/host_deploy_impl.sh"),
			bridgeLeafSpec("host_install", "Systemd host service installation/management", "scripts/libexec/host_install_impl.sh"),
			bridgeLeafSpec("host_preflight", "Host readiness preflight checks", "scripts/libexec/host_preflight_impl.sh"),
			bridgeLeafSpec("host_upgrade_slots", "Slot-based host upgrade orchestration", "scripts/libexec/host_upgrade_slots_impl.sh"),
			bridgeLeafSpec("onboard", "Tool onboarding workflows", "scripts/libexec/onboard_impl.sh"),
			bridgeLeafSpec("prepare_secrets_env", "Host secrets contract refresh/sync", "scripts/libexec/prepare_secrets_env_impl.sh"),
			bridgeLeafSpec("prod_smoke_helm", "Helm production smoke workflow", "scripts/libexec/prod_smoke_helm_impl.sh"),
			bridgeLeafSpec("prod_smoke_test", "Runtime production smoke checks", "scripts/libexec/prod_smoke_test_impl.sh"),
			bridgeLeafSpec("release_bundle", "Deployment release bundle build/verify", "scripts/libexec/release_bundle_impl.sh"),
			bridgeLeafSpec("switch_claude_mode", "Claude mode switching helper", "scripts/libexec/switch_claude_mode_impl.sh"),
		},
	}
}
