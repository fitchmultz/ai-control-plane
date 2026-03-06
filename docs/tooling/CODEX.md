# Codex CLI (OpenAI) - AI Control Plane Gateway Setup

This guide is the canonical Codex walkthrough for this repo.

## Default Path (Recommended): ChatGPT Subscription Through LiteLLM

This is the harder path and the default demo path:
- Codex routes through LiteLLM gateway
- Gateway enforces policy/budgets/rate limits
- Upstream billing is ChatGPT subscription (not OpenAI API key billing)

### Zero-to-One Checklist

1. Start services:

```bash
make up
```

1. Trigger ChatGPT device login on the gateway host:

```bash
make chatgpt-login
```

This command also switches LiteLLM to the ChatGPT overlay config for Codex subscription routing.

If your org disables remote device login, use fallback cache copy:

```bash
codex login
make chatgpt-auth-copy
make chatgpt-login
```

`make chatgpt-auth-copy` writes normalized credentials to `demo/auth/chatgpt/auth.json`.
The ChatGPT overlay mounts that path into LiteLLM.

1. After completing browser/device login, verify gateway health:

```bash
make health
```

1. Onboard Codex (subscription mode is default for this shortcut):

```bash
make onboard-codex VERIFY=1
```

1. Export the variables printed by onboarding, then launch Codex:

```bash
codex
```

1. In Codex, use API-key auth with the LiteLLM virtual key shown by onboarding.

## What `make chatgpt-login` Actually Does

- Calls LiteLLM `/v1/models` with authorization (master key)
- Sends a starter `/v1/responses` request to `chatgpt-gpt5.3-codex`
- Triggers ChatGPT OAuth device flow for the LiteLLM ChatGPT provider when needed
- Activates the ChatGPT compose overlay (`demo/docker-compose.chatgpt.yml`) for LiteLLM.
- If this LiteLLM build blocks startup while waiting for OAuth, it prints the
  device URL/code directly from `demo-litellm-1` logs.
- For restricted orgs, `make chatgpt-auth-copy` converts local `~/.codex/auth.json`
  into LiteLLM ChatGPT auth format and copies it to container auth cache.

If OAuth is required, complete the browser/device prompt and rerun `make chatgpt-login`.

## Codex Environment Contract

For routed modes (`subscription` or `api-key`):

```bash
export OPENAI_BASE_URL="http://127.0.0.1:4000"
export OPENAI_API_KEY="sk-..."    # LiteLLM virtual key, not OpenAI API key
export OPENAI_MODEL="chatgpt-gpt5.3-codex"
```

Important: `OPENAI_BASE_URL` must not include `/v1`.

## Other Modes

### API-Key Upstream (easier setup)

```bash
make onboard-codex MODE=api-key VERIFY=1
```

### Direct Subscription (bypass gateway enforcement, OTEL visibility only)

```bash
make up-production
make onboard-codex MODE=direct VERIFY=1
```

## Troubleshooting

- If `make chatgpt-login` says model alias missing: restart after config changes:
  - `make restart`
- If onboarding key generation fails: ensure `demo/.env` has `LITELLM_MASTER_KEY`
- If verify fails on authorized checks: run `make health` and inspect `make logs-litellm`

## References

- <https://docs.litellm.ai/docs/providers/chatgpt>
- <https://docs.litellm.ai/docs/tutorials/openai_codex>
- <https://developers.openai.com/codex/auth#fallback-authenticate-locally-and-copy-your-auth-cache>
- `docs/observability/OTEL_SETUP.md`
