// command_specs_legacy.go - Delegated command metadata helpers.
//
// Purpose:
//
//	Provide typed specs for Make-backed and bridge-backed command families that
//	remain intentionally delegated outside the native CLI runtime.
//
// Responsibilities:
//   - Define reusable delegated spec helpers.
//   - Compose delegated deploy, host, demo, terraform, and bridge command trees.
//   - Keep delegated command metadata inside the typed command registry.
//
// Scope:
//   - Command metadata and delegated backend wiring only.
//
// Usage:
//   - Consumed by command_registry.go when composing the root command tree.
//
// Invariants/Assumptions:
//   - Native command parsing/help/completion ownership lives in typed specs elsewhere.
package main

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
	return bridgeLeafSpecWithArgs(name, summary, relativePath)
}

func bridgeLeafSpecWithArgs(name string, summary string, relativePath string, bridgeArgs ...string) *commandSpec {
	return &commandSpec{
		Name:              name,
		Summary:           summary,
		Description:       summary + ".",
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind:               commandBackendBridge,
			BridgeRelativePath: relativePath,
			BridgeArgs:         append([]string(nil), bridgeArgs...),
		},
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
			"acpctl deploy readiness-evidence run",
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
			releaseBundleCommandSpec(),
			readinessEvidenceCommandSpec(),
			pilotCloseoutBundleCommandSpec(),
			artifactRetentionCommandSpec(),
		},
	}
}

func hostCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "host",
		Summary:     "Host-first deployment and operations",
		Description: "Host-first deployment and operations.",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host check --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl host apply --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl host install --service-user acp",
		},
		Children: []*commandSpec{
			bridgeLeafSpec("preflight", "Validate host readiness", "scripts/libexec/host_preflight_impl.sh"),
			bridgeLeafSpecWithArgs("check", "Run declarative host preflight/check mode", "scripts/libexec/host_deploy_impl.sh", "check"),
			bridgeLeafSpecWithArgs("apply", "Run declarative host apply/converge", "scripts/libexec/host_deploy_impl.sh", "apply"),
			bridgeLeafSpecWithArgs("install", "Install systemd service", "scripts/libexec/host_install_impl.sh", "install"),
			bridgeLeafSpecWithArgs("uninstall", "Uninstall systemd service", "scripts/libexec/host_install_impl.sh", "uninstall"),
			bridgeLeafSpecWithArgs("service-status", "Show service status", "scripts/libexec/host_install_impl.sh", "service-status"),
			bridgeLeafSpecWithArgs("service-start", "Start the systemd service", "scripts/libexec/host_install_impl.sh", "service-start"),
			bridgeLeafSpecWithArgs("service-stop", "Stop the systemd service", "scripts/libexec/host_install_impl.sh", "service-stop"),
			bridgeLeafSpecWithArgs("service-restart", "Restart the systemd service", "scripts/libexec/host_install_impl.sh", "service-restart"),
			bridgeLeafSpec("secrets-refresh", "Validate and sync canonical host secrets into the Compose runtime env file", "scripts/libexec/prepare_secrets_env_impl.sh"),
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
			bridgeLeafSpec("onboard", "Tool onboarding workflows", "scripts/libexec/onboard_impl.sh"),
			bridgeLeafSpec("prepare_secrets_env", "Host secrets contract refresh/sync", "scripts/libexec/prepare_secrets_env_impl.sh"),
			bridgeLeafSpec("prod_smoke_helm", "Helm production smoke workflow", "scripts/libexec/prod_smoke_helm_impl.sh"),
			bridgeLeafSpec("prod_smoke_test", "Runtime production smoke checks", "scripts/libexec/prod_smoke_test_impl.sh"),
			bridgeLeafSpec("release_bundle", "Deployment release bundle build/verify", "scripts/libexec/release_bundle_impl.sh"),
		},
	}
}
