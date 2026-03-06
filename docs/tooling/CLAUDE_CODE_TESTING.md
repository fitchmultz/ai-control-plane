# Claude Code Testing Modes

This document explains how to test Claude Code with LiteLLM Gateway using two different authentication modes.

Boundary: this guide covers Claude Code CLI routing through LiteLLM. It does not cover vendor-hosted web UI governance (ChatGPT web / Claude web); use LibreChat for managed browser governance.

## Quick Start: One-Command Onboarding

The easiest way to configure Claude Code is using the onboarding script:

```bash
# Onboard Claude Code in API key mode (recommended for testing)
make onboard TOOL=claude MODE=api-key

# Onboard in MAX subscription mode (routed through gateway)
make onboard TOOL=claude MODE=subscription

# With connectivity verification
make onboard TOOL=claude MODE=api-key VERIFY=1

# For remote Docker host
make onboard TOOL=claude MODE=api-key HOST=GATEWAY_HOST
```

This will:
1. Generate a virtual key with the specified budget
2. Display the environment variable configuration
3. Optionally verify connectivity to the gateway

`make onboard TOOL=claude ...` prints the required Claude environment values.
Apply those values in your preferred config-management workflow for `~/.claude/`.

## Mode Overview

| Mode | Authentication | Cost Tracking | Use Case |
|------|----------------|---------------|----------|
| **API Key Mode** | LiteLLM virtual key (Bearer token) | Full gateway enforcement | Per-tool tracking with provider API keys |
| **MAX Subscription Mode** | Claude Code OAuth + LiteLLM virtual key (custom header) | Gateway enforcement + subscription billing | Use Claude MAX subscription through gateway |

## Key Differences

### API Key Mode (`settings.local.json`)
- Uses `ANTHROPIC_AUTH_TOKEN` with virtual key
- Gateway enforces budgets, rate limits, and policies
- Requires `ANTHROPIC_API_KEY` in `demo/.env`
- Full governance and control

### MAX Subscription Mode (`settings.max.json`)
- Uses `ANTHROPIC_CUSTOM_HEADERS` with virtual key in `x-litellm-api-key`
- Claude Code signs in with MAX subscription (OAuth)
- Gateway enforces budgets, rate limits, and policies (gateway-side enforcement)
- OAuth token forwarded to Anthropic API (via Authorization header)
- No upstream API key needed - billing handled by Anthropic subscription

**Important:** The gateway **can still enforce** policies (model allowlists, budgets, rate limits) even in subscription mode. The subscription only affects upstream billing, not gateway enforcement.

## Configuration Files

Configuration files are stored in `~/.claude/` (your home directory) to avoid committing secrets to the repository:

### API Key Mode - `~/.claude/settings.local.json`

```json
{
    "env": {
        "ANTHROPIC_BASE_URL": "http://127.0.0.1:4000",
        "ANTHROPIC_AUTH_TOKEN": "<LITELLM_VIRTUAL_KEY>"
    },
    "model": "haiku"
}
```

**Key alias:** `claude-code`
**Key source:** generated via `make onboard TOOL=claude MODE=api-key` or `make key-gen`

### MAX Subscription Mode - `~/.claude/settings.max.json`

```json
{
    "env": {
        "ANTHROPIC_BASE_URL": "http://127.0.0.1:4000",
        "ANTHROPIC_MODEL": "claude-haiku-4-5",
        "ANTHROPIC_CUSTOM_HEADERS": "x-litellm-api-key: Bearer <LITELLM_VIRTUAL_KEY>"
    },
    "model": "haiku"
}
```

**Key alias:** `claude-code-max`
**Key source:** generated via `make onboard TOOL=claude MODE=subscription` or `make key-gen`

## Switching Between Modes

### Using the Mode Switcher Script

```bash
# Check current mode
./scripts/acpctl.sh bridge switch_claude_mode status

# Switch to API key mode
./scripts/acpctl.sh bridge switch_claude_mode api-key

# Switch to MAX subscription mode
./scripts/acpctl.sh bridge switch_claude_mode max
```

### Manual Mode Switching

#### To use API Key Mode:

1. Ensure active config is `settings.local.json`:
   ```bash
   ls -la ~/.claude/settings.local.json
   ```

2. If not, activate it:
   ```bash
   cp ~/.claude/settings.local.json ~/.claude/settings.json
   ```

