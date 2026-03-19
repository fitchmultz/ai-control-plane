// cmd_deploy.go - Typed deploy artifact command tree.
//
// Purpose:
//   - Own the typed deploy artifact command surface.
//
// Responsibilities:
//   - Compose release and readiness subcommands into the deploy tree.
//
// Scope:
//   - Deploy command metadata only.
//
// Usage:
//   - Registered by `command_registry.go` as the `deploy` root command.
//
// Invariants/Assumptions:
//   - Runtime lifecycle orchestration remains on the Make surface.
package main

func deployCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "deploy",
		Summary:     "Typed evidence and artifact workflows",
		Description: "Typed evidence and artifact workflows.",
		Examples: []string{
			"acpctl deploy readiness-evidence run",
			"acpctl deploy release-bundle build",
			"acpctl deploy pilot-closeout-bundle build",
			"acpctl deploy assessor-packet build",
		},
		Children: []*commandSpec{
			releaseBundleCommandSpec(),
			readinessEvidenceCommandSpec(),
			pilotCloseoutBundleCommandSpec(),
			assessorPacketCommandSpec(),
			artifactRetentionCommandSpec(),
		},
	}
}
