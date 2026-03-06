# API-Key Governance Demonstration

**Executive Summary:** This document demonstrates how the AI Control Plane enforces governance controls through the API-key authentication path, providing centralized policy enforcement, audit logging, and budget management.

---

## Overview

The API-key governance demonstration showcases three pillars of AI governance:

1. **Model Allowlist Enforcement** - Only approved models can be accessed
2. **Budget and Rate Limit Controls** - Per-key spending limits and request throttling
3. **Comprehensive Audit Logging** - Full attribution of requests to principals

---

## Governance Controls

### 1. Model Allowlist Enforcement

**Configuration Location:** `demo/config/litellm.yaml`

**Approved Models:**

| Model Alias | Provider | Purpose |
|-------------|----------|---------|
| `openai-gpt5.2` | OpenAI | General purpose LLM access |
| `claude-sonnet-4-5` | Anthropic | Advanced reasoning tasks |
| `claude-haiku-4-5` | Anthropic | Fast, cost-effective tasks |

**Enforcement Mechanism:**
- Gateway validates model against per-key allowlist
- Requests to unapproved models are rejected with HTTP 400
- Model list is configurable per virtual key

**Evidence:**

```bash
# Generate key with restricted model list
curl -X POST http://localhost:4000/key/generate \
  -H "Authorization: Bearer $MASTER_KEY" \
  -d '{
    "key_alias": "restricted-key",
    "models": ["claude-haiku-4-5"]
  }'

# Request to approved model succeeds
# Request to unapproved model fails with "model not allowed"
```

---

### 2. Budget and Rate Limit Enforcement

**Global Settings (from `litellm.yaml`):**
- Global proxy budget: $100.00 per 30 days
- Per-user default budget: $10.00 per 7 days
- Default RPM: 60 requests per minute
- Default TPM: 90,000 tokens per minute

**Per-Key Configuration:**

```bash
# Generate key with specific limits
curl -X POST http://localhost:4000/key/generate \
  -H "Authorization: Bearer $MASTER_KEY" \
  -d '{
    "key_alias": "budget-limited-key",
    "max_budget": 0.50,
    "rpm_limit": 10,
    "models": ["openai-gpt5.2", "claude-haiku-4-5"]
  }'
```

**Enforcement Behavior:**
- Budget exhausted → Request blocked with "budget exceeded" error
- RPM exceeded → Request blocked with "rate limit exceeded" error
- Real-time tracking in LiteLLM-managed virtual key/budget tables (see `make db-status`)

---

### 3. Audit Logging

**Database Schema:** `"LiteLLM_SpendLogs"` (usage/cost metadata)

**Captured Fields:**

| Field | Description | Governance Purpose |
|-------|-------------|-------------------|
| `api_key` | Key token identifier (join to `"LiteLLM_VerificationToken".token` to resolve `key_alias`) | Attribution (join key) |
| `model` | Model accessed | Policy compliance |
| `spend` | Cost of request | Budget tracking |
| `prompt_tokens` | Input tokens | Usage monitoring |
| `completion_tokens` | Output tokens | Usage monitoring |
| `status` | success/fail | Policy enforcement evidence |
| `startTime` | Request timestamp | Timeline analysis |

**Query Examples:**

```sql
-- Audit trail for specific principal
 SELECT model, spend, "prompt_tokens", "completion_tokens", status
 FROM "LiteLLM_SpendLogs" s
 JOIN "LiteLLM_VerificationToken" v
   ON s.api_key = v.token
 WHERE v.key_alias = 'governance-demo-key'
 ORDER BY s."startTime" DESC;

-- Budget utilization per key
SELECT v.key_alias, v.spend, b.max_budget,
       ROUND((v.spend/NULLIF(b.max_budget,0)*100)::numeric, 2) as percent_used
FROM "LiteLLM_VerificationToken" v
JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id;
```

---

## Demo Scenarios

### Scenario-to-Goal Mapping

