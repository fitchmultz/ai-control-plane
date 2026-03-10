// cmd_bridge_catalog.go - Transitional bridge command catalog.
//
// Purpose:
//   - Keep the remaining bridge-backed implementation entrypoints available
//     without presenting them as a primary public workflow surface.
//
// Responsibilities:
//   - Define the hidden `bridge` command tree for compatibility execution.
//
// Scope:
//   - Bridge command metadata only.
//
// Usage:
//   - Registered by `command_registry.go`; intended for internal compatibility.
//
// Invariants/Assumptions:
//   - `bridge` remains hidden from primary help and completion surfaces.
package main

func bridgeCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "bridge",
		Summary:     "Execute mapped bridge implementations directly",
		Description: "Execute mapped bridge implementations directly.",
		Hidden:      true,
		Examples: []string{
			"acpctl bridge release_bundle build --version v1.2.3",
		},
		Children: []*commandSpec{
			bridgeLeafSpec("host_install", "Systemd host service installation/management", "scripts/libexec/host_install_impl.sh"),
			bridgeLeafSpec("onboard", "Tool onboarding workflows", "scripts/libexec/onboard_impl.sh"),
			bridgeLeafSpec("prepare_secrets_env", "Host secrets contract refresh/sync", "scripts/libexec/prepare_secrets_env_impl.sh"),
			bridgeLeafSpec("prod_smoke_helm", "Helm production smoke workflow", "scripts/libexec/prod_smoke_helm_impl.sh"),
			bridgeLeafSpec("prod_smoke_test", "Runtime production smoke checks", "scripts/libexec/prod_smoke_test_impl.sh"),
			bridgeLeafSpec("release_bundle", "Deployment release bundle build/verify", "scripts/libexec/release_bundle_impl.sh"),
		},
	}
}
