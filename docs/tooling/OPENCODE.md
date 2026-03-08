# OpenCode Configuration for AI Control Plane

## Overview

OpenCode is an AI-powered coding assistant that can be configured to work with the LiteLLM gateway for centralized logging, authentication, and cost control. This document covers two deployment paths:

1. **Gateway-Routed Mode (Recommended)**: OpenCode sends all requests through the LiteLLM gateway
2. **Subscription Mode (Bypass Demo)**: OpenCode authenticates directly via Codex subscription

In enterprise contexts, the subscription mode represents a **bypass vector** that demonstrates the need for endpoint enforcement and egress controls.

OpenCode is a first-class supported CLI tool in this repository's governed-tooling model.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Path A: OpenCode Through LiteLLM (Recommended)](#path-a-opencode-through-litellm-recommended)
3. [Path B: OpenCode with Codex Subscription (Bypass Demo)](#path-b-opencode-with-codex-subscription-bypass-demo)
4. [OAuth Token Safety](#oauth-token-safety)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)
7. [Related Documentation](#related-documentation)

---

## Prerequisites

### Required Components

| Component | Purpose |
|-----------|---------|
| OpenCode CLI | AI coding assistant tool |
| LiteLLM Gateway | Central API gateway for routing and logging |
| PostgreSQL | Persistent storage for audit logs |

### Quick Setup with Onboarding Script

The easiest way to configure OpenCode is using the onboarding script:

```bash
# Gateway mode (recommended)
make onboard TOOL=opencode MODE=gateway

# With verification
make onboard TOOL=opencode MODE=gateway VERIFY=1
```

This will generate a LiteLLM virtual key and display the environment variables you need to set.

### Manual Key Generation (Alternative)

If you prefer to generate keys manually:

```bash
# Using the Makefile
make key-gen ALIAS=opencode-user BUDGET=25.00
```

Store the returned key securely. You will need it for configuration.

---

## Path A: OpenCode Through LiteLLM (Recommended)

This configuration routes all OpenCode requests through the LiteLLM gateway, enabling central authentication, logging, and policy enforcement.

### LiteLLM Integration

LiteLLM provides documented integration with OpenCode. The quickstart guide covers the complete setup: <https://docs.litellm.ai/docs/tutorials/opencode_integration>

### Configuration Steps

#### Step 1: Set Environment Variables

Configure OpenCode to point to the LiteLLM gateway:

**Local Mode (Single Machine):**
```bash
export OPENAI_BASE_URL="http://127.0.0.1:4000"
export OPENAI_API_KEY="<YOUR_LITELLM_VIRTUAL_KEY>"
```

**Remote Mode (Gateway Host):**
```bash
export OPENAI_BASE_URL="https://GATEWAY_HOST"
export OPENAI_API_KEY="<YOUR_LITELLM_VIRTUAL_KEY>"
```

Replace `<YOUR_LITELLM_VIRTUAL_KEY>` with the key generated in the prerequisites step.

#### Step 2: Configure Model Selection

OpenCode will use the model aliases configured in `demo/config/litellm.yaml`. The current configuration includes:

| Alias | Model |
|-------|-------|
| `openai-gpt5.2` | `openai/gpt-5.2` |
| `claude-sonnet-4-5` | `anthropic/claude-sonnet-4-5` |

For API-key mode, ensure the corresponding provider API key is set in `demo/.env`.

#### Step 3: Start OpenCode

Launch OpenCode with the gateway configuration:

```bash
opencode
```

OpenCode will now route all requests through the LiteLLM gateway.

### Benefits of Gateway Mode

- **Central Authentication**: Single virtual key per user/service
- **Request Logging**: Usage/cost logs stored in LiteLLM-managed `"LiteLLM_SpendLogs"` (metadata-only)
- **Cost Controls**: Per-key budget limits and spend tracking
- **Policy Enforcement**: Model allowlist and rate limiting at the gateway
- **Attribution**: Clear audit trail linking requests to specific keys

---

## Path B: OpenCode with Codex Subscription (Bypass Demo)

OpenCode can authenticate via a plugin that uses Codex subscription (OAuth) instead of API credits. This represents a realistic bypass scenario in enterprise environments.

### OpenCode Codex Auth Plugin

The OpenCode Codex authentication plugin is documented at: <https://numman-ali.github.io/opencode-openai-codex-auth/>

### Why This Is a Bypass Vector

When OpenCode authenticates directly via Codex subscription:

- **Requests bypass the LiteLLM gateway** entirely
- **No central logging** of AI interactions
- **No cost controls** or budget enforcement
- **No policy enforcement** for model access
- **Audit trail** requires vendor Compliance API (enterprise tier only)

### Enterprise Governance Requirements

To control this bypass vector, enterprises must enforce:

| Control | Purpose |
|---------|---------|
| **Allowable Auth Methods** | Force API-key mode via managed configuration |
| **Allowable Endpoints** | Block direct vendor endpoints at egress |
| **Endpoint Posture** | Require gateway routing for compliance |
| **Tool Config Management** | Prevent local configuration overrides |

### Demo Scenario: Bypass Detection

In the AWS lab environment, this scenario demonstrates:

1. **Baseline**: Configure OpenCode through the gateway (Path A)
2. **Bypass Attempt**: Reconfigure OpenCode to use Codex subscription
3. **Detection**: SWG policies or egress controls detect/block direct vendor access
4. **Remediation**: Restore gateway configuration

### Local Lab Limitations

On a home network, you cannot enforce corporate egress controls. The local lab can only:

- Demonstrate the configuration difference between modes
- Show what the bypass looks like in tool settings
- Document the detection requirements for enterprise environments

For full enforcement validation, use the AWS lab deployment.

---

## OAuth Token Safety

When using subscription mode (Path B), OpenCode forwards OAuth tokens to upstream providers.

### Critical Security Requirements

- **NEVER log Authorization headers** in LiteLLM or any reverse proxy
- **Redact tokens** before copying log excerpts for documentation
- **Review logs** before sharing to ensure no tokens are present
- **Strip headers** from stored traffic for compliance

### Log File Security

OAuth tokens may appear in logs when using subscription mode. Before sharing logs:

```bash
# Run automated secrets audit
make secrets-audit

# Or manually check for tokens
grep -i "authorization" demo/logs/litellm.log
grep -i "bearer" demo/logs/litellm.log
```

The secrets audit (`make secrets-audit`) is the recommended approach as it:
- Scans all mounted logs, backups, and build contexts
- Detects multiple leak patterns (Authorization headers, Bearer tokens, API keys, JWTs)
- Never prints raw secrets in output
- Fails fast on first confirmed leak
- Is automatically run during `make lint` and `make ci`

If leaks are found, redact the log file:

```bash
# Redact authorization headers (backup first!)
sed -i.bak 's/Authorization: Bearer [^ ]*/Authorization: Bearer [REDACTED]/g' demo/logs/litellm.log

# Re-run audit to verify
make secrets-audit
```

---

## Verification

### Verify Gateway Routing

After configuring OpenCode in gateway mode, verify requests appear in audit logs:

```bash
# Check database status
make db-status

# Query recent audit log entries
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c \
  "SELECT
          s.request_id,
          s.model,
          COALESCE(v.key_alias, 'unknown') AS key_alias,
          s.spend,
          s.status,
          s.\"startTime\" AS start_time
   FROM \"LiteLLM_SpendLogs\" s
   LEFT JOIN \"LiteLLM_VerificationToken\" v
     ON s.api_key = v.token
   ORDER BY s.\"startTime\" DESC
   LIMIT 10;"
```

Expected output: Recent requests from OpenCode with your key_alias.

### Verify Attribution

Confirm requests are attributed correctly:

```bash
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c \
  "SELECT
          v.key_alias,
          COUNT(*) AS request_count,
          SUM(s.spend) AS total_spend
   FROM \"LiteLLM_SpendLogs\" s
   JOIN \"LiteLLM_VerificationToken\" v
     ON s.api_key = v.token
   WHERE v.key_alias = 'opencode-user'
   GROUP BY v.key_alias;"
```

### Run Detection Rules

Verify security monitoring is working:

```bash
make detection
```

---

## Troubleshooting

### Connection Errors

**Symptom**: OpenCode cannot connect to the gateway

**Diagnosis**:
```bash
# Test gateway connectivity
curl http://127.0.0.1:4000/health  # Local mode
curl https://GATEWAY_HOST/health  # Remote mode
```

**Solutions**:
- Verify LiteLLM is running: `make ps`
- Check firewall rules allow port 443 for the TLS gateway
- Confirm base URL matches your deployment mode

### Authentication Failures

**Symptom**: 401 Unauthorized errors

**Diagnosis**:
```bash
# Verify virtual key exists
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c \
  "SELECT key_alias, max_budget FROM \"LiteLLM_VerificationToken\" WHERE key_alias = 'opencode-user';"
```

**Solutions**:
- Confirm `OPENAI_API_KEY` matches your LiteLLM virtual key
- Check key has not exceeded budget: `make db-status`
- Regenerate key if needed: `make key-gen ALIAS=opencode-user BUDGET=25.00`

### Model Not Available

**Symptom**: OpenCode reports model not found

**Diagnosis**:
```bash
# Check configured models
curl -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  http://127.0.0.1:4000/v1/models
```

**Solutions**:
- Verify model alias exists in `demo/config/litellm.yaml`
- Check provider API key is set in `demo/.env`
- Restart LiteLLM after configuration changes: `make restart`

### No Logs Appearing

**Symptom**: Requests succeed but no audit log entries

**Diagnosis**:
```bash
# Check database connectivity
make db-status

# Check LiteLLM logs for errors
docker compose logs --tail=50 litellm
```

**Solutions**:
- Verify `DATABASE_URL` is correctly set in `demo/.env`
- Restart services: `make down && make up`
- Check `forward_client_headers_to_llm_api: true` in litellm.yaml

---

## Related Documentation

| Document | Purpose |
|----------|---------|
| [DEPLOYMENT.md](../DEPLOYMENT.md) | Network setup and deployment modes |
| [DATABASE.md](../DATABASE.md) | Audit log verification and database queries |
| [DETECTION.md](../security/DETECTION.md) | Security monitoring and detection rules |
| [NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md](../demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md) | Bypass prevention and egress controls |
| [Local Demo Implementation Plan](../LOCAL_DEMO_PLAN.md) | End-to-end demo flow and environment assumptions |
| [TOOLING_REFERENCE_LINKS.md](TOOLING_REFERENCE_LINKS.md) | Upstream authoritative links for OpenCode, Claude Code, Codex, Cursor/Copilot, and LiteLLM |
| [LiteLLM OpenCode Integration](https://docs.litellm.ai/docs/tutorials/opencode_integration) | Official LiteLLM OpenCode quickstart |
| [OpenCode Docs](https://opencode.ai/docs/) | Official OpenCode documentation |
| [OpenCode GitHub](https://github.com/anomalyco/opencode) | OpenCode source repository |
| [OpenCode Codex Auth Plugin](https://numman-ali.github.io/opencode-openai-codex-auth/) | OpenCode subscription authentication documentation |
