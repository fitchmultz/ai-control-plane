# SaaS/Subscription Governance Demonstration

**Executive Summary:** This document demonstrates how the AI Control Plane governs SaaS/subscription-based AI tooling with a route-based model: gateway-routed CLI traffic (Claude Code/Codex) gets full enforcement + observability, while direct-to-vendor bypass paths remain detection-first.

---

## What This Doc Is Responsible For

- Demonstrating the **route-based governance model**: gateway-routed traffic gets enforcement + observability; bypass traffic gets detection + response.
- Providing a repeatable walkthrough for subscription-mode onboarding and evidence collection.
- Showing how telemetry from multiple sources (gateway, OTEL, compliance exports) merges into a unified SIEM feed.

## What This Doc Does NOT Cover

- Full production hardening (see `DEPLOYMENT.md`, `RUNBOOK.md`).
- API-key enforcement details (see `API_KEY_GOVERNANCE_DEMO.md`).
- Egress/SWG/CASB enforcement (architecture-level; requires customer infrastructure).

## Invariants / Assumptions

- **API-key mode** can be enforced inline at the gateway (blocking, budgets, rate limits).
- **Subscription-backed upstream (routed)** can also be enforced at the gateway; upstream billing is handled by the vendor subscription.
- **Direct subscription (bypass)** is detection-first: investigation and governance reporting apply, while enforcement depends on vendor capabilities and customer egress controls.
- Gateway sees subscription traffic when tools are routed through it (Claude Code and Codex can both route through gateway when using LiteLLM's subscription provider support).
- ChatGPT web and Claude web are vendor-hosted interfaces and are not forcibly gateway-routed by default; use LibreChat when a managed browser interface must stay on the governed path.
- OTEL collector provides telemetry for tools that support it or when direct vendor auth is used.
- Compliance API exports provide audit-grade logs where available (OpenAI Enterprise).

---

## Overview

The SaaS/subscription governance demonstration showcases how Project addresses the **governance gap** for AI tools that authenticate via OAuth or vendor workspaces rather than API keys:

| Governance Pillar | API-Key Mode (Routed) | Subscription-Backed Upstream (Routed) | Direct Subscription (Bypass) |
|-------------------|--------------|---------------------------------------|------------------------------|
| **Authentication** | Virtual keys (gateway-managed) | OAuth / vendor workspaces + virtual key | OAuth / vendor workspaces |
| **Enforcement** | Inline blocking, budgets, rate limits | Inline blocking, budgets, rate limits (gateway-side) | Detection + response (logging + investigation) |
| **Telemetry** | PostgreSQL audit logs | PostgreSQL audit logs (dual with OTEL possible) | OTEL + compliance exports |
| **Attribution** | key_alias | key_alias + user_id / email | user_id / email |

### Scenario-to-Goal Matrix

| Scenario | Primary Demo Goal | Evidence Produced |
|----------|-------------------|-------------------|
| `2` Claude subscription | Prove subscription-backed CLI can stay on governed path | Gateway audit records + attribution |
| `3` Codex subscription | Prove Codex routed and direct paths are distinct | Routed audit evidence and direct-path caveat |
| `9` Cursor governed path | Prove IDE workflow can route through LiteLLM | Key alias + model + spend in PostgreSQL |
| `10` Confidential detection | Prove deterministic confidential payload detection | DR-008 correlation to request principal |
| `11` Chargeback/showback | Prove user-to-department cost allocation | JSON/CSV chargeback artifacts by cost center |

### Route-Based Governance Model

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                     AI Control Plane Governance                                          │
├─────────────────────────────┬───────────────────────────────────┬───────────────────────┤
│   API-Key Mode              │   Subscription-Backed (Routed)    │ Direct Subscription   │
│   (Enforce + Observe)       │   (Enforce + Observe)             │ (Detect + Respond)    │
│                             │                                   │                       │
│  ┌─────────────────────┐    │  ┌─────────────────────┐         │  ┌─────────────────┐  │
│  │ LiteLLM Gateway     │    │  │ LiteLLM Gateway     │         │  │ Codex CLI       │  │
│  │                     │    │  │                     │         │  │ (direct OAuth)  │  │
│  │ • Allowlist blocking│    │  │ • Allowlist blocking│         │  │                 │  │
│  │ • Budget enforcement│    │  │ • Budget enforcement│         │  │ OTEL only       │  │
│  │ • Rate limiting     │    │  │ • Rate limiting     │         │  │                 │  │
│  └─────────────────────┘    │  │ • OAuth forwarding  │         │  └─────────────────┘  │
│                             │  └─────────────────────┘         │                       │
│  Enforcement: ACTIVE        │  Vendor billing: Subscription     │  Enforcement: VENDOR  │
│  Telemetry: PostgreSQL      │  Telemetry: PostgreSQL (+OTEL)    │  Telemetry: OTEL      │
└─────────────────────────────┴───────────────────────────────────┴───────────────────────┘
                              │
                              ▼
              ┌─────────────────────────────┐
              │  Normalized Evidence Schema  │
              │  (principal, model, tokens,  │
              │   cost, policy, correlation) │
              └─────────────────────────────┘
```

### Subscription-Backed Upstream Routing

Both Claude Code and Codex CLI **can** route through the gateway when using subscription-backed upstream providers:

**Claude Code (MAX Subscription)**
- Routes through gateway using `ANTHROPIC_BASE_URL` + custom header auth
- OAuth token forwarded to Anthropic via `forward_client_headers_to_llm_api: true`
- Gateway enforces budgets, rate limits, model allowlists
- See: [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)

**Codex CLI (ChatGPT Subscription)**
- Can route through gateway using LiteLLM's ChatGPT provider
- Device/OAuth tokens stored on gateway host (`litellm-proxy login`)
- Gateway enforces budgets, rate limits, model allowlists
- See: [LiteLLM ChatGPT Provider](https://docs.litellm.ai/docs/providers/chatgpt) and [LiteLLM OpenAI Codex](https://docs.litellm.ai/docs/tutorials/openai_codex)

---

## Prerequisites

1. Docker and Docker Compose installed
2. AI Control Plane repository cloned
3. Basic environment configured (`make install`)
4. For live compliance exports: OpenAI Enterprise credentials (optional; fixture mode works without)

---

## Canonical Runbook Flow

### Step 1: Bring Up Services

```bash
# Start the AI Control Plane
make up

# Verify health
make health
```

### Step 2: Claude Code Subscription-Through-Gateway Path

Claude Code supports routing through the gateway even in subscription mode, giving us **dual telemetry** (gateway logs + OTEL):

```bash
# Onboard Claude Code in subscription mode
make onboard TOOL=claude MODE=subscription VERIFY=1
```

This will:
1. Generate a LiteLLM virtual key for gateway authentication
2. Display environment variables for Claude Code configuration
3. Optionally verify connectivity

**Key configuration output:**
```bash
# Set these environment variables:
export ANTHROPIC_BASE_URL=http://127.0.0.1:4000
export ANTHROPIC_CUSTOM_HEADERS="x-litellm-api-key: Bearer sk-litellm-..."
# Then select subscription login in Claude Code
```

Run the subscription scenario:

```bash
make demo-scenario SCENARIO=2
```

**What this demonstrates:**
- Claude Code using OAuth identity but routing through the gateway
- Gateway provides policy enforcement (allowlists, budgets)
- OAuth token forwarded to upstream (enables subscription billing)
- Telemetry captured in PostgreSQL audit logs

### Step 3: Codex Subscription-Through-Gateway Path

Codex CLI can also route through the gateway using LiteLLM's ChatGPT provider support:

```bash
# First, authenticate with ChatGPT on the gateway host
make chatgpt-login

# Onboard Codex in subscription-backed mode
make onboard TOOL=codex MODE=subscription VERIFY=1
```

**Key configuration output:**
```bash
# Set these environment variables:
export OPENAI_BASE_URL=http://127.0.0.1:4000
export OPENAI_API_KEY="sk-litellm-..."  # LiteLLM virtual key
# Then select API key authentication in Codex
```

**What this demonstrates:**
- Codex routing through gateway with ChatGPT subscription as upstream
- Gateway enforces budgets, rate limits, model allowlists
- Upstream billing via ChatGPT subscription
- Telemetry captured in PostgreSQL audit logs

### Step 3B: Cursor Governed Path (IDE)

Run the Cursor scenario:

```bash
make demo-scenario SCENARIO=9
```

**What this demonstrates:**
- Cursor can use an OpenAI-compatible base URL through LiteLLM
- Requests remain attributable by user/team/cost center alias
- IDE traffic inherits the same gateway policy and observability posture

### Alternative: Codex Direct Subscription (OTEL Mode)

If Codex uses direct OAuth to OpenAI (bypassing the gateway), use OTEL for telemetry:

**Note:** OTEL collector is opt-in (Compose profile `otel`) and is not started by `make up` or CI.

```bash
# Start the OTEL collector (enables the 'otel' Compose profile)
make up-production

# Verify OTEL health
make otel-health

# Onboard Codex in direct mode (no gateway routing)
make onboard TOOL=codex MODE=direct VERIFY=1
```

**Key configuration output:**
```bash
# For direct subscription telemetry, ensure OTEL is configured:
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

Make a Codex request (manual step):

```bash
# In a terminal with the OTEL environment variables set:
codex "Hello, verify telemetry capture"
```

Check captured telemetry:

```bash
make logs
```

**What this demonstrates:**
- Codex using direct OpenAI OAuth authentication
- No gateway enforcement possible (direct vendor connection)
- OTEL collector captures telemetry for governance visibility
- Detection and response controls only (no inline blocking)

### Step 4: Compliance Exports (Fixture Mode)

Pull OpenAI compliance exports for enterprise governance:

```bash
# Pull compliance exports (fixture mode by default)
make validate-detections

# View the exported events
make db-status
```

**Output location:**
```
demo/logs/compliance/compliance_events.jsonl
```

**To use live mode** (requires credentials):

```bash
# Set credentials (do not commit these)
export OPENAI_COMPLIANCE_API_KEY="your-key"
export OPENAI_ORG_ID="your-org-id"

# Pull live compliance data
make validate-detections
```

**What this demonstrates:**
- Enterprise audit exports with user attribution
- Pre-redacted by OpenAI (no prompt/response content)
- Metadata includes: user email, model, tokens, cost, timestamp
- Complements OTEL telemetry with vendor-verified records

### Step 5: Evidence Pipeline (Merge + Validate)

Run the complete evidence pipeline to merge all sources:

```bash
# Run full pipeline: export + merge + validate
make release-bundle
```

Or run individual steps:

```bash
# Export gateway logs from PostgreSQL
make release-bundle

# Merge all sources (gateway + OTEL + compliance)
make release-bundle

# Validate JSON structure
make release-bundle-verify

# View the unified evidence feed
make release-bundle-verify
```

### Step 6: Confidential File Detection + Chargeback Evidence

```bash
# Deterministic confidential payload and detection correlation
make demo-scenario SCENARIO=10

# User/department cost allocation outputs
make demo-scenario SCENARIO=11
```

**What this demonstrates:**
- Confidential-content alerts can be correlated to a principal and rule ID
- Chargeback reports provide executive-ready showback by department
- Governance storyline is complete: enforce, detect, attribute, allocate

**Output locations:**
| Source | File Path |
|--------|-----------|
| Gateway | `demo/logs/gateway/gateway_events.jsonl` |
| OTEL | `demo/logs/otel/telemetry.jsonl` |
| Compliance | `demo/logs/compliance/compliance_events.jsonl` |
| **Unified** | `demo/logs/normalized/evidence.jsonl` |

**What this demonstrates:**
- Normalized schema across all AI activity sources
- Chronological merge with deduplication
- Single SIEM-ingestable feed
- Evidence integrity validation

### Step 6: SIEM-Facing Demonstration

Run the interactive SIEM demo:

```bash
make validate-detections
```

View the normalized schema:

```bash
cat demo/config/normalized_schema.yaml
```

View SIEM query examples:

```bash
make validate-siem-queries
# Or directly:
cat demo/config/siem_queries.yaml
```

**What this demonstrates:**
- Unified evidence schema for cross-source correlation
- Platform-specific queries (Splunk, ELK, Sentinel)
- Detection rules mapped to SIEM formats
- Sample telemetry for testing integrations

### Step 7: Executive Reporting

Generate canonical governance evidence:

```bash
make db-status
make release-bundle
make release-bundle-verify

# Legacy scorecard command is a compatibility stub in public snapshot
make chargeback-report
```

**What this demonstrates:**
- Executive-ready readiness evidence package
- Cross-source usage aggregation
- Policy compliance metrics
- Detection findings summary

---

## Evidence Checklist (What to Screenshot)

After running the demo, capture these artifacts:

| Artifact | Command | File Location |
|----------|---------|---------------|
| Gateway health | `make health` | Terminal output |
| Virtual keys | `make db-status` | Terminal output |
| OTEL telemetry | `make logs` | `demo/logs/otel/telemetry.jsonl` |
| Compliance events | `make db-status` | `demo/logs/compliance/compliance_events.jsonl` |
| Unified evidence | `make release-bundle-verify` | `demo/logs/normalized/evidence.jsonl` |
| Detection results | `make detection` | Terminal output |
| Governance readiness evidence | `make release-bundle`; `make release-bundle-verify` | Terminal output + bundle artifacts |

---

## Normalized Schema Overview

All evidence sources normalize to the same schema:

```yaml
# Core identity
principal:
  id: "user@example.com"          # User identifier
  type: "user"
  email: "user@example.com"

# AI request details
ai:
  request:
    id: "req-123"
    timestamp: "2026-01-30T12:00:00Z"
  model:
    id: "claude-sonnet-4"
    provider: "anthropic"
  tokens:
    prompt: 100
    completion: 50
    total: 150
  cost:
    amount: 0.0015
    currency: "USD"

# Policy outcome
policy:
  action: "allowed"               # allowed, blocked, error
  reason: null

# Correlation
correlation:
  trace:
    id: "trace-abc123"

# Source attribution
source:
  type: "gateway"                 # gateway, otel, compliance
```

---

## Enforcement vs Detection Controls

### What We Can Enforce

| Control | API-Key Mode | Subscription-Backed (Routed) | Direct Subscription (Bypass) |
|---------|--------------|------------------------------|------------------------------|
| Model allowlist | ✓ Gateway blocks non-approved | ✓ Gateway blocks non-approved | ✗ Vendor-dependent |
| Budget limits | ✓ Gateway enforces per-key | ✓ Gateway enforces per-key | ✗ Detection only |
| Rate limiting | ✓ Gateway throttles | ✓ Gateway throttles | ✗ Detection only |
| Request logging | ✓ PostgreSQL audit logs | ✓ PostgreSQL audit logs | ✓ OTEL / compliance |
| Real-time blocking | ✓ HTTP 400/429 responses | ✓ HTTP 400/429 responses | ✗ Post-hoc alerting |

### What We Can Only Detect

| Scenario | Detection Method | Response |
|----------|------------------|----------|
| Direct subscription bypasses gateway | OTEL/compliance visibility gap | Alert + investigate |
| Shadow IT AI usage | Egress logs (if available) | Alert + block via SWG |
| Unapproved model in subscription | Compliance API analysis | Governance report |
| Excessive token usage | Detection rules (DR-002) | Alert + review |

---

## Failure Modes / Troubleshooting

### OTEL Collector Not Receiving Telemetry

```bash
# Check OTEL collector health
make otel-health

# Check OTEL logs
make logs

# Verify environment variables are set
env | grep OTEL
```

See: `docs/observability/OTEL_SETUP.md`

### Compliance Exports Empty

```bash
# Verify fixture mode is working
ls -la demo/logs/compliance/

# For live mode, verify credentials
env | grep OPENAI
```

### Gateway Logs Missing

```bash
# Verify database connectivity
make db-status

# Check if services are healthy
make health
```

### Evidence Pipeline Errors

```bash
# Validate individual sources exist
ls -la demo/logs/gateway/
ls -la demo/logs/otel/
ls -la demo/logs/compliance/

# Run validation separately
make release-bundle-verify
```

---

## Compliance Alignment

### Strategy Document Mapping

| Governance Control | Strategy Section | Implementation |
|-------------------|------------------|----------------|
| Route-based model | 4.1 Provider/Model Allowlisting | Gateway enforcement + OTEL detection |
| Evidence pipeline | 4.2 Evidence Pipeline | Normalized schema + unified feed |
| SaaS governance | 4.3 SaaS/Subscription Controls | OTEL + compliance exports |
| Cross-source correlation | 4.2 Normalized Evidence | principal.id matching across sources |

### Evidence Artifacts

**For Compliance Audits:**
1. **Configuration Evidence:** `demo/config/normalized_schema.yaml` shows field mapping
2. **Telemetry Records:** OTEL logs show subscription-mode activity
3. **Compliance Exports:** Vendor-verified audit logs (OpenAI Enterprise)
4. **Unified Evidence:** `demo/logs/normalized/evidence.jsonl` for SIEM ingestion

---

## Management Presentation Guide

### Key Talking Points

**1. The Governance Gap**
> "Most organizations only govern API-key AI usage, leaving a huge blind spot for subscription tooling. Our control plane closes this gap by routing supported CLI flows through the gateway and instrumenting bypass paths."

**2. Route-Based Model**
> "API-key mode and subscription-backed gateway routing both give us enforceable controls: block, budget, and throttle. Direct subscription bypass paths shift to detection and response with OTEL/compliance evidence."

**3. Unified Evidence**
> "Whether traffic comes through our gateway or goes direct to the vendor, we normalize it all into one SIEM feed. One query shows all AI activity across the enterprise."

**4. Vendor Partnership**
> "For OpenAI Enterprise customers, we ingest official compliance exports—vendor-verified audit logs that satisfy regulatory requirements."

### Live Demo Script

```bash
# 1. Show route-based architecture
echo "=== Track 1: API-Key Enforcement ==="
make demo-scenario SCENARIO=1

echo "=== Track 2: Subscription-Through-Gateway ==="
make demo-scenario SCENARIO=2
make demo-scenario SCENARIO=3

# 2. Pull compliance exports
echo "=== Compliance API Integration ==="
make validate-detections
make db-status | head -5

# 3. Run evidence pipeline
echo "=== Unified Evidence Pipeline ==="
make release-bundle
make release-bundle-verify | head -5

# 4. Show SIEM queries
echo "=== SIEM Integration ==="
make validate-detections

# 5. Generate governance evidence
echo "=== Executive Reporting ==="
make db-status
make release-bundle
make release-bundle-verify
make chargeback-report
```

---

## Seat-Based Billing and Showback

This section covers chargeback for subscription/SaaS-based AI tools where billing is seat-based rather than usage-based.

### What Can Be Enforced vs Only Observed

| Control | API-Key Mode | Subscription Mode |
|---------|--------------|-------------------|
| Real-time blocking | ✓ Gateway-enforced | ✗ Vendor-dependent |
| Budget enforcement | ✓ Per-key limits | ✗ Limited (vendor features) |
| Usage attribution | ✓ Key aliases | ✓ OTEL/workspace/export evidence |
| Content logging | ✓ Configurable | ✓ Vendor exports |

### Chargeback Model for Subscriptions

**Primary Basis: Seat Assignment**

```
┌─────────────────────────────────────────────────────────────┐
│               Seat-Based Chargeback Flow                     │
├─────────────────────────────────────────────────────────────┤
│  1. Vendor Invoice: 100 seats × $20/seat = $2,000           │
│                                                             │
│  2. Seat Roster:                                            │
│     - Platform Engineering: 40 seats                        │
│     - Data Science: 35 seats                                │
│     - Engineering: 25 seats                                 │
│                                                             │
│  3. Chargeback Allocation:                                  │
│     - CC-12345 (Platform): $800 (40%)                       │
│     - CC-67890 (Data Science): $700 (35%)                   │
│     - CC-54321 (Engineering): $500 (25%)                    │
└─────────────────────────────────────────────────────────────┘
```

### Usage Telemetry as Showback Signal

While seat cost is the **billable amount**, usage telemetry provides **optimization insights**:

```bash
# View OTEL telemetry (showback signal)
make logs

# Check compliance exports (where available)
make db-status
```

**Key Point:** Token-based `spend` in OTEL/compliance exports is **not the vendor invoice** for seat-based subscriptions. It is:
- A usage indicator for optimization
- Evidence of who is using what
- Input for right-sizing seat allocations

### Step-by-Step: Subscription Chargeback

**Step 1: Pull Seat Rosters**

Export from vendor admin consoles:
- ChatGPT Business/Enterprise: Admin → Members → Export
- Claude Team: Admin Console → Users → Export
- Copilot: GitHub Admin → Copilot → Usage

**Step 2: Map Seats to Cost Centers**

Create mapping table:

| User Email | Department | Team | Cost Center |
|------------|-----------|------|-------------|
| alice@company.com | Engineering | Platform | CC-12345 |
| bob@company.com | Data Science | ML | CC-67890 |

**Step 3: Calculate Allocations**

```sql
-- Example: Seat allocation calculation
-- (Run in your finance system, not gateway DB)
SELECT
  cost_center,
  COUNT(*) AS seats,
  seat_cost,
  COUNT(*) * seat_cost AS allocated_cost
FROM seat_roster
WHERE vendor = 'ChatGPT Enterprise'
  AND billing_period = '2026-01'
GROUP BY cost_center, seat_cost;
```

**Step 4: Validate with Usage Telemetry**

Cross-check seat assignments with actual usage:

```bash
# Export gateway logs (if subscription is routed through gateway)
make release-bundle

# Check if high-seat teams have usage to justify allocation
# Look for: seats assigned but no usage (optimization opportunity)
```

**Step 5: Generate Report**

Use the [Financial Showback/Chargeback Report Template](../templates/FINANCIAL_SHOWBACK_CHARGEBACK_REPORT.md):

- Section 1: Seat-based billing summary
- Section 2: Seat assignment by cost center
- Section 3: Usage telemetry highlights (showback context)
- Section 4: Reconciliation (vendor invoice vs internal records)

### Two-Bucket Reporting

For organizations with both API-key and subscription usage:

| Bucket | Billing Model | Chargeback Basis | Data Source |
|--------|---------------|------------------|-------------|
| **Usage-Based** | API keys | Token consumption | Gateway logs |
| **Seat-Based** | Subscriptions | Seat assignment | Vendor roster |

**Combined Report:**
```
Total AI Costs: $5,000
├── API-Key Usage: $3,000 (60%)
│   ├── Platform Engineering (CC-12345): $1,200
│   ├── Data Science (CC-67890): $1,050
│   └── Engineering (CC-54321): $750
└── Subscriptions: $2,000 (40%)
    ├── Platform Engineering (CC-12345): $800
    ├── Data Science (CC-67890): $700
    └── Engineering (CC-54321): $500
```

### Reconciliation Notes

**API-Key Mode:**
- Internal totals should closely match provider invoices (<5% variance)
- Variance sources: timing, currency conversion, rounding

**Subscription Mode:**
- Internal usage telemetry ≠ vendor invoice
- Vendor invoice is seat-based
- Usage telemetry is for optimization, not chargeback calculation

See [Financial Governance and Chargeback](../policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md) for complete reconciliation guidance.

---

## Related Documentation

- [Enterprise AI Control Plane Strategy](../ENTERPRISE_STRATEGY.md) - Strategic overview
- [API_KEY_GOVERNANCE_DEMO.md](API_KEY_GOVERNANCE_DEMO.md) - API-key enforcement details
- [FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md](../policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md) - Chargeback workflows and attribution model
- [NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md](NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md) - Bypass prevention and egress controls
- [SIEM_INTEGRATION.md](../security/SIEM_INTEGRATION.md) - Deep-dive SIEM integration
- [OTEL_SETUP.md](../observability/OTEL_SETUP.md) - OTEL collector configuration
- [../demo/README.md](../../demo/README.md) - Demo environment quick start
- [RUNBOOK.md](../RUNBOOK.md) - Operational troubleshooting

---

## References

- [LiteLLM Claude Code MAX Subscription Guide](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
- [LiteLLM ChatGPT Provider Documentation](https://docs.litellm.ai/docs/providers/chatgpt)
- [LiteLLM OpenAI Codex Tutorial](https://docs.litellm.ai/docs/tutorials/openai_codex)

---

## Appendix: Model Cost Reference

| Model | Approximate Cost per 1K Tokens | Use Case |
|-------|-------------------------------|----------|
| claude-haiku-4-5 | Very Low | Testing, high-volume tasks |
| openai-gpt5.2 | Low | General purpose |
| claude-sonnet-4-5 | Medium | Complex reasoning |
| gpt-4 | High | Advanced reasoning |

*Use cheaper models for testing to minimize costs.*
