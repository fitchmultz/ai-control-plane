// command_registry.go - Canonical acpctl command tree assembly.
//
// Purpose:
//
//	Assemble the root acpctl command tree from per-domain typed command specs.
//
// Responsibilities:
//   - Compose domain command specs into one root command tree.
//   - Keep command ownership and user-facing ordering centralized.
//
// Scope:
//   - Root composition only; command parsing/help/completion lives in command_spec.go.
//
// Usage:
//   - Consumed by the command-spec compiler during CLI startup.
//
// Invariants/Assumptions:
//   - Each domain file owns its own command metadata and typed binding logic.
package main

func acpctlCommandSpec() *commandSpec {
	return rootCommandSpec(
		ciCommandSpec(),
		envCommandSpec(),
		chargebackCommandSpec(),
		opsCommandSpec(),
		statusCommandSpec(),
		healthCommandSpec(),
		doctorCommandSpec(),
		benchmarkCommandSpec(),
		smokeCommandSpec(),
		completionCommandSpec(),
		onboardCommandSpec(),
		deployCommandSpec(),
		validateCommandSpec(),
		dbCommandSpec(),
		keyCommandSpec(),
		upgradeCommandSpec(),
		hostCommandSpec(),
		hiddenGenerateDocsCommandSpec(),
		hiddenCompleteCommandSpec(),
	)
}