| Scenario | Goal | Notes |
|----------|------|-------|
| `4` | Budget and rate-limit enforcement | Deterministic limit behavior |
| `5` | Governance summary for management | Policy, audit, and budget proof points |
| `9` | Cursor governed path | IDE workflow through gateway |
| `11` | Chargeback/showback | User and department allocation outputs |

### Running the Governance Demo

```bash
# Run the comprehensive governance demonstration
make demo-scenario SCENARIO=5
```

### Scenario 5 Walkthrough

**Step 1: Configuration Display**
- Shows approved models from `litellm.yaml`
- Displays rate limits and budget settings
- Confirms security features enabled

**Step 2: Key Generation**

```bash
# Creates key with:
# - Budget: $0.50
# - RPM limit: 5
# - Models: [openai-gpt5.2, claude-sonnet-4-5, claude-haiku-4-5]
```

**Step 3: Approved Model Access**
- Tests access to `claude-haiku-4-5` (cheap model) ✓
- Tests access to `openai-gpt5.2` ✓
- Verifies requests succeed

**Step 4: Unapproved Model Blocking**
- Attempts access to `gpt-4` (not in allowlist)
- Expects HTTP 400 or "model not allowed" error
- Confirms policy enforcement

**Step 5: Audit Trail Verification**

```
Principal Identity: governance-demo-key
Model: claude-haiku-4-5
Spend: $0.000125
Tokens: 25 prompt, 15 completion
Status: success
```

**Step 6: Budget Tracking**

```
Key: governance-demo-key
Spend: $0.000250
Max Budget: $0.50
Percent Used: 0.05%
```

---

## Compliance Alignment

### Strategy Document Mapping

| Governance Control | Strategy Section | Implementation |
|-------------------|------------------|----------------|
| Model Allowlist | 4.1 Provider/Model Allowlisting | `litellm.yaml` model_list + per-key models array |
| Central Auth | 4.1 Central Authentication | Gateway validates all API keys |
| Centralized Logs | 4.1 Centralized Logging | PostgreSQL `"LiteLLM_SpendLogs"` (usage/cost metadata) |
| Evidence Pipeline | 4.2 Evidence Pipeline | key_alias → model → tokens → spend |
| Usage-Based Billing | 6.1 Usage-Based Billing | `"LiteLLM_VerificationToken"` + `"LiteLLM_BudgetTable"` |

### Evidence Artifacts

**For Compliance Audits:**
1. **Configuration Evidence:** `demo/config/litellm.yaml` shows approved models
2. **Policy Enforcement Logs:** Database shows blocked requests
3. **Attribution Records:** Every request linked to key_alias
4. **Budget Reports:** Join `"LiteLLM_VerificationToken"` + `"LiteLLM_BudgetTable"` for spend vs max_budget

---

## Management Presentation Guide

### Key Talking Points

**1. Centralized Control**
> "All AI access flows through the gateway. We control which models are available, who can access them, and how much they can spend—all from a single configuration point."

**2. Financial Governance**
> "Every API key has a budget cap. When the budget is exhausted, access stops. No surprise bills, no runaway costs."

**3. Complete Audit Trail**
> "We know exactly who used what model, when, and at what cost. This data feeds our SIEM for anomaly detection and compliance reporting."

**4. Security Integration**
> "The gateway integrates with our existing security infrastructure—OAuth for SSO, OpenTelemetry for observability, and database logging for forensics."

### Live Demo Script

```bash
# 1. Show configuration
cat demo/config/litellm.yaml | grep -A20 "model_list:"

# 2. Create a governance-controlled key
make key-gen ALIAS=management-demo BUDGET=0.25

# 3. Run the full governance demo
make demo-scenario SCENARIO=5

# 4. Show audit trail
make db-status
```

---

## Technical Reference

### Database Tables

**`LiteLLM_SpendLogs`** - Request-level usage/cost metadata

