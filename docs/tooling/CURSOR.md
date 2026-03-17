# Cursor Configuration for AI Control Plane

Cursor can be configured to use a custom OpenAI-compatible base URL. In this repo that means routing Cursor traffic through the LiteLLM gateway for centralized logging, authentication, and cost control.

## Recommended flow

Use the guided wizard with Cursor preselected:

```bash
make onboard-cursor
```

The wizard prints the base URL, LiteLLM virtual key, and model alias you should enter in Cursor.

For a remote or TLS-enabled host, use the typed wizard entrypoint and answer the host/TLS prompts:

```bash
./scripts/acpctl.sh onboard cursor
```

## Settings to copy into Cursor

The wizard prints values in this shape:

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
export OPENAI_BASE_URL="$GATEWAY_URL"
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="openai-gpt5.2"
```

Map those values into Cursor's custom provider settings:
- **Base URL** → `OPENAI_BASE_URL`
- **API Key** → `OPENAI_API_KEY`
- **Model** → `OPENAI_MODEL`

## Verification

1. Run the wizard and keep verification enabled.
2. Paste the printed values into Cursor.
3. Restart Cursor if it cached previous provider settings.
4. Make a test request and confirm health/logging:

```bash
make health
make db-status
```

## Notes

- Cursor version-specific settings may move, but the gateway contract stays the same.
- If users authenticate directly to vendor endpoints outside this configuration, that is a bypass path handled with network controls and detections, not inline gateway enforcement.

## References

- [ACPCTL](ACPCTL.md)
- [Deployment](../DEPLOYMENT.md)
- [LiteLLM Cursor integration](https://docs.litellm.ai/docs/tutorials/cursor_integration)