3. Start Claude Code:
   ```bash
   claude
   ```

4. Select "API Key" or "Enter API key manually"

### To use MAX Subscription Mode:

1. Activate MAX subscription config:
   ```bash
   cp ~/.claude/settings.max.json ~/.claude/settings.json
   ```

2. Start Claude Code:
   ```bash
   claude
   ```

3. Select "Claude account with subscription" (Pro, Max, Team, or Enterprise)

4. Complete OAuth flow in browser (click "Authorize")

## How It Works

### API Key Mode Flow

```
Claude Code → LiteLLM Gateway (validates virtual key)
              ↓
              Enforces budgets, rate limits, policies
              ↓
              Calls Anthropic API with API key from .env
              ↓
              Returns response to Claude Code
```

### MAX Subscription Mode Flow

```
Claude Code → LiteLLM Gateway (validates x-litellm-api-key header)
              ↓
              Enforces budgets, rate limits, policies
              ↓
              Forwards OAuth token to Anthropic API (via Authorization header)
              ↓
              Anthropic validates subscription, processes request
              ↓
              Returns response to Claude Code via gateway
```

**Critical setting:** `forward_client_headers_to_llm_api: true` in `demo/config/litellm.yaml`

This setting enables LiteLLM to forward the OAuth token from Claude Code to Anthropic while still handling gateway authentication, logging, and enforcement.

## Route-Based Governance Model

This demonstration showcases the enterprise AI Control Plane's ability to handle two distinct governance tracks:

| Aspect | API Key Mode | MAX Subscription Mode |
|--------|--------------|----------------------|
| **Authentication** | Virtual key via `ANTHROPIC_AUTH_TOKEN` | OAuth token + virtual key via header |
| **Gateway Enforcement** | Full enforcement (budgets, rate limits, model allowlist) | Full enforcement (budgets, rate limits, model allowlist) |
| **Billing** | Via gateway (provider API key) | Via Anthropic subscription |
| **Use Case** | Per-tool/team tracking with shared API key | Individual subscription through gateway |
| **Key Alias** | `claude-code` | `claude-code-max` |

### Management Demo Script

Use this script to demonstrate the route-based governance model to management:

```bash
# 1. Start the gateway
cd /path/to/ai-control-plane
make up

# 2. Verify services are healthy
make health

# 3. Show current mode (starts in API Key mode)
./scripts/acpctl.sh bridge switch_claude_mode status

# 4. Demo API Key Mode (Full Enforcement)
echo "=== API Key Mode: Full Gateway Enforcement ==="
./scripts/acpctl.sh bridge switch_claude_mode api-key
claude
# In Claude: /model haiku, then "Hello from API key mode"
# Press Ctrl+D to exit

# 5. Check gateway logs - show key alias "claude-code"
make logs

# 6. Show database usage for claude-code key
make db-status

# 7. Switch to MAX Subscription Mode (Gateway Enforcement + Subscription Billing)
echo "=== MAX Subscription Mode: OAuth + Gateway Enforcement ==="
./scripts/acpctl.sh bridge switch_claude_mode max
claude
# In Claude: Select "Claude account with subscription", complete OAuth
# Then: /model haiku, then "Hello from MAX subscription mode"
# Press Ctrl+D to exit

# 8. Check gateway logs - show key alias "claude-code-max"
make logs

# 9. Show database usage for both keys
make db-status

# 10. Security verification - ensure no OAuth tokens in logs
docker compose -f demo/docker-compose.yml logs litellm | grep -i "authorization" || echo "PASS: No Authorization headers logged"
```

### Key Takeaways for Management

1. **Unified Visibility**: Both modes log through the gateway, providing centralized usage tracking
2. **Consistent Enforcement**: Gateway enforces policies (budgets, rate limits, allowlists) regardless of upstream billing method
3. **Flexible Billing**: API key mode uses provider API keys; subscription mode uses individual Anthropic subscriptions
4. **Security**: OAuth tokens are forwarded but never logged
5. **Cost Control**: Per-key budgets prevent runaway spending in both modes

## Virtual Keys in Database

View all virtual keys:

```bash
cd demo && source .env && curl -s -X GET "http://127.0.0.1:4000/key/list" \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" | python3 -m json.tool
```

