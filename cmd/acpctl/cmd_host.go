// cmd_host.go - Host command tree assembly.
//
// Purpose:
//   - Own the supported host-first workflow surface.
//
// Responsibilities:
//   - Define the typed `host` command metadata.
//   - Keep supported host workflows grouped under one root command.
//
// Scope:
//   - Host command metadata only.
//
// Usage:
//   - Registered by `command_registry.go` as the `host` root command.
//
// Invariants/Assumptions:
//   - Native host command coverage can expand incrementally without changing
//     the root command structure.
package main

func hostCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "host",
		Summary:     "Host-first deployment and operations",
		Description: "Host-first deployment and operations.",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host check --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl host apply --inventory deploy/ansible/inventory/hosts.yml",
			"acpctl host install --service-user acp --backup-retention-keep 14",
			"acpctl cert renew-auto --env-file /etc/ai-control-plane/secrets.env",
		},
		Children: []*commandSpec{
			hostPreflightCommandSpec(),
			hostDeployCommandSpec("check"),
			hostDeployCommandSpec("apply"),
			bridgeLeafSpecWithArgs("install", "Install systemd service and automated backup timer", "scripts/libexec/host_install_impl.sh", "install"),
			bridgeLeafSpecWithArgs("uninstall", "Uninstall systemd service and automated backup timer", "scripts/libexec/host_install_impl.sh", "uninstall"),
			bridgeLeafSpecWithArgs("service-status", "Show service and backup timer status", "scripts/libexec/host_install_impl.sh", "service-status"),
			bridgeLeafSpecWithArgs("service-start", "Start the systemd service", "scripts/libexec/host_install_impl.sh", "service-start"),
			bridgeLeafSpecWithArgs("service-stop", "Stop the systemd service", "scripts/libexec/host_install_impl.sh", "service-stop"),
			bridgeLeafSpecWithArgs("service-restart", "Restart the systemd service", "scripts/libexec/host_install_impl.sh", "service-restart"),
		},
	}
}
