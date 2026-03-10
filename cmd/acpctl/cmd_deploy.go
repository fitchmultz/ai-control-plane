// cmd_deploy.go - Delegated deploy command tree.
//
// Purpose:
//   - Own the delegated deploy command surface for runtime lifecycle and
//     release-oriented workflows.
//
// Responsibilities:
//   - Define Make-backed deploy operations.
//   - Compose release and readiness subcommands into the deploy tree.
//
// Scope:
//   - Deploy command metadata only.
//
// Usage:
//   - Registered by `command_registry.go` as the `deploy` root command.
//
// Invariants/Assumptions:
//   - Runtime orchestration remains Make-backed in this refactor wave.
package main

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
