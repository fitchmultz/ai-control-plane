# Claude Code Testing Modes

This document explains the supported Claude Code paths for the AI Control Plane gateway.

Boundary: this guide covers Claude Code CLI routing through LiteLLM. It does not cover vendor-hosted web UI governance; use LibreChat for managed browser governance.

## Quick start

### API-key mode

```bash
make onboard-claude
```

Accept the default `api-key` mode in the wizard.

### Subscription-through-gateway mode

```bash
./scripts/acpctl.sh onboard claude
```

Choose `subscription` in the wizard.

The onboarding wizard generates the LiteLLM virtual key, prints the environment variables Claude Code needs, and can run gateway verification before you launch the tool.

## Mode overview

| Mode | Authentication | Cost tracking | Use case |
|------|----------------|---------------|----------|
| API key | LiteLLM virtual key via `ANTHROPIC_API_KEY` | Full gateway enforcement | Lowest-friction governed path |
| Subscription | Claude Code OAuth + LiteLLM virtual key via `ANTHROPIC_CUSTOM_HEADERS` | Gateway enforcement + subscription billing | Subscription-backed upstream while keeping gateway telemetry |

## Configuration contract

### API-key mode

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
export ANTHROPIC_BASE_URL="$GATEWAY_URL"
export ANTHROPIC_API_KEY="sk-..."
export ANTHROPIC_MODEL="claude-haiku-4-5"
```

### Subscription-through-gateway mode

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
export ANTHROPIC_BASE_URL="$GATEWAY_URL"
export ANTHROPIC_CUSTOM_HEADERS="x-litellm-api-key: Bearer sk-..."
export ANTHROPIC_MODEL="claude-haiku-4-5"
```

## How to test

1. Start the gateway:

```bash
make up
make health
```

2. Run onboarding in the mode you want.
3. Export the printed environment values.
4. Launch Claude Code.
5. Make a test request and inspect logs or status output:

```bash
make logs
make db-status
```

## Operational notes

- API-key mode requires `ANTHROPIC_API_KEY` in `demo/.env` for upstream provider access.
- Subscription mode depends on LiteLLM forwarding the Claude OAuth header upstream; keep `forward_client_headers_to_llm_api: true` aligned with the tracked config.
- The wizard does not write `~/.claude/` config files; keep local config ownership under your own config-management workflow.
- Rotate LiteLLM virtual keys if they are exposed.

## References

- [Claude Code quick reference](CLAUDE_CODE_QUICKREF.md)
- [ACPCTL](ACPCTL.md)
- [LiteLLM Claude Code MAX subscription docs](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
