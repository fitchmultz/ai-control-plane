# AI Control Plane -- 10-Minute Demo Script

## Overview

- **Target audience**: CTO, Security leadership, Practice leadership
- **Total time**: 10 minutes (strict)
- **Prerequisites**: `make up-offline` running, clean database state

## Pre-Demo Checklist (5 minutes before)

- [ ] Run `make health` -- verify all services healthy
- [ ] Run `make db-status` -- confirm clean state
- [ ] Terminal prepared with large font (14pt+ monospace)
- [ ] Backup terminal ready (tmux session: `tmux new -s demo-backup`)
- [ ] Demo environment running: `make up-offline` (no provider keys needed)

### Known Environment Issues

**Port 4000 Conflict**: If another container is using port 4000, the gateway will fail to start.
```bash
# Check for conflicts
docker ps --filter "publish=4000"
# Stop conflicting container if found
docker stop <container-name>
```

**Offline Mode Model Names**: In offline mode, use `mock-gpt` and `mock-claude` instead of real model names.

---

## Interactive Mode

For guided presentations with stakeholder engagement, use interactive mode:

### Running in Interactive Mode

```bash
# Pause after each step for Q&A
make demo-scenario SCENARIO=1

# Or with preset
INTERACTIVE=1 make demo-preset PRESET=executive-demo

# With automatic narration
make demo-scenario SCENARIO=1
```

### Interactive Mode Controls

- **Enter**: Continue to next step
- **q**: Quit demo early
- **Ctrl+C**: Abort demo

### Time Estimates (Interactive vs Non-Interactive)

| Scenario | Normal | Interactive |
|----------|--------|-------------|
| 1: API Path | 30s | 2-3 min |
| 2: Claude Subscription | 45s | 3-4 min |
| 3: Codex Subscription | 45s | 3-4 min |
| 4: Budget | 45s | 3-5 min |
| 5: Governance Summary | 30s | 2-3 min |
| 6: DLP | 60s | 5-7 min |
| 7: Network Enforcement | 40s | 3-4 min |
| 8: Rapid Response | 45s | 3-5 min |
| 9: Cursor Governed Path | 45s | 3-4 min |
| 10: Confidential Detection | 60s | 4-6 min |
| 11: Chargeback/Showback | 60s | 4-6 min |

### Recommended Presentation Flow

1. Start with narration: `make demo-scenario SCENARIO=1`
2. Follow with `make demo-scenario SCENARIO=9` for IDE governance path
3. Close with `make demo-scenario SCENARIO=11` for chargeback attribution
4. Take questions before continuing
5. Use `q` to exit early if running over time

---

## Milestone 1: Architecture Overview (1:30)

**Time**: 0:00 - 1:30

### Narration

> "The AI Control Plane uses a route-based governance model:
>
> **Gateway-routed path**: CLI tools and managed web UX routed through LiteLLM get enforcement plus observability in one place.
>
> **Bypass/direct path**: direct-to-vendor traffic is detection + response unless customer network controls prevent bypass.
>
> The key insight: governance quality depends on routing discipline plus explicit customer network hardening."

### Commands to Run

```bash
make health
```

**Expected output**: All services healthy, containers running.

```bash
docker compose -f demo/docker-compose.yml ps
```

**Expected output**: `litellm` and `postgres` containers running.

### Architecture Diagram Reference

Show the route-based model:
- Gateway-routed (API-key or subscription-backed CLI): Tool/App -> Gateway (port 4000) -> Provider
- Direct bypass: Tool -> Provider (OAuth) -> OTEL/Compliance Export -> SIEM

### Evidence Reference

- `demo/logs/evidence/01_fresh_clone_health.log`
- `demo/logs/evidence/03_service_startup.log`

### Fallback

If health check fails:
> "Let me show you the architecture from the documentation instead..."

Open `docs/ENTERPRISE_STRATEGY.md` lines 54-75.

---

## Milestone 2: Secure Request Flow (2:00)

**Time**: 1:30 - 3:30

### Narration