Current keys:
- `claude-code`: API key mode testing
- `claude-code-max`: MAX subscription mode testing

## Testing Checklist

### API Key Mode
- [ ] Start Claude Code with API key mode
- [ ] Verify requests appear in LiteLLM logs with `claude-code` key alias
- [ ] Test all models (sonnet, haiku, opus)
- [ ] Check budget enforcement by exceeding limit
- [ ] Verify rate limiting with high request rate

### MAX Subscription Mode
- [ ] Start Claude Code with MAX subscription mode
- [ ] Complete OAuth flow successfully
- [ ] Verify requests appear in LiteLLM logs with `claude-code-max` key alias
- [ ] Test all models (sonnet, haiku, opus)
- [ ] Check that OAuth token is not logged (security)
- [ ] Verify subscription billing through Anthropic console
- [ ] Verify budget enforcement still works (gateway-side)

## Monitoring

### View Real-Time Logs

```bash
# Follow all service logs
make logs

# Follow only LiteLLM logs
docker compose -f demo/docker-compose.yml logs litellm -f

# Check recent errors
docker compose -f demo/docker-compose.yml logs litellm --tail 100 | grep ERROR
```

### Check Budget Usage

```bash
cd demo && source .env && curl -s -X GET "http://127.0.0.1:4000/budget/usage" \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" | python3 -m json.tool
```

### View Virtual Key Details

```bash
cd demo && source .env && curl -s -X GET "http://127.0.0.1:4000/key/info" \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" | python3 -m json.tool
```

### Access LiteLLM WebUI

Local mode: http://127.0.0.1:4000/ui
- Username: `admin`
- Password: `LITELLM_MASTER_KEY` value from `demo/.env`

## Troubleshooting

### API Key Mode Issues

**Problem:** 401 Unauthorized from LiteLLM
- **Solution:** Check virtual key is valid in database
- **Command:** `make db-status`

**Problem:** Model not found
- **Solution:** Verify model name matches config in `demo/config/litellm.yaml`

### MAX Subscription Mode Issues

**Problem:** OAuth token not being forwarded
- **Solution:** Ensure `forward_client_headers_to_llm_api: true` in litellm.yaml
- **Verification:** Check `demo/config/litellm.yaml` line 32

**Problem:** LiteLLM authentication failing
- **Solution:** Verify `x-litellm-api-key` header is set correctly
- **Check:** Look at `~/.claude/settings.max.json`

**Problem:** Anthropic authentication errors
- **Solution:** Complete OAuth flow in browser, ensure active subscription
- **Check:** Anthropic console at https://console.anthropic.com

## Security Verification Checklist

Before using subscription mode in production, verify:

- [ ] OAuth tokens are NOT present in gateway logs
- [ ] Authorization headers are redacted from any trace output
- [ ] Database only contains metadata (key alias, model, cost), not tokens
- [ ] LiteLLM `redact_user_api_key_info: true` is configured

**Verification commands:**

```bash
# Run automated secrets audit (recommended)
make secrets-audit

# Or manually check live container logs
# This should return NO results (grep finds nothing)
docker compose -f demo/docker-compose.yml logs litellm | grep -i "authorization"

# Check for bearer tokens in logs
docker compose -f demo/docker-compose.yml logs litellm | grep -i "bearer" | grep -v "x-litellm-api-key" || echo "PASS: No OAuth bearer tokens found"
```

**CI/CD Integration:**
The secrets audit is automatically run during `make lint` and `make ci`. If leaks are detected, the CI gate will fail. This ensures no credentials are accidentally committed or shared.

## Security Notes

### OAuth Token Safety

- **NEVER log Authorization headers** - OAuth tokens may appear in logs
- **Redact tokens** when copying log excerpts for documentation
- **LiteLLM does NOT log OAuth tokens by default** (safe behavior)
- **Review logs before sharing** if subscription mode is used

### Secret Management

- **Never commit Claude settings JSON** (or virtual keys) anywhere in the repo
- **Rotate keys if exposed** - Delete and regenerate
- **Use different keys for different modes** - Isolate tracking

## References

- [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
- [LiteLLM ChatGPT Provider Documentation](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex Tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)
- [Project README](../README.md)
- [Demo Environment README](../../demo/README.md)
- [Database Documentation](../DATABASE.md)
- [Deployment Guide](../DEPLOYMENT.md)