```sql
 SELECT 
   COALESCE(v.key_alias, 'unknown') as principal,
   model as model,
   COALESCE("prompt_tokens",0) + COALESCE("completion_tokens",0) as total_tokens,
   spend as cost_usd,
   status,
   "startTime" as timestamp
 FROM "LiteLLM_SpendLogs" s
 LEFT JOIN "LiteLLM_VerificationToken" v
   ON s.api_key = v.token
 ORDER BY "startTime" DESC
 LIMIT 10;
```

**`LiteLLM_VerificationToken` + `LiteLLM_BudgetTable`** - Budget tracking

```sql
 SELECT 
   v.key_alias,
   ROUND(v.spend::numeric, 4) as spent,
   ROUND(b.max_budget::numeric, 4) as budget,
   ROUND((v.spend/NULLIF(b.max_budget,0)*100)::numeric, 2) as pct_used
 FROM "LiteLLM_VerificationToken" v
 JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id
 WHERE b.max_budget > 0;
```

**`LiteLLM_VerificationToken`** - Key metadata

```sql
 SELECT 
   key_alias,
   max_budget,
   user_id,
   created_at
 FROM "LiteLLM_VerificationToken"
 WHERE key_alias IS NOT NULL;
```

### API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /key/generate` | Create virtual key with governance controls |
| `GET /v1/models` | List available models |
| `POST /v1/chat/completions` | Execute inference (enforces all policies) |

---

## Showback/Chargeback Walkthrough

This section demonstrates how to use the API-key governance model for internal chargeback.

### Step 1: Generate Keys with Attribution

Use the `key_alias` convention for cost-center mapping:

```bash
# Team-scoped key with cost center attribution
make key-gen ALIAS=team-platform__cc-12345 BUDGET=100.00 RPM=100

# Service account key
make key-gen ALIAS=svc-analytics__team-data__cc-67890 BUDGET=50.00 RPM=50

# User key (use employee ID, not email)
make key-gen ALIAS=usr-jdoe123__team-eng__cc-54321 BUDGET=25.00 RPM=25
```

### Step 2: Generate Usage Data

Run the governance demo to create sample traffic:

```bash
make demo-scenario SCENARIO=5
```

### Step 3: Query Spend by Principal

```sql
-- Top 10 principals by spend (monthly)
SELECT
  COALESCE(v.key_alias, 'unknown') AS principal,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  COUNT(*) AS request_count
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY COALESCE(v.key_alias, 'unknown')
ORDER BY SUM(s.spend) DESC
LIMIT 10;
```

### Step 4: Extract Team and Cost Center

```sql
-- Spend by cost center (chargeback allocation)
SELECT
  CASE
    WHEN v.key_alias LIKE '%__cc-%' THEN
      substring(v.key_alias from '__cc-([0-9]+)')
    ELSE 'unknown-cc'
  END AS cost_center,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN
      substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown-team'
  END AS team,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  COUNT(*) AS request_count
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY cost_center, team
ORDER BY SUM(s.spend) DESC;
```

### Step 5: Spend by Model + Team (Optimization)

```sql
-- Which teams use expensive models?
SELECT
  s.model,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN
      substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  COUNT(*) AS requests
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY s.model, team
ORDER BY SUM(s.spend) DESC;
```

### Step 6: Generate Governance Report

```bash
# Canonical evidence workflow
make db-status
make release-bundle
make release-bundle-verify

# Legacy scorecard commands are compatibility stubs in public snapshot
make chargeback-report
make chargeback-report OUTPUT_FORMAT=json
make chargeback-report REPORT_MONTH=YYYY-MM
```

### Key Takeaways

1. **Usage-based billing** (API keys) maps directly to token consumption
2. **Key aliases** enable attribution to teams/cost centers
3. **Gateway logs** provide granular spend tracking
4. **Reconciliation** compares internal totals to provider invoices

See [Financial Governance and Chargeback](../policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md) for complete chargeback workflow details.

---

## Key Compromise Response

This section demonstrates the rapid response workflow when an API key is suspected to be compromised.