> "Let me demonstrate the key lifecycle. Every request requires a virtual key with specific attributes: budget limit, approved models, and rate limits.
>
> Keys are generated with a single command, and every request is attributed to the key holder for audit and chargeback."

### Commands to Run

```bash
# Generate a demo key with budget limit
ACP_OFFLINE_MODE=1 make key-gen ALIAS=demo-key BUDGET=1.00
```

**Expected output**: Key generated with alias `demo-key`, $1.00 budget.

```bash
# Make a test request through the gateway
curl -X POST "${GATEWAY_URL:-http://127.0.0.1:4000}/v1/chat/completions" \
  -H "Authorization: Bearer $DEMO_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "mock-claude", "messages": [{"role": "user", "content": "Say hello in one word"}]}'
```

**Expected output**: Response from the model (in offline mode, this returns a mock response).

```bash
# Show the audit log entry
make db-status
```

**Expected output**: Shows recent requests attributed to `demo-key`.

### Key Talking Points

1. **Attribution**: Every request is linked to a key alias (not just a token)
2. **Budget tracking**: Real-time spend accumulation
3. **Model enforcement**: Request blocked if model not in key's allowed list

### Evidence Reference

- `demo/logs/evidence/04_key_lifecycle.log`
- `docs/demos/API_KEY_GOVERNANCE_DEMO.md` lines 36-80

### Offline Mode Note

For deterministic offline demos, set `ACP_OFFLINE_MODE=1` before key generation so the generated key matches the offline model aliases. If you still encounter "Invalid model name" errors, use the raw API to generate a key with the correct offline models:

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:4000}"
MASTER_KEY="$(./scripts/acpctl.sh env get LITELLM_MASTER_KEY)"

# Generate key with offline models
curl -X POST "${GATEWAY_URL}/key/generate" \
  -H "Authorization: Bearer $MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key_alias": "demo-key", "max_budget": 1.00, "models": ["mock-gpt", "mock-claude"]}'
```

Then use `mock-claude` or `mock-gpt` in your test requests instead of real model names.

### Fallback

If key generation fails:
> "The key generation uses the LiteLLM `/key/generate` endpoint. Let me show you the raw API..."

```bash
curl -X POST "${GATEWAY_URL:-http://127.0.0.1:4000}/key/generate" \
  -H "Authorization: Bearer $MASTER_KEY" \
  -d '{"key_alias": "fallback-key", "max_budget": 1.00}'
```

---

## Milestone 3: DLP Block Demo (1:30)

**Time**: 3:30 - 5:00

### Narration

> "Now let's demonstrate data loss prevention. The gateway integrates with Microsoft Presidio for content-based DLP.
>
> When a request contains sensitive data -- Social Security numbers, credit cards, API keys -- the routed guardrail path is designed to block the request before it reaches the provider. In offline mode, treat this milestone as configuration and workflow proof unless you have revalidated live blocking in a provider-capable environment."

### Commands to Run

```bash
# Run the DLP scenario
make demo-scenario SCENARIO=6
```

Or manually:

```bash
# Test with SSN (will be blocked)
curl -X POST "${GATEWAY_URL:-http://127.0.0.1:4000}/v1/chat/completions" \
  -H "Authorization: Bearer $DEMO_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "mock-claude", "messages": [{"role": "user", "content": "My SSN is 123-45-6789"}]}'
