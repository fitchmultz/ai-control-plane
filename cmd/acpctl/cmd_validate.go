// cmd_validate.go - Validation command tree assembly.
//
// Purpose:
//   - Compose the typed `validate` command tree from focused validation domains.
//
// Responsibilities:
//   - Define the root `validate` command surface.
//   - Keep subcommand ownership grouped by validation concern.
//
// Scope:
//   - Root command assembly only.
//
// Usage:
//   - Invoked through `acpctl validate <subcommand>`.
//
// Invariants/Assumptions:
//   - Subcommands own their own bind/run logic in focused files.
package main

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
			"acpctl validate policy-rules",
		},
		Children: []*commandSpec{
			makeLeafSpec("lint", "Run static validation/lint gate", "lint"),
			validateConfigCommandSpec(),
			validateDetectionsCommandSpec(),
			validateSIEMQueriesCommandSpec(),
			validatePolicyRulesCommandSpec(),
			validatePublicHygieneCommandSpec(),
			validateLicenseCommandSpec(),
			validateSupplyChainCommandSpec(),
			validateSecretsAuditCommandSpec(),
			validateComposeHealthchecksCommandSpec(),
			validateHeadersCommandSpec(),
			validateEnvAccessCommandSpec(),
			makeLeafSpec("security", "Run Make-composed security gate (hygiene, secrets, license, supply chain)", "security-gate"),
		},
	}
}
