# Codex CLI (OpenAI) - AI Control Plane Gateway Setup

This guide is the canonical Codex walkthrough for this repo.

## Recommended flow

1. Start services:

```bash
make up
```

2. If you want ChatGPT-subscription-backed upstream routing, complete device login on the gateway host:

```bash
make chatgpt-login
```

If your org disables remote device login, use the fallback cache-copy flow:

```bash
codex login
make chatgpt-auth-copy
make chatgpt-login
```

3. Launch the onboarding wizard with Codex preselected:

```bash
make onboard-codex
```

The wizard asks you to choose one of these modes:
- `subscription` — default; routes through the gateway and uses ChatGPT subscription upstream
- `api-key` — routes through the gateway and uses provider API keys upstream
- `direct` — bypasses the gateway and emits OTEL-only visibility settings

4. Apply the printed settings and launch Codex.

## Routed Codex environment contract

For `subscription` or `api-key` mode the wizard prints:

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
export OPENAI_BASE_URL="$GATEWAY_URL"
export OPENAI_API_KEY="sk-..."    # LiteLLM virtual key, not a provider raw key
export OPENAI_MODEL="chatgpt-gpt5.3-codex" # or another selected alias
```

Important: `OPENAI_BASE_URL` must not include `/v1`.

## Direct Codex visibility mode

When you choose `direct`, onboarding prints OTEL settings instead of gateway API-key exports:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://127.0.0.1:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"
export OTEL_SERVICE_NAME="codex-cli"
```

Use this mode only when you intentionally want visibility without gateway enforcement.

## Operational notes

- `make onboard-codex` can optionally write an ACP-managed `~/.codex/config.toml` when the target file is safe to manage.
- When that ACP-managed Codex config is written, onboarding now lints the file as real TOML, checks the expected provider contract, and verifies private file permissions before reporting success.
- The wizard prints a verification section with `[OK]`, `[FAIL]`, and `[SKIP]` so you can see whether local lint, gateway reachability, and authorized model access actually passed.
- `subscription` mode still expects you to authenticate Codex with the generated LiteLLM virtual key; the upstream ChatGPT subscription remains on the gateway host.
- If onboarding verification fails, run `make health` first and inspect `make logs-litellm`.

## Troubleshooting

- If `make chatgpt-login` says the model alias is missing, restart services after config changes: `make restart`
- If key generation fails, verify the master key with `./scripts/acpctl.sh env get LITELLM_MASTER_KEY`
- If authorized checks fail, run `make health` and inspect `make logs-litellm`

## References

- [ACPCTL](ACPCTL.md)
- [OTEL Setup](../observability/OTEL_SETUP.md)
- [LiteLLM ChatGPT provider docs](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)
- [OpenAI Codex auth docs](https://developers.openai.com/codex/auth)