```

**Expected output**: HTTP 400/403 with "blocked" message.

### DLP Entities Configured

**Blocked Entities (prevent request from proceeding):**

| Entity | Action |
|--------|--------|
| US_SSN | BLOCK |
| US_PASSPORT | BLOCK |
| US_DRIVER_LICENSE | BLOCK |
| CREDIT_CARD | BLOCK |
| US_BANK_NUMBER | BLOCK |
| US_ITIN | BLOCK |
| CRYPTO | BLOCK |
| IBAN_CODE | BLOCK |

**Masked Entities (redacted in transit):**

| Entity | Action |
|--------|--------|
| EMAIL_ADDRESS | MASK |
| PHONE_NUMBER | MASK |
| LOCATION | MASK |
| PERSON | MASK |

> **Note:** See `docs/security/DETECTION.md` lines 416-430 for complete DLP configuration.

### Known Limitation (Acknowledge This)

> "In this demo environment, DLP is regex-based. Production deployments should use Presidio's ML-based content analysis for higher accuracy. See docs/KNOWN_LIMITATIONS.md."

**Offline Mode DLP Limitation**: Offline mode is useful for workflow rehearsal, but it is not the final proof point for live DLP blocking. If DLP does not block in your demo, present the configuration and state that provider-capable validation is required before claiming live block efficacy:

```bash
cat demo/config/litellm-offline.yaml | grep -A25 "guardrails:"
```

### Evidence Reference

- `demo/logs/evidence/06_dlp_guardrails.log`
- `docs/security/DETECTION.md` lines 404-475

### Fallback

If DLP doesn't block:
> "DLP requires Presidio services to be running. Let me show the configuration..."

```bash
cat demo/config/litellm.yaml | grep -A20 "guardrails:"
```

---

## Milestone 4: Budget Control (1:30)

**Time**: 5:00 - 6:30

### Narration

> "Budget enforcement is real-time. When a key's budget is exhausted, subsequent requests are rejected -- no surprise bills, no runaway automation.
>
> Let me show you the current budget state and explain how this maps to chargeback."

### Commands to Run

```bash
# Show budget status for all keys
make db-status
```

**Expected output**: Shows keys with spend vs. max_budget.

```bash
# Run budget enforcement scenario (simulates exhaustion)
make demo-scenario SCENARIO=4
```

### Budget Tracking Schema

```sql
SELECT
  v.key_alias,
  ROUND(v.spend::numeric, 4) AS spent,
  ROUND(b.max_budget::numeric, 4) AS budget,
  ROUND((v.spend/NULLIF(b.max_budget,0)*100)::numeric, 2) AS percent_used
FROM "LiteLLM_VerificationToken" v
JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id;
```

### Chargeback Convention

Key aliases follow a convention for attribution:

```
team-platform__cc-12345      # Team + cost center
svc-analytics__team-data     # Service account
usr-jdoe123__team-eng        # Individual user
```

### Evidence Reference

- `demo/logs/evidence/05_budget_enforcement.log`
- `docs/demos/API_KEY_GOVERNANCE_DEMO.md` lines 53-80

### Fallback

If no budget data visible:
> "Budget tracking populates as requests are made. Let me show the schema..."

```bash
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "\d \"LiteLLM_BudgetTable\""
```

---

## Milestone 5: Detection Rules (1:30)

**Time**: 6:30 - 8:00

### Narration

> "Detection rules analyze usage patterns and surface anomalies. These are SIEM-style rules that run against the gateway logs.
>
> High-severity detections trigger immediate alerts. Let me run the detection suite and show you what we find."

### Commands to Run

```bash
# Run all detection rules
make detection
```

**Expected output**: Validation output indicating which detection rules are active.

```bash
# Re-run detection validation
make detection
```

### Key Detection Rules

| Rule | Severity | What It Detects |
|------|----------|-----------------|
| DR-001 | High | Non-approved model access |
| DR-006 | High | Failed authentication attempts |
| DR-002 | Medium | Token usage spike (>100K tokens) |
| DR-005 | Medium | Rapid request rate (>60/min) |

### Response Mapping

> "High-severity detections can trigger automated key suspension. DR-001 (non-approved model) suspends immediately; DR-006 (auth failures) has a 5-minute grace period."

### Evidence Reference

- `demo/logs/evidence/09_detection_rules_all.log`
- `docs/security/DETECTION.md` lines 10-290

### Fallback

If no findings:
> "Clean! No detections means no anomalies. Let me show sample findings from the documentation..."

Open `docs/security/DETECTION.md` lines 625-640 (example output).

---

## Milestone 6: Rapid Response & Recovery (2:00)

**Time**: 8:00 - 10:00

### Narration

> "When we detect a compromise, response time matters. Let me demonstrate the rapid response workflow:
>
> 1. Detection identifies suspicious key
> 2. Operator revokes the key
> 3. Key is immediately invalidated
> 4. Evidence is preserved for forensics"

### Commands to Run

```bash
# Run the rapid response scenario (official entrypoint)
make demo-scenario SCENARIO=8
```

This scenario will:
1. Generate a test key
2. Simulate suspicious activity (non-approved model access)
3. Run detections to identify the compromise
4. Revoke the key
5. Verify revocation
6. Show forensics evidence

### Key Revocation Commands

```bash
# Revoke a key by alias
make key-revoke ALIAS=<alias>
```

### Evidence Collection

```bash
# Capture detection findings
make detection > incident-$(date +%Y%m%d)-detections.txt

