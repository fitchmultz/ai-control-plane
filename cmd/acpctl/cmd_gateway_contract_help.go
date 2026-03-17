// cmd_gateway_contract_help.go - Shared gateway contract help text.
//
// Purpose:
//   - Centralize the operator-facing gateway and secret contract used by typed commands.
//
// Responsibilities:
//   - Provide reusable help sections for commands that depend on gateway access.
//   - Keep the documented URL and secret resolution order consistent across commands.
//   - Reduce duplicated help text drift in the command tree.
//
// Scope:
//   - Help metadata only.
//
// Usage:
//   - Called by command specs that talk to the gateway or require the master key.
//
// Invariants/Assumptions:
//   - The gateway contract matches `internal/config/gateway.go`.
//   - Secret access examples stay typed and non-executing.
package main

func gatewayContractHelpSection() commandHelpSection {
	return commandHelpSection{
		Title: "Gateway contract",
		Lines: []string{
			"Default URL resolution: ACP_GATEWAY_URL or GATEWAY_URL, then GATEWAY_HOST + LITELLM_PORT + ACP_GATEWAY_SCHEME/ACP_GATEWAY_TLS, then http://127.0.0.1:4000.",
			"Secret sourcing: LITELLM_MASTER_KEY in your shell or repo-local demo/.env, read via ./scripts/acpctl.sh env get LITELLM_MASTER_KEY.",
			"Database mode: ACP_DATABASE_MODE.",
		},
	}
}