### Detection

Compromised keys are typically detected through:

| Detection Rule | Indicator | Severity |
|----------------|-----------|----------|
| DR-001 | Non-approved model access attempts | High |
| DR-002 | Token usage spike outside normal patterns | Medium |
| DR-005 | Rapid request rate suggesting automation | Medium |
| DR-006 | Failed authentication attempts | High |

**Run detections to identify compromised keys:**

```bash
# Run detection rules and review findings
make detection
```

### Immediate Response (First 15 Minutes)

**Step 1: Identify affected key(s)**

```bash
# Review detection output for suspicious keys
make detection

# Check database for recent activity
make db-status
```

**Step 2: Revoke compromised key(s)**

```bash
# Revoke a single key by alias
make key-revoke ALIAS=<alias>

# Repeat as needed for all impacted aliases
```

**Step 3: Verify revocation**

```bash
# Verify key no longer appears in active keys
make db-status | grep compromised-key-alias
# Should show no results or "not found"

# Test authentication failure (optional)
curl -X POST http://localhost:4000/v1/chat/completions \
  -H "Authorization: Bearer $OLD_KEY" \
  -d '{"model": "gpt-5.2", "messages": [{"role": "user", "content": "test"}]}'
# Should return HTTP 401 Unauthorized
```

### Evidence Collection

Preserve forensics for incident documentation:

```bash
# 1. Capture detection findings
make detection > incident-YYYY-MM-DD-detections.txt

# 2. Query audit logs for affected key
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
  SELECT 
    TO_CHAR(s.\"startTime\", 'YYYY-MM-DD HH24:MI:SS') AS timestamp,
    v.key_alias, s.model, s.status, s.spend,
    s.\"prompt_tokens\" + s.\"completion_tokens\" AS total_tokens
  FROM \"LiteLLM_SpendLogs\" s
  JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE v.key_alias = 'compromised-key-alias'
  ORDER BY s.\"startTime\" DESC
  LIMIT 50;
" > incident-YYYY-MM-DD-audit-log.csv

# 3. Capture database state
make db-status > incident-YYYY-MM-DD-db-state.txt
```

### Key Rotation

After revoking the compromised key, issue replacement credentials:

```bash
# Generate replacement key
make key-gen ALIAS=replacement-key BUDGET=10.00

# Distribute new key to legitimate clients
# (application-specific deployment process)
```

### Demo Scenario

Run the automated scenario that demonstrates this entire workflow:

```bash
# Run scenario 8: Rapid Response Key Compromise
make demo-scenario SCENARIO=8
```

This scenario will:
1. Generate a test key
2. Simulate suspicious activity (non-approved model access)
3. Run detections to identify the compromise
4. Revoke the key
5. Verify the revocation
6. Show forensics evidence

### Complete Documentation

See [RUNBOOK.md section 9.6](../RUNBOOK.md#96-rapid-response-containment--key-lifecycle) for:
- Detailed rapid response procedures
- Dual-control / break-glass processes
- Emergency egress lockdown procedures
- Evidence preservation checklists

---

## Related Documentation

- [Enterprise AI Control Plane Strategy](../ENTERPRISE_STRATEGY.md) - Strategic overview
- [BUDGETS_AND_RATE_LIMITS.md](../policy/BUDGETS_AND_RATE_LIMITS.md) - Detailed budget configuration
- [FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md](../policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md) - Chargeback workflows and SQL recipes
- [DATABASE.md](../DATABASE.md) - Database schema and queries
- [demo/README.md](../../demo/README.md) - Demo environment quick start

---

## Appendix: Model Cost Reference

| Model | Approximate Cost per 1K Tokens | Use Case |
|-------|-------------------------------|----------|
| claude-haiku-4-5 | Very Low | Testing, high-volume tasks |
| openai-gpt5.2 | Low | General purpose |
| claude-sonnet-4-5 | Medium | Complex reasoning |

*Use cheaper models for testing to minimize costs.*