# Capture audit logs
make db-status > incident-$(date +%Y%m%d)-dbstate.txt
```

### Recovery Workflow

1. **Revoke** compromised key immediately
2. **Investigate** via audit logs and detection findings
3. **Rotate** generate replacement key
4. **Document** preserve evidence for incident ticket

### Evidence Reference

- `demo/logs/evidence/09_detection_rules_all.log`
- `docs/demos/API_KEY_GOVERNANCE_DEMO.md` lines 400-525
- `docs/RUNBOOK.md` section 9.6

### Fallback

If scenario fails:
> "Let me walk through the manual steps..."

```bash
# Manual key revocation demo
make key-gen ALIAS=manual-revoke-test BUDGET=0.50
make key-revoke ALIAS=<alias>
# Attempt to use revoked key (should fail with 401)
```

---

## Closing Summary (30 seconds)

### What We Demonstrated

| Milestone | Key Takeaway |
|-----------|-------------|
| Architecture | Two-track model: enforcement (API-key) + governance (subscription) |
| Request Flow | Key generation, attribution, audit logging |
| DLP | Content-based blocking before provider |
| Budget | Real-time enforcement, no runaway costs |
| Detection | SIEM-style rules, automated alerts |
| Rapid Response | Key revocation in seconds, evidence preserved |

### Known Limitations (Be Transparent)

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| Port 4000 Conflict | Gateway fails if port occupied | Use `LITELLM_HOST_PORT` override |
| Offline Token Estimation | Token counts estimated | Use real providers for precision |
| DLP Regex Baseline | Demo DLP is regex-only | Use Presidio for production |

### Next Steps

> "For production deployment, we'd configure:
> - Egress deny-by-default to prevent bypass
> - SIEM integration for alerting
> - Key rotation policies
>
> Questions?"

---

## Timing Summary

| Milestone | Duration | Cumulative |
|-----------|----------|------------|
| Architecture | 1:30 | 1:30 |
| Request Flow | 2:00 | 3:30 |
| DLP Block | 1:30 | 5:00 |
| Budget Control | 1:30 | 6:30 |
| Detection | 1:30 | 8:00 |
| Rapid Response | 2:00 | 10:00 |

---

## Demo Environment Recovery

If demo gets into a bad state:

```bash
# Reset to clean state
make down
make clean
make up-offline

# Verify health
make health
```

---

## Live-Demo Checklist (Final Verification)

Use this checklist immediately before presenting to ensure deterministic execution:

- [ ] `make health` passes
- [ ] `make db-status` shows clean state
- [ ] Offline mode configured for demo commands (`ACP_OFFLINE_MODE=1`) if required
- [ ] Rapid response path verified: `make demo-scenario SCENARIO=8` (execution check)
- [ ] Confidential detection path verified: `make demo-scenario SCENARIO=10`
- [ ] Chargeback/showback path verified: `make demo-scenario SCENARIO=11`
- [ ] Fallback prepared: `make key-gen` + `make key-revoke`

---

## Related Documentation

- [Q&A Packet](q_and_a.md) -- Answers to hard questions
- [Known Limitations](../KNOWN_LIMITATIONS.md) -- Active and closed limitations
- [Deployment Guide](../DEPLOYMENT.md) -- Full deployment documentation
- [API Key Governance Demo](../demos/API_KEY_GOVERNANCE_DEMO.md) -- Detailed walkthrough
- [Detection Rules](../security/DETECTION.md) -- Complete rule reference
- [Runbook](../RUNBOOK.md) -- Operational procedures
