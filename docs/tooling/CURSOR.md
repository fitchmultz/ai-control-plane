# Cursor Configuration for AI Control Plane

## Overview

Cursor is an AI-powered code editor that can be configured to use custom OpenAI-compatible base URLs. This enables enterprise deployments to route Cursor requests through a central gateway like LiteLLM for logging, authentication, and cost control.

### Key Features

- **Custom Base URL Support**: Configure Cursor to point to LiteLLM gateway
- **Per-User Key Tracking**: Generate unique LiteLLM virtual keys per user
- **Audit Trail**: All Cursor requests logged to central database
- **Budget Controls**: Per-key spend limits and cost tracking

### Authentication Boundary

This repo supports Cursor through the **gateway-routed API-compatible path**:
- Cursor `Base URL` -> LiteLLM gateway
- Cursor `API Key` -> LiteLLM virtual key

If users authenticate directly to vendor endpoints outside this configuration, that is a bypass path. Bypass paths are handled with network controls (SWG/CASB/egress) plus detection and response workflows, not gateway inline enforcement.

### Version Variability Note

Cursor settings vary by version. Configuration steps may differ between releases. This document covers the general approach; consult Cursor documentation for version-specific details.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Configuration Steps](#configuration-steps)
3. [Local vs Remote Mode](#local-vs-remote-mode)
4. [Verification](#verification)
5. [Troubleshooting](#troubleshooting)
6. [Related Documentation](#related-documentation)

---

## Prerequisites

### Required Components

| Component | Purpose |
|-----------|---------|
| Cursor IDE | AI-powered code editor |
| LiteLLM Gateway | Central API gateway for routing and logging |
| PostgreSQL | Persistent storage for audit logs |

### Quick Setup with Onboarding Script

The easiest way to configure Cursor is using the onboarding script:

```bash
# Generate configuration for Cursor
make onboard TOOL=cursor

# With verification
make onboard TOOL=cursor VERIFY=1

# For remote Docker host
make onboard TOOL=cursor HOST=GATEWAY_HOST
```

This will generate a LiteLLM virtual key and display the configuration values you need to enter in Cursor settings.

### Manual Key Generation (Alternative)

If you prefer to generate keys manually:

```bash
# Using the Makefile
make key-gen ALIAS=cursor-user-username BUDGET=25.00
```

Store the returned key securely. You will need it for Cursor configuration.

---

## Configuration Steps

### Step 1: Open Cursor Settings

1. Open Cursor
2. Navigate to **Settings** (Cursor menu on macOS, File menu on Windows/Linux)
3. Select **Models** or **AI Provider** settings

The exact location varies by Cursor version. Look for sections related to:
- OpenAI Configuration
- Custom Base URL
- API Key Settings
- Model Provider

### Step 2: Configure Custom Base URL

Set the custom OpenAI base URL to point to your LiteLLM gateway:

**Local Mode (Single Machine):**
```
http://127.0.0.1:4000
```

**Remote Mode (Docker Host):**
```
http://GATEWAY_HOST:4000
```

Replace `GATEWAY_HOST` with your gateway host (hostname or IP).

### Step 3: Set Provider Key

Configure the API key field to use your LiteLLM virtual key:

```
sk-<your-litellm-virtual-key>
```

This is the key generated in the prerequisites step, not your provider's raw API key.

### Step 4: Configure Model Selection

Cursor will use the model aliases configured in `demo/config/litellm.yaml`. The current configuration includes:

| Alias | Model |
|-------|-------|
| `openai-gpt5.2` | `openai/gpt-5.2` |
| `claude-sonnet-4-5` | `anthropic/claude-sonnet-4-5` |

For API-key mode, ensure the corresponding provider API key is set in `demo/.env`:
- `OPENAI_API_KEY` for OpenAI models
- `ANTHROPIC_API_KEY` for Anthropic models

### Step 5: Save and Test

1. Save the configuration
2. Close and reopen Cursor if required
3. Make a test request in the Cursor chat interface

---

## Local vs Remote Mode

### Local Mode (Single Machine)

All services run on the same machine as Cursor.

| Setting | Value |
|---------|-------|
| Base URL | `http://127.0.0.1:4000` |
| API Key | Your LiteLLM virtual key |
| Network | Localhost only |

Use this mode for:
- Local development and testing
- Single-machine demos
- Initial configuration testing

### Remote Mode (Docker Host)

Services run on a remote Docker host, Cursor connects over network.

| Setting | Value |
|---------|-------|
| Base URL | `http://GATEWAY_HOST:4000` |
| API Key | Your LiteLLM virtual key |
| Network | Client to Docker host |

Use this mode for:
- Multi-user environments
- Remote client + gateway host demo topology (optional)
- Production-like deployments

### Network Topology (Remote Mode)

```
┌─────────────────────────────────────┐
│      Client Machine                 │
│      (Cursor running here)          │
│  ┌───────────────────────────────┐  │
│  │  Cursor Settings:             │  │
│  │  Base URL: GATEWAY_HOST:4000  │  │
│  │  API Key: sk-<virtual-key>    │  │
│  └───────────────────────────────┘  │
└──────────────┬────────────────────────┘
               │ Network
               ▼
┌─────────────────────────────────────┐
│      Docker Host                    │
│      GATEWAY_HOST                   │
│  ┌───────────────────────────────┐  │
│  │  LiteLLM: 0.0.0.0:4000        │  │
│  │  Routes to upstream providers │  │
│  │  Logs to PostgreSQL           │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

For network setup details, see [DEPLOYMENT.md](../DEPLOYMENT.md).

---

## Verification

### Step 1: Make a Test Request

In Cursor, make a simple request in the AI chat:

```
Hello, can you write a Python function to add two numbers?
```

### Step 2: Check Audit Logs

Verify the request appears in LiteLLM audit logs:

```bash
# Check recent audit log entries
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c \
  "SELECT
          s.request_id,
          s.model,
          COALESCE(v.key_alias, 'unknown') AS key_alias,
          s.spend,
          s.status,
          TO_CHAR(s.\"startTime\", 'YYYY-MM-DD HH24:MI:SS') AS timestamp
   FROM \"LiteLLM_SpendLogs\" s
   LEFT JOIN \"LiteLLM_VerificationToken\" v
     ON s.api_key = v.token
   ORDER BY s.\"startTime\" DESC
   LIMIT 10;"
```

Expected output: Recent requests with your `key_alias` (e.g., `cursor-user-username`).

### Step 3: Verify Attribution

Confirm requests are correctly attributed to your user key:

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
   WHERE v.key_alias LIKE 'cursor-%'
   GROUP BY v.key_alias
   ORDER BY total_spend DESC;"
```

### Step 4: Check Budget Status

Verify budget tracking is working:

```bash
make db-status
```

Look for your key in the budget table output.

### Step 5: Run Detection Rules

Verify security monitoring is working:

```bash
make detection
```

This runs all SIEM-style detection rules against the audit log.

---

## Troubleshooting

### Cursor Settings Not Found

**Symptom**: Cannot find custom base URL settings in Cursor

**Possible Causes**:
- Cursor version does not support custom base URL
- Settings location changed in newer versions
- Enterprise edition required for custom endpoints

**Solutions**:
- Check Cursor documentation for your version
- Consider upgrading to latest version
- For enterprise deployments, contact Cursor support

### Connection Errors

**Symptom**: Cursor shows connection errors or timeouts

**Diagnosis**:
```bash
# Test gateway connectivity
curl http://127.0.0.1:4000/health  # Local mode
curl http://GATEWAY_HOST:4000/health  # Remote mode

# Check if LiteLLM is running
make ps
```

**Solutions**:
- Verify LiteLLM is running: `make up`
- Check firewall allows port 4000 from client
- Confirm base URL matches your deployment mode
- Check Docker host connectivity in remote mode

### Authentication Failures

**Symptom**: 401 Unauthorized errors in Cursor

**Diagnosis**:
```bash
# Verify virtual key exists
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c \
  "SELECT key_alias, max_budget, user_id
   FROM \"LiteLLM_VerificationToken\"
   WHERE key_alias = 'cursor-user-username';"
```

**Solutions**:
- Confirm API key in Cursor matches your LiteLLM virtual key
- Check key has not exceeded budget: `make db-status`
- Regenerate key if needed: `make key-gen ALIAS=cursor-user-username BUDGET=25.00`
- Ensure key is not expired

### Model Not Available

**Symptom**: Cursor reports model not found or unavailable

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
- Confirm Cursor is requesting a valid model name

### No Logs Appearing

**Symptom**: Cursor requests succeed but no audit log entries

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
- Check that requests are actually being made in Cursor
- Verify Cursor is using the custom base URL (not default)

### Base URL Not Taking Effect

**Symptom**: Configuration looks correct but traffic bypasses gateway

**Possible Causes**:
- Cursor cached old configuration
- Settings applied to wrong provider
- Restart required for changes to take effect

**Solutions**:
- Fully close and reopen Cursor
- Check for multiple provider sections in settings
- Verify configuration in all relevant settings panels
- Check Cursor logs for connection errors

---

## Version-Specific Notes

Cursor configuration varies by version. This section documents known differences:

### Cursor Version 0.x

- Custom base URL may be under "Advanced" settings
- Provider key configuration may be separate from base URL
- Some features require Pro or Business tier

### Cursor Version 1.x+

- Unified provider settings panel
- Model-specific configuration options
- Improved custom endpoint support

### Getting Your Version

Check Cursor version in **Help > About** (or equivalent menu for your platform).

---

## Enterprise Deployment Considerations

For production enterprise deployments of Cursor with LiteLLM:

### Configuration Management

- **Managed Configuration**: Use Cursor's enterprise configuration to enforce base URL
- **Prevent Overrides**: Lock settings to prevent users from bypassing gateway
- **Standardized Models**: Pre-configure approved model aliases

### Egress Controls

- **Firewall Rules**: Block direct access to `api.openai.com` and other provider endpoints
- **DNS Filtering**: Use DNS-based filtering to prevent bypass
- **Proxy Inspection**: Use SWG to inspect and log all AI tool traffic

### User Onboarding

1. Generate unique virtual key per user
2. Provide configuration instructions with their specific key
3. Verify first request appears in audit logs
4. Set up budget alerts for each user

### Monitoring

- Set up detection rules for anomalous usage (see [DETECTION.md](../security/DETECTION.md))
- Configure budget alerts per user
- Regular review of audit logs for policy violations

---

## Related Documentation

| Document | Purpose |
|----------|---------|
| [DEPLOYMENT.md](../DEPLOYMENT.md) | Network setup and deployment modes |
| [DATABASE.md](../DATABASE.md) | Audit log verification and database queries |
| [DETECTION.md](../security/DETECTION.md) | Security monitoring and detection rules |
| [Local Demo Implementation Plan](../LOCAL_DEMO_PLAN.md) | End-to-end demo flow and environment assumptions |
| [LiteLLM Documentation](https://docs.litellm.ai/) | Official LiteLLM documentation |

---

## Additional Resources

- **Cursor Documentation**: <https://cursor.sh/docs> (consult for version-specific settings)
- **LiteLLM Proxy Documentation**: <https://docs.litellm.ai/docs/proxy>
- **OpenAI API Compatibility**: <https://platform.openai.com/docs/api-reference>
