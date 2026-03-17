# OpenCode Configuration for AI Control Plane

OpenCode is a first-class supported CLI tool in this repository's governed tooling model.

## Recommended flow

Use the guided wizard with OpenCode preselected:

```bash
make onboard-opencode
```

The wizard uses the gateway-routed OpenAI-compatible path and prints the environment variables OpenCode needs.

## Printed environment contract

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
export OPENAI_BASE_URL="$GATEWAY_URL"
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="openai-gpt5.2"
```

For a remote or TLS-enabled host, run the typed entrypoint instead and answer the host/TLS prompts:

```bash
./scripts/acpctl.sh onboard opencode
```

## Why this path matters

Routing OpenCode through the gateway gives you:
- central authentication through a LiteLLM virtual key
- request logging and attribution
- budget controls and rate limiting
- model allowlists and shared governance policy

## Verification

The onboarding wizard can verify gateway reachability and authorized model access before you launch OpenCode. After onboarding, you can also confirm runtime health manually:

```bash
make health
make db-status
```

## Notes

- This repo standardizes on the gateway-routed API-key path for OpenCode onboarding.
- Subscription-bypass scenarios belong in governance demos and egress controls, not in the default operator happy path.

## References

- [ACPCTL](ACPCTL.md)
- [Tooling reference links](TOOLING_REFERENCE_LINKS.md)
- [LiteLLM OpenCode integration](https://docs.litellm.ai/docs/tutorials/opencode_integration)
- [OpenCode docs](https://opencode.ai/docs/)
