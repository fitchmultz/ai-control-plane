// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Render stable operator help for onboarding commands without coupling text
//	rendering to execution flow.
//
// Responsibilities:
//   - Print root onboarding help.
//   - Print tool-specific notes.
//   - Share help-token parsing helpers.
//
// Scope:
//   - Operator help text only.
//
// Usage:
//   - Called by Run during help paths.
//
// Invariants/Assumptions:
//   - Help examples must match the canonical CLI entrypoint.
package onboard

import (
	"io"
)

func printMainHelp(w io.Writer) {
	fprintf(w, "Usage: acpctl onboard <tool> [options]\n\n")
	fprintf(w, "Tools:\n")
	fprintf(w, "  codex\n")
	fprintf(w, "  claude\n")
	fprintf(w, "  opencode\n")
	fprintf(w, "  cursor\n")
	fprintf(w, "  copilot\n\n")
	fprintf(w, "Options:\n")
	fprintf(w, "  --mode <mode>          auth mode (tool-dependent)\n")
	fprintf(w, "  --alias <alias>        virtual key alias (default: <tool>-cli)\n")
	fprintf(w, "  --budget <usd>         key budget in USD (default: 10.00)\n")
	fprintf(w, "  --model <model>        model alias override\n")
	fprintf(w, "  --host <host>          gateway host (default: 127.0.0.1)\n")
	fprintf(w, "  --port <port>          gateway port (default: 4000)\n")
	fprintf(w, "  --tls                  use https for base URL\n")
	fprintf(w, "  --verify               run authorized gateway checks\n")
	fprintf(w, "  --write-config         write ACP-managed ~/.codex/config.toml (Codex only)\n")
	fprintf(w, "  --show-key             print full key value\n")
	fprintf(w, "  --help, -h             show help\n\n")
	fprintf(w, "Codex modes:\n")
	fprintf(w, "  subscription           routed through gateway; upstream via ChatGPT provider (default)\n")
	fprintf(w, "  api-key                routed through gateway; upstream via API-key providers\n")
	fprintf(w, "  direct                 no gateway routing; OTEL visibility only\n\n")
	fprintf(w, "Examples:\n")
	fprintf(w, "  acpctl onboard codex --mode subscription --verify\n")
	fprintf(w, "  acpctl onboard codex --mode api-key --write-config\n")
	fprintf(w, "  acpctl onboard claude --mode api-key --verify\n")
}

func printToolHelp(w io.Writer, tool string) {
	switch tool {
	case "codex":
		fprintf(w, "Codex notes:\n")
		fprintf(w, "  - For subscription mode, run `make chatgpt-login` on the gateway host first.\n")
		fprintf(w, "  - Codex uses OPENAI_BASE_URL without /v1.\n")
		fprintf(w, "  - --write-config only updates files already managed by acpctl, or creates a new managed config.\n")
	case "claude":
		fprintf(w, "Claude notes:\n")
		fprintf(w, "  - Exports ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY for gateway routing.\n")
		fprintf(w, "  - Keep mode=api-key unless you have a separate subscription OAuth flow configured.\n")
	case "opencode", "cursor", "copilot":
		fprintf(w, "OpenAI-compatible tool notes:\n")
		fprintf(w, "  - Exports OPENAI_BASE_URL and OPENAI_API_KEY for gateway routing.\n")
	}
}

func isHelpToken(value string) bool {
	switch value {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}
