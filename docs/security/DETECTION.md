# AI Control Plane - Detection Rules

## Overview

The AI Control Plane includes SIEM-style detection rules for identifying security anomalies, policy violations, and cost risks. These rules analyze LiteLLM-managed usage/cost logs (e.g., `"LiteLLM_SpendLogs"`) and related budget tables to surface actionable insights.

**Important:** In LiteLLM’s PostgreSQL schema, `"LiteLLM_SpendLogs"."api_key"` is a **token identifier** (not a human-friendly key alias). For operator-friendly attribution, join to `"LiteLLM_VerificationToken"` on `("LiteLLM_SpendLogs".api_key = "LiteLLM_VerificationToken".token)` and use `key_alias` (and `user_id` when populated).

## Rule Quality Status

Detection rules are explicitly classified in `demo/config/detection_rules.yaml` using:
- `operational_status`: `validated` or `example`
- `coverage_tier`: `decision-grade` or `demo`
- `expected_signal`: short statement of the exact behavior the rule should surface

| Rule ID | Name | Operational Status | Coverage Tier | Expected Signal (summary) |
|---------|------|--------------------|---------------|---------------------------|
| `DR-001` | Non-Approved Model Access | Validated | Decision-grade | Any non-approved model request in last 24h |
| `DR-002` | Token Usage Spike | Example | Demo | Key exceeds static 24h token threshold |
| `DR-003` | High Block/Error Rate | Validated | Decision-grade | Key >10% non-success over >=10 requests |
| `DR-004` | Budget Exhaustion Warning | Validated | Decision-grade | Key below 20% budget remaining |
| `DR-005` | Rapid Request Rate | Example | Demo | Key exceeds 60 requests/minute |
| `DR-006` | Failed Authentication Attempts | Validated | Decision-grade | Key with >=5 failures in 24h |
| `DR-007` | Budget Threshold Alert | Validated | Decision-grade | Key at or above 80% budget usage |
| `DR-008` | DLP Block Event Detected | Validated | Decision-grade | Guardrail/DLP block markers in status/response |
| `DR-009` | Repeated PII Submission Attempts | Example | Demo | Repeated DLP-style failures by same key |

## Detection Rules

### DR-001: Non-Approved Model Access

**Severity:** High
**Category:** Policy Violation
**Enabled:** Yes

Detects requests to models not in the approved list configured in `litellm.yaml`. This helps ensure AI usage complies with organizational governance policies.

**What it checks:**
- Requests in the last 24 hours to models not explicitly approved
- Compares `model_id` against the approved list from `demo/config/litellm.yaml` `model_list`

**Typical findings:**
- Developers using non-approved models for testing
- Misconfigured model aliases
- Intentional policy bypass attempts

**Remediation:**
- Add the model to the approved list in `litellm.yaml` if usage is legitimate
- Revoke the virtual key if usage is unauthorized
- Review the user's access permissions

**SQL Query:**
```sql
-- DR-001 uses psql variables for secure parameterization.
-- The approved models list is passed via APPROVED_MODELS_JSON variable
-- and expanded server-side using jsonb_array_elements_text().
-- This prevents SQL injection that could occur with string interpolation.

		SELECT
		  s.request_id,
		  s.model AS model_id,
		  COALESCE(v.key_alias, 'unknown') AS key_alias,
		  CASE
		    WHEN NULLIF(BTRIM(s."user"), '') IS NOT NULL
		      AND LOWER(BTRIM(s."user")) <> 'unknown'
		      AND BTRIM(s."user") !~ '\s'
		      THEN BTRIM(s."user")
		    WHEN NULLIF(BTRIM(v.user_id), '') IS NOT NULL
		      AND LOWER(BTRIM(v.user_id)) <> 'unknown'
		      AND BTRIM(v.user_id) !~ '\s'
		      THEN BTRIM(v.user_id)
		    WHEN NULLIF(BTRIM(v.key_alias), '') IS NOT NULL
		      AND LOWER(BTRIM(v.key_alias)) <> 'unknown'
		      AND BTRIM(v.key_alias) !~ '\s'
		      THEN BTRIM(v.key_alias)
		    ELSE 'unknown'
		  END AS user_id,
		  TO_CHAR(s."startTime", 'YYYY-MM-DD HH24:MI:SS') AS timestamp,
		  s.status
		FROM "LiteLLM_SpendLogs" s
		LEFT JOIN "LiteLLM_VerificationToken" v
		  ON s.api_key = v.token
		WHERE s.model NOT IN (SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb))
		  AND s."startTime" > NOW() - INTERVAL '24 hours'
		ORDER BY s."startTime" DESC
		LIMIT 100;
	```

**Security Note:** The `:'APPROVED_MODELS_JSON'` syntax is a psql variable that is safely quoted as a SQL literal by the psql client. The `jsonb_array_elements_text()` function expands the JSON array server-side, eliminating SQL injection risks from user-controlled model names in `litellm.yaml`.

---

### DR-002: Token Usage Spike

**Severity:** Medium
**Category:** Anomaly Detection
**Enabled:** Yes

Detects unusual token consumption patterns per virtual key. Spikes may indicate compromised keys, abuse, or legitimate heavy usage.

**What it checks:**
- Total tokens consumed (prompt + completion) per key in the last 24 hours
- Flags keys exceeding 100,000 tokens

**Typical findings:**
- Automated scripts making excessive API calls
- Compromised keys being used by external actors
- Legitimate batch processing workloads

**Remediation:**
- Verify the identity of the key holder
- Review request patterns for signs of automation
- Revoke and re-issue the key if compromised

**SQL Query:**
```sql
		SELECT
		  COALESCE(v.key_alias, 'unknown') AS key_alias,
		  SUM(s."prompt_tokens" + s."completion_tokens") AS total_tokens,
		  COUNT(*) AS request_count,
		  TO_CHAR(MAX(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS last_seen
		FROM "LiteLLM_SpendLogs" s
		LEFT JOIN "LiteLLM_VerificationToken" v
		  ON s.api_key = v.token
		WHERE s."startTime" > NOW() - INTERVAL '24 hours'
		GROUP BY COALESCE(v.key_alias, 'unknown')
		HAVING SUM(s."prompt_tokens" + s."completion_tokens") > 100000
		ORDER BY total_tokens DESC;
	```

**Threshold configuration:**
- Modify `threshold_tokens` in `detection_rules.yaml` (default: 100,000)

---

### DR-003: High Block/Error Rate

**Severity:** Medium
**Category:** Availability
**Enabled:** Yes

Detects virtual keys with elevated failure rates. High error rates may indicate authentication issues, rate limiting, or provider problems.

**What it checks:**
- Error rate (non-success status) per key in the last 24 hours
- Minimum of 10 requests required
- Flags keys with >10% error rate

**Typical findings:**
- Expired or invalid API keys
- Rate limiting by upstream providers
- Network connectivity issues
- Malformed requests from client applications

**Remediation:**
- Verify provider API keys are valid and not expired
- Check for rate limiting on the provider account
- Review client application logs for request formatting issues
- Verify network connectivity to upstream providers

**SQL Query:**
```sql
		SELECT
		  COALESCE(v.key_alias, 'unknown') AS key_alias,
		  COUNT(*) FILTER (WHERE s.status != 'success') AS error_count,
		  COUNT(*) AS total_requests,
		  ROUND(100.0 * COUNT(*) FILTER (WHERE s.status != 'success') / NULLIF(COUNT(*), 0), 2) AS error_rate
		FROM "LiteLLM_SpendLogs" s
		LEFT JOIN "LiteLLM_VerificationToken" v
		  ON s.api_key = v.token
		WHERE s."startTime" > NOW() - INTERVAL '24 hours'
		GROUP BY COALESCE(v.key_alias, 'unknown')
		HAVING COUNT(*) >= 10
		  AND COUNT(*) FILTER (WHERE s.status != 'success') * 100.0 / NULLIF(COUNT(*), 0) > 10
		ORDER BY error_rate DESC;
	```

**Threshold configuration:**
- Modify `error_threshold_percent` in `detection_rules.yaml` (default: 10)
- Modify `min_requests` in `detection_rules.yaml` (default: 10)

---

### DR-004: Budget Exhaustion Warning

**Severity:** Low
**Category:** Cost Management
**Enabled:** Yes

Alerts when virtual keys approach their budget limits. This helps prevent service interruptions and enables proactive cost management.

**What it checks:**
- Remaining budget as percentage of maximum budget
- Flags keys with less than 20% remaining

**Typical findings:**
- Keys approaching their configured spend limits
- Potential for service interruption if budget is exhausted

**Remediation:**
- Increase the budget limit if usage is legitimate
- Review usage patterns to identify optimization opportunities
- Generate a new key if immediate access is needed

**SQL Query:**
```sql
	SELECT
	  v.key_alias,
	  ROUND(v.spend::numeric, 4) AS spent,
	  ROUND(b.max_budget::numeric, 4) AS maximum,
	  ROUND(((b.max_budget - v.spend) / NULLIF(b.max_budget, 0) * 100)::numeric, 2) AS percent_remaining
	FROM "LiteLLM_VerificationToken" v
	JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id
	WHERE b.max_budget > 0
	  AND ((b.max_budget - v.spend) / NULLIF(b.max_budget, 0)) < 0.2
	ORDER BY percent_remaining ASC;
```

**Threshold configuration:**
- Modify `warning_threshold_percent` in `detection_rules.yaml` (default: 80)

---

### DR-005: Rapid Request Rate

**Severity:** Medium
**Category:** Anomaly Detection
**Enabled:** Yes

Detects suspiciously high request frequency per virtual key. Rapid request rates may indicate automated abuse or compromised keys.

**What it checks:**
- Requests per minute per key in the last hour
- Flag any minute with more than 60 requests

**Typical findings:**
- Compromised keys being used by automated scripts
- Legitimate high-frequency workloads (e.g., batch processing)
- Application bugs causing request loops

**Remediation:**
- Investigate the source of rapid requests
- Revoke compromised keys
- Implement rate limiting in LiteLLM configuration
- Work with key owners to understand usage patterns

**SQL Query:**
```sql
		SELECT
		  COALESCE(v.key_alias, 'unknown') AS key_alias,
		  COUNT(*) AS request_count,
		  TO_CHAR(MIN(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS first_request,
		  TO_CHAR(MAX(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS last_request
		FROM "LiteLLM_SpendLogs" s
		LEFT JOIN "LiteLLM_VerificationToken" v
		  ON s.api_key = v.token
		WHERE s."startTime" > NOW() - INTERVAL '1 hour'
		GROUP BY COALESCE(v.key_alias, 'unknown'), DATE_TRUNC('minute', s."startTime")
		HAVING COUNT(*) > 60
		ORDER BY request_count DESC
		LIMIT 50;
	```

**Threshold configuration:**
- Modify `max_requests_per_minute` in `detection_rules.yaml` (default: 60)

---

### DR-006: Failed Authentication Attempts

**Severity:** High
**Category:** Security
**Enabled:** Yes

Detects repeated authentication failures. Multiple failed attempts may indicate brute force attacks or compromised credentials.

**What it checks:**
- Failed authentication attempts per key in the last 24 hours
- Flags keys with 5 or more failures

**Typical findings:**
- Brute force attacks against the gateway
- Expired or invalid virtual keys being used
- Configuration errors in client applications

**Remediation:**
- Revoke keys with repeated failures
- Investigate source IP addresses of failed attempts
- Implement IP-based blocking for repeated failures
- Review key distribution and rotation policies

**SQL Query:**
	```sql
		SELECT
		  COALESCE(v.key_alias, 'unknown') AS key_alias,
		  COUNT(*) FILTER (WHERE s.status = 'failure') AS failed_attempts,
		  COUNT(*) AS total_attempts,
		  TO_CHAR(MAX(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS last_attempt
		FROM "LiteLLM_SpendLogs" s
		LEFT JOIN "LiteLLM_VerificationToken" v
		  ON s.api_key = v.token
		WHERE s."startTime" > NOW() - INTERVAL '24 hours'
		GROUP BY COALESCE(v.key_alias, 'unknown')
		HAVING COUNT(*) FILTER (WHERE s.status = 'failure') >= 5
		ORDER BY failed_attempts DESC;
		```

**Threshold configuration:**
- Modify `max_failures` in `detection_rules.yaml` (default: 5)

---

### DR-007: Budget Threshold Alert

**Severity:** Medium
**Category:** Cost Management
**Enabled:** Yes

Alerts when keys exceed configurable budget usage thresholds (≥80% spent).

**What it checks:**
- Current spend vs max budget per key
- Flags keys at ≥80% of budget limit

**Typical findings:**
- Keys approaching budget exhaustion
- Potential for service interruption

**Remediation:**
- Review usage and increase budget or investigate patterns

**SQL Query:**
```sql
SELECT
  v.key_alias,
  ROUND(v.spend::numeric, 4) AS spend,
  ROUND(b.max_budget::numeric, 4) AS max_budget,
  ROUND((v.spend / NULLIF(b.max_budget, 0) * 100)::numeric, 2) AS percent_used
FROM "LiteLLM_VerificationToken" v
JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id
WHERE b.max_budget > 0
  AND (v.spend / NULLIF(b.max_budget, 0)) >= 0.8
ORDER BY percent_used DESC;
```

---

### DR-008: DLP Block Event Detected

**Severity:** High
**Category:** Security
**Enabled:** Yes

Detects requests blocked by Presidio DLP guardrails due to PII or sensitive data.

**What it checks:**
- Requests in the last 24 hours with failure status
- Response content indicating guardrail/Presidio/blocked keywords

**Typical findings:**
- Users accidentally sending PII to LLMs
- Attempts to share credentials or sensitive data
- Policy violations caught by content scanning

**Remediation:**
- Investigate blocked requests
- Educate users on safe data handling
- Review key access if malicious intent suspected

**Auto-Response:** Disabled (alert only)

---

### DR-009: Repeated PII Submission Attempts

**Severity:** High
**Category:** Security
**Enabled:** Yes
**Auto-Response:** Enabled (suspend_key after 10 min grace period)

Detects keys with multiple DLP blocks indicating persistent PII exposure attempts.

**What it checks:**
- 3 or more DLP blocks within 1 hour
- Same key attempting PII submission repeatedly

**Typical findings:**
- Compromised credentials being used for data exfiltration
- Insider threats attempting to leak sensitive data
- Broken applications repeatedly sending PII

**Remediation:**
- Key is auto-suspended after threshold
- Investigate source and intent
- Rotate credentials if compromised

---

### DR-010: Potential Prompt Injection Attempt

**Severity:** Medium
**Category:** Policy Violation
**Enabled:** Yes

Detects requests with patterns associated with prompt injection or jailbreak attempts.

**What it checks:**
- Failed requests containing injection-related keywords
- Patterns: "ignore previous", "jailbreak", "prompt injection", "developer mode", "dAN"

**Typical findings:**
- Attempts to bypass safety filters
- Jailbreak attempts on AI models
- Testing of security boundaries

**Remediation:**
- Review request content
- Consider additional content filtering
- Restrict key permissions if abuse confirmed

---

### Guardrail Capability Map (LiteLLM Native vs Presidio)

The AI Control Plane uses both native LiteLLM guardrails and Presidio, with different roles:

| Capability | LiteLLM Native Guardrails | Presidio Guardrails |
|---|---|---|
| In-memory content filtering | Yes (`litellm_content_filter`) | No |
| Prompt injection detection | Yes (`prompt_injection_detection`) | No |
| Deterministic PII/entity detection | Limited | Yes |
| Custom organization-specific recognizers | No | Yes |
| Policy actions (BLOCK/MASK/REDACT/ALLOW) | Provider-dependent | Yes (entity-config driven) |

Reference docs:
- <https://docs.litellm.ai/docs/proxy/guardrails/quick_start>
- <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/litellm_content_filter>
- <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/prompt_injection_detection>
- <https://docs.litellm.ai/docs/proxy/guardrails/ai_guardrails/presidio>

### Guardrail Lifecycle Control Points

Guardrail controls are implemented and operated across three stages:

| Stage | Goal | Typical Controls | Evidence Path |
|---|---|---|---|
| Pre-call | Prevent unsafe prompts before model invocation | Presidio DLP, prompt-injection filters, secret detection hooks | Gateway status/response fields + detection rules |
| In-call | Constrain runtime behavior during generation/tool execution | Tool allowlists, response schema checks, parameter constraints | Gateway policy logs + runbook events |
| Post-call | Improve quality and response readiness | Detection tuning, false-positive review, exception handling, SIEM correlation | Detection outputs, monthly governance reports |

Use pre-call controls for deterministic blocking/masking, in-call controls for constrained execution, and post-call workflows for continuous policy tuning.

### Content-Based DLP Detection (Presidio)

For deterministic entity-level DLP, the AI Control Plane uses Microsoft Presidio. This enables blocking, masking, or redacting sensitive data before it reaches LLM providers.

**How It Works:**

1. **Presidio Analyzer** scans request content for PII entities (SSN, credit cards, AWS keys, etc.)
2. **Presidio Anonymizer** applies configured actions (BLOCK, MASK, REDACT)
3. **LiteLLM Guardrails** enforce policies in "pre_call" mode (before LLM request)
4. **Detection Rules** (DR-008 through DR-010) monitor and alert on DLP events

**Configured PII Entities:**

| Entity Type | Action | Description |
|-------------|--------|-------------|
| US_SSN | BLOCK | US Social Security Numbers |
| US_PASSPORT | BLOCK | US Passport numbers |
| US_DRIVER_LICENSE | BLOCK | Driver's license numbers |
| US_BANK_NUMBER | BLOCK | Bank account numbers |
| US_ITIN | BLOCK | US Individual Taxpayer ID Numbers |
| CREDIT_CARD | BLOCK | Credit card numbers |
| CRYPTO | BLOCK | Cryptocurrency wallet addresses |
| IBAN_CODE | BLOCK | International Bank Account Numbers |
| AWS_ACCESS_KEY | BLOCK | AWS Access Key IDs |
| EMAIL_ADDRESS | MASK | Email addresses (masked) |
| PHONE_NUMBER | MASK | Phone numbers (masked) |
| LOCATION | MASK | Location/address information |
| PERSON | MASK | Person names |

### Custom PII Recognizers

Organizations can extend Presidio's detection capabilities with custom recognizers for internal data formats.

**Supported Custom Entities:**

| Entity | Format | Action | Use Case |
|--------|--------|--------|----------|
| ACP_EMPLOYEE_ID | EMP-XXXXXX | BLOCK | Employee identification codes |
| ACP_CUSTOMER_ACCOUNT | CUST-XXXXXXXX | BLOCK | Customer account numbers |
| ACP_SYSTEM_KEY | ACPKEY-XXXX-XXXX-XXXX | BLOCK | Internal system credentials |
| ACP_PROJECT_CODE | PROJ-XXXX-XXXX | MASK | Internal project references |

**Adding Custom Recognizers:**

See `docs/demos/CUSTOM_PII_RECOGNIZERS.md` for the complete guide on creating and deploying organization-specific PII patterns.

**Configuration Files:**
- Recognizers: `demo/config/presidio/recognizers/custom_recognizers.yaml`
- Actions: `demo/config/litellm.yaml` (pii_entities_config section)

**Detection Rules for Content Analysis:**

| Rule ID | Name | Severity | Description |
|---------|------|----------|-------------|
| DR-008 | DLP Block Event | High | Detects requests blocked by Presidio |
| DR-009 | Repeated PII Submission | High | Multiple blocks from same key (suspicious) |
| DR-010 | Prompt Injection Attempt | Medium | Patterns suggesting jailbreak attempts |

**Testing DLP:**

Run the updated Scenario 6 to test Presidio blocking:

```bash
make up
make demo-scenario SCENARIO=6
```

Or test manually:

```bash
# Generate a test key
make key-gen ALIAS=dlp-test BUDGET=5.00

# Test with PII (will be blocked)
curl -X POST http://127.0.0.1:4000/v1/chat/completions \
  -H "Authorization: Bearer $TEST_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-haiku-4-5",
    "messages": [{"role": "user", "content": "My SSN is 123-45-6789"}]
  }'

# Expected response: 400/403 with "blocked" message
```

**Architecture:**

```
User Request → LiteLLM Proxy → Presidio Analyzer → [BLOCK/MASK/ALLOW] → LLM Provider
                    ↓
              PostgreSQL Logs → Detection Rules → Alerts/Auto-Response
```

**For Advanced Configuration:**

Edit `demo/config/litellm.yaml` to customize:
- Additional PII entity types
- Action mappings (BLOCK vs MASK vs ALLOW)
- Custom blocked messages
- Per-model guardrail configuration

---

## Presidio Configuration Reference

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PRESIDIO_ANALYZER_URL` | Presidio Analyzer endpoint | `http://presidio-analyzer:5000` |
| `PRESIDIO_ANONYMIZER_URL` | Presidio Anonymizer endpoint | `http://presidio-anonymizer:5001` |

### LiteLLM Guardrail Configuration

Full configuration in `demo/config/litellm.yaml`:

```yaml
guardrails:
  - guardrail_name: "presidio-dlp-guardian"
    litellm_params:
      guardrail: presidio
      mode: "pre_call"  # Always pre_call for DLP
      presidio_analyzer_api_base: "http://presidio-analyzer:3000"
      presidio_anonymizer_api_base: "http://presidio-anonymizer:3000"
      pii_entities_config:
        # Format: ENTITY_TYPE: ACTION
        US_SSN: "BLOCK"
        EMAIL_ADDRESS: "MASK"
        PERSON: "MASK"
      default_on: true  # Apply to all requests
      blocked_message: "Custom message for blocked requests"
```

### Supported Entity Types

Presidio supports 50+ built-in entity types including:
- **Financial:** CREDIT_CARD, US_BANK_NUMBER, IBAN_CODE, CRYPTO
- **Credentials:** Custom recognizers required for GITHUB_TOKEN, Azure credentials, GCP keys, etc. AWS_ACCESS_KEY is now included by default.
- **PII:** US_SSN, US_PASSPORT, US_DRIVER_LICENSE, US_ITIN, PERSON
- **Contact:** EMAIL_ADDRESS, PHONE_NUMBER, LOCATION
- **Healthcare:** US_MEDICARE, US_MEDICAID

**Note:** Cloud provider tokens (AWS, GCP, Azure) and API keys require custom recognizers.
See [Microsoft Presidio Documentation](https://microsoft.github.io/presidio/) for details on
adding custom recognizers.

See [Microsoft Presidio Documentation](https://microsoft.github.io/presidio/) for full list.

### Action Types

| Action | Behavior | Use Case |
|--------|----------|----------|
| `BLOCK` | Request rejected; HTTP 400/403 returned | High-risk PII (SSN, credentials) |
| `MASK` | Entity replaced with `<ENTITY_TYPE>` | Medium risk (emails, phones) |
| `REDACT` | Entity removed entirely | Alternative to masking |
| `ALLOW` | Request proceeds; entity logged | Low risk (person names) |

---

## Running Detection Queries

### Basic Usage

From the project root, run detection validation:

```bash
# Canonical PR/ops entrypoint
make validate-detections

# Compatibility alias
make detection

# Typed entrypoint (supports verbose diagnostics)
./scripts/acpctl.sh validate detections --verbose
```

### Advanced Options

Public-snapshot command surface supports full-rule validation only. Rule-scoped execution, severity filtering, and JSON-stream output are not exposed via canonical entrypoints.

For automation:
- Use `make validate-detections` for CI-friendly pass/fail checks.
- Use `./scripts/acpctl.sh validate detections --verbose` when you need detailed diagnostics.
- Keep `make detection` and `make detection-normalized` as compatibility aliases for `make validate-detections`.

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `DB_NAME` | PostgreSQL database name | `litellm` |
| `DB_USER` | PostgreSQL user | `litellm` |
| `DETECTION_RULES` | Path to rules YAML | `demo/config/detection_rules.yaml` |
| `VERBOSE` | Enable verbose output | `0` |

---

## Interpreting Results

### Response Format

Detection results are displayed as:

```
=== DR-001: Non-Approved Model Access ===
  Description: Detects requests to models not in the approved list
  Category: policy_violation
  Severity: [HIGH]

  request_id   | model_id     | key_alias | user_id | timestamp            | status
  -------------+--------------+-----------+---------+----------------------+--------
  abc123-def   | gpt-4-turbo  | dev-key   | alice   | 2026-01-29 10:15:23  | success
  xyz789-ghi   | claude-3-5   | test-key  | bob     | 2026-01-29 09:45:12  | success

  Remediation: Add model to approved list in litellm.yaml or revoke key access
```

### Severity Indicators

- **[HIGH]** - Requires immediate attention, potential security incident
- **[MED]** - Investigate promptly, may indicate issues
- **[LOW]** - Monitor for trends, no immediate action required

### Exit Codes

`make validate-detections` is a CI-style wrapper and returns:

| Exit Code | Meaning |
|-----------|---------|
| **0** | Validation passed |
| **1** | Validation failed (domain/prereq/runtime wrapped as failure) |

For automation that needs typed exit-code granularity, call the typed entrypoint directly:

| Exit Code | Meaning |
|-----------|---------|
| **0** | Validation passed |
| **1** | Domain validation failure |
| **2** | Prerequisites not ready |
| **3** | Runtime/internal error |
| **64** | Usage error |

```bash
make validate-detections
code=$?

if [ "$code" -ne 0 ]; then
	case "$code" in
	1) echo "Detection rule validation failed" >&2 ;;
	2) echo "Detection prerequisites not ready" >&2 ;;
	3) echo "Detection runtime error occurred" >&2 ;;
	64) echo "Detection usage error" >&2 ;;
  esac
fi
```

### False Positives

Detection rules use threshold-based heuristics. False positives may occur in these scenarios:

1. **Token spikes** during legitimate batch processing
2. **High error rates** during provider outages
3. **Rapid request rates** from legitimate automated tools
4. **Budget warnings** for keys intentionally set with low limits

Always investigate context before taking remedial action. Consider tuning thresholds in `detection_rules.yaml` for your environment.

---

## Detection Output Contract (Public Snapshot)

Public-snapshot detection entrypoints (`make validate-detections`, `make validate-siem-queries`, and `make validate-siem-schema`) emit human-readable validation output and exit codes; they do not emit the legacy structured JSON findings payload.

For machine-parsed evidence in this snapshot:
- Use the normalized evidence feed: `demo/logs/normalized/evidence.jsonl`
- Use SIEM mappings in `demo/config/siem_queries.yaml`
- Capture validator evidence with `make validate-detections 2>&1 | tee detection-validation.log`

---

## Evaluating Against Normalized Evidence

### Overview

Detection rules can be evaluated against the **unified evidence feed** (`demo/logs/normalized/evidence.jsonl`) instead of querying the PostgreSQL database directly. This enables:

- **Cross-source detection**: Detect anomalies across gateway, OTEL, and compliance API sources
- **Offline analysis**: Run detections on exported evidence files
- **Consistent schema**: All evidence uses the normalized schema regardless of source

### Data Source Comparison

| Aspect | PostgreSQL Source | Normalized Source |
|--------|-------------------|-------------------|
| Data freshness | Real-time | Based on last evidence export |
| Sources covered | Gateway only | Gateway + OTEL + Compliance |
| Budget rules (DR-004, DR-007) | ✅ Supported | ❌ Not available |
| Required tools | Docker + PostgreSQL | jq only |
| Performance | Fast (SQL) | Moderate (JSONL processing) |

### Running Against Normalized Evidence

```bash
# First, ensure evidence is exported and merged
make release-bundle

# Run detections against normalized evidence
make validate-detections

# Compatibility alias (same behavior as validate-detections)
make detection-normalized

# For SIEM/automation pipelines, consume normalized evidence directly
head -n 20 demo/logs/normalized/evidence.jsonl
```

### Rule Support Matrix

| Rule | PostgreSQL | Normalized | Notes |
|------|------------|------------|-------|
| DR-001: Non-Approved Model Access | ✅ | ✅ | Uses `ai.model.id` and `policy.action` |
| DR-002: Token Usage Spike | ✅ | ✅ | Aggregates `ai.tokens.total` per principal |
| DR-003: High Block/Error Rate | ✅ | ✅ | Calculates error rate from `policy.action` |
| DR-004: Budget Exhaustion Warning | ✅ | ❌ | Requires budget table (database only) |
| DR-005: Rapid Request Rate | ✅ | ✅ | Groups by principal and minute |
| DR-006: Failed Authentication Attempts | ✅ | ✅ | Counts `policy.action = "error"` |
| DR-007: Budget Threshold Alert | ✅ | ❌ | Requires budget table (database only) |
| DR-008: DLP Block Event Detected | ✅ | ⚠️ | Uses `content_analysis` fields (gateway only) |
| DR-009: Repeated PII Submission Attempts | ✅ | ⚠️ | Aggregation requires gateway logs |
| DR-010: Potential Prompt Injection Attempt | ✅ | ⚠️ | Uses response analysis (gateway only) |

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `NORMALIZED_EVIDENCE` | Path to evidence JSONL file | `demo/logs/normalized/evidence.jsonl` |

### Example Output

`make validate-detections` prints operator-focused status output and exits with the standard contract (`0` success, `1` findings/domain failure, `2` prerequisites, `3` runtime, `64` usage).

```bash
$ make validate-detections
Validating detection rules...
✓ Detection validation passed
```

---

## Integrating with External SIEM

### Exporting to SIEM

Use the normalized evidence feed as the structured SIEM payload source:

```bash
# Refresh unified evidence feed
make release-bundle
make release-bundle-verify

# Send normalized evidence to Splunk HEC
curl -s http://splunk-hec:8088/services/collector/raw \
  -H "Authorization: Splunk <TOKEN>" \
  -H "Content-Type: application/json" \
  --data-binary @demo/logs/normalized/evidence.jsonl

# Send normalized evidence to Elasticsearch
curl -s -X POST "https://elastic:9200/ai-control-plane-evidence/_bulk" \
  -H "Content-Type: application/x-ndjson" \
  --data-binary @demo/logs/normalized/evidence.jsonl
```

### Scheduled Scans

Use cron to run detections on a schedule:

```crontab
# Run detection every hour
0 * * * * cd /path/to/ai-control-plane && make validate-detections

# Run full detection daily at 9 AM
0 9 * * * cd /path/to/ai-control-plane && make validate-detections
```

### Webhook Configuration

For real-time alerting, configure webhooks in your monitoring stack:

```bash
# Example: Send to Slack on domain validation findings
make validate-detections
code=$?
if [ "$code" -eq 1 ]; then
  curl -X POST -H 'Content-type: application/json' \
	--data '{"text":"AI Control Plane detection findings require review"}' \
    https://hooks.slack.com/services/YOUR/WEBHOOK/URL
fi
```

---

## Real-Time Webhook Alerting

The detection system now supports automatic webhook notifications when rules trigger. This enables immediate alerting to Slack, PagerDuty, or custom endpoints without requiring external polling.

### Configuration

Enable alerting by setting environment variables before running detections:

```bash
# Configure webhook URLs
export SLACK_SECURITY_WEBHOOK_URL="https://hooks.slack.com/services/XXX/YYY/ZZZ"
export SLACK_ALERTS_WEBHOOK_URL="https://hooks.slack.com/services/XXX/YYY/ZZZ"
export PAGERDUTY_ROUTING_KEY="your-routing-key-here"

# Enable notifications
export NOTIFICATIONS_ENABLED=1
```

Alternatively, edit `demo/config/detection_rules.yaml` to enable the alert_routing section:

```yaml
alert_routing:
  enabled: true
  
  deduplication:
    enabled: true
    window_minutes: 5
    cache_ttl_hours: 24
  
  routes:
    high:
      - channel: slack
        webhook_url: "${SLACK_SECURITY_WEBHOOK_URL}"
        mention: "@channel"
        color: "danger"
      - channel: pagerduty
        routing_key: "${PAGERDUTY_ROUTING_KEY}"
        severity: "critical"
    
    medium:
      - channel: slack
        webhook_url: "${SLACK_ALERTS_WEBHOOK_URL}"
        mention: "@here"
        color: "warning"
    
    low:
      - channel: slack
        webhook_url: "${SLACK_NOTIFICATIONS_WEBHOOK_URL}"
        color: "#808080"
  
  generic_webhooks:
    - name: "siem_integration"
      url: "${GENERIC_WEBHOOK_URL}"
      headers:
        Authorization: "Bearer ${GENERIC_WEBHOOK_TOKEN}"
      filter_severity: ["high", "medium"]
```

**Note:** Only the `enabled` field is read from YAML. Webhook URLs and routing details must be configured via environment variables as shown above.

### Running with Notifications

```bash
# Run detection validation with notification env vars configured
make validate-detections

# Typed entrypoint equivalent (verbose diagnostics)
./scripts/acpctl.sh validate detections --verbose

# Capture output for downstream alert routing
make validate-detections 2>&1 | tee detections.log
```

### Environment Variables

| Variable | Purpose | Severity | Default |
|----------|---------|----------|---------|
| `SLACK_SECURITY_WEBHOOK_URL` | Slack webhook for high-severity alerts | High | (none) |
| `SLACK_ALERTS_WEBHOOK_URL` | Slack webhook for medium-severity alerts | Medium | (none) |
| `SLACK_NOTIFICATIONS_WEBHOOK_URL` | Slack webhook for low-severity alerts | Low | (none) |
| `PAGERDUTY_ROUTING_KEY` | PagerDuty Events API v2 routing key | High | (none) |
| `GENERIC_WEBHOOK_URL` | Generic webhook for SIEM integration | Medium+ | (none) |
| `GENERIC_WEBHOOK_TOKEN` | Bearer token for generic webhook auth | Medium+ | (none) |
| `NOTIFICATIONS_ENABLED` | Master switch ("1"/"true" to enable, "0"/"false" to disable) | All | auto-detected |
| `NOTIFICATION_DEDUP_WINDOW` | Deduplication window in minutes | All | 5 |
| `NOTIFICATION_CACHE_TTL` | Cache TTL in hours | All | 24 |
| `NOTIFICATION_DEDUP_CACHE` | Path to dedup cache file | All | `demo/logs/.alert_dedup_cache` |

**Note:** Notifications are automatically enabled when any webhook URL is configured, even without `NOTIFICATIONS_ENABLED`. Set `NOTIFICATIONS_ENABLED=0` to explicitly disable.

### Deduplication

To prevent alert storms, the dispatcher deduplicates alerts based on:
- Rule ID
- Key alias (or "global" if not applicable)
- Time bucket (5-minute windows by default)

This means if DR-001 triggers 10 times in 5 minutes for the same key, only one alert is sent. Configure the window via:

```bash
export NOTIFICATION_DEDUP_WINDOW=10  # 10-minute dedup window
```

### Supported Channels

| Channel | Severity | Description |
|---------|----------|-------------|
| Slack | High/Medium/Low | Rich formatted alerts with @channel/@here mentions |
| PagerDuty | High | Critical alerts via Events API v2 |
| Generic Webhook | Medium+ | Custom SIEM integration endpoint |

---

## Automated Response Configuration

The detection system supports automated key suspension for high-severity rules, enabling SOAR-style containment with break-glass safety mechanisms. This closes the detection-to-response loop, reducing Mean Time To Response (MTTR) from minutes/hours to seconds for critical security events.

### Configuration

Enable auto-response by adding an `auto_response` block to detection rules in `demo/config/detection_rules.yaml`:

```yaml
- rule_id: DR-001
  name: "Non-Approved Model Access"
  severity: "high"
  # ... other fields ...
  auto_response:
    enabled: true
    action: "suspend_key"
    grace_period_minutes: 0
```

**Auto-Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Whether auto-response is active for this rule |
| `action` | string | Action to take: `suspend_key`, `quarantine_key`, or `alert_only` |
| `grace_period_minutes` | int | Delay before action (for transient failure handling) |

### Default Configuration

The following rules have auto-response enabled by default:

| Rule ID | Severity | Action | Grace Period |
|---------|----------|--------|--------------|
| DR-001 | High | suspend_key | 0 min (immediate) |
| DR-006 | High | suspend_key | 5 min (allows transient failures) |

### Break-Glass Override

To disable all auto-responses during incidents or maintenance:

```bash
# Option 1: Environment variable
export ACP_DISABLE_AUTO_RESPONSE=1

# Run detection (auto-response will be skipped)
make validate-detections

# Option 2: Command-line flag
make validate-detections
```

### Grace Periods and Debouncing

**Grace Periods:**
- Set `grace_period_minutes: 0` for immediate action (DR-001 policy violations)
- Set `grace_period_minutes: 5` for transient failure tolerance (DR-006 auth failures)

**Debouncing:**
- Multiple detections of the same rule for the same key within the grace period trigger only one action
- This prevents alert storms and repeated suspension attempts
- Debounce state is tracked in `demo/logs/.auto_response_debounce`

### Audit Logging

All automated actions are logged to `demo/logs/auto_response.log`:

```json
{"timestamp":"2026-02-06T20:30:00Z","detection_id":"DR-001-123456","rule_id":"DR-001","action":"suspend_key","target_key":"compromised-key","status":"SUCCESS","details":"Key suspended via /key/delete API","actor":"auto_response_system"}
```

**Log Fields:**
- `timestamp`: ISO 8601 timestamp of the action
- `detection_id`: Unique identifier for the detection instance
- `rule_id`: Detection rule that triggered the action
- `action`: Type of action taken (suspend_key, quarantine_key, alert_only)
- `target_key`: Key alias that was targeted
- `status`: SUCCESS, FAILED, or SKIPPED
- `details`: Additional context
- `actor`: Always "auto_response_system" for automated actions

### Viewing Audit Logs

```bash
# View recent auto-response actions
tail -f demo/logs/auto_response.log | jq .

# Filter for failed actions
cat demo/logs/auto_response.log | jq 'select(.status == "FAILED")'

# Filter for specific rule
cat demo/logs/auto_response.log | jq 'select(.rule_id == "DR-001")'

# Count actions by status
cat demo/logs/auto_response.log | jq -s 'group_by(.status) | map({status: .[0].status, count: length})'
```

### Manual Recovery After Auto-Suspension

If a key was suspended in error:

```bash
# 1. Identify auto-suspended keys
cat demo/logs/auto_response.log | jq 'select(.status == "SUCCESS")'

# 2. Investigate the key activity (forensics)
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
  SELECT * FROM \"LiteLLM_SpendLogs\" s
  JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE v.key_alias = 'suspended-key-alias'
  ORDER BY s.\"startTime\" DESC LIMIT 20;
"

# 3. Generate a replacement key
make key-gen ALIAS=replacement-key BUDGET=10.00

# 4. Document the incident (auto-suspension is logged)
```

### Testing Auto-Response

Test auto-response logic without affecting live keys:

```bash
# Run the auto-response tests
make validate-detections

# Dry-run detection (no auto-response triggered)
make validate-detections

# Run detection without auto-response
make validate-detections
```

---

## SIEM Integration

### Normalized Evidence Schema

The AI Control Plane implements a **normalized evidence schema** (see [SIEM_INTEGRATION.md](SIEM_INTEGRATION.md)) that unifies telemetry from multiple sources into a consistent format for SIEM ingestion.

Key fields mapped for SIEM:

| Detection Rule Field | Normalized Schema Field | SIEM Field (Splunk) |
|---------------------|------------------------|---------------------|
| `user_id` | `principal.id` | `user_id` |
| `model_id` | `ai.model.id` | `model_id` |
| `start_date` | `ai.request.timestamp` | `_time` |
| `spend` | `ai.cost.amount` | `spend` |
| `status` | `policy.action` | `action` |

### SIEM Query Library

Pre-built SIEM queries for all detection rules are available in:

```bash
demo/config/siem_queries.yaml
```

Supported platforms:
- **Splunk SPL** - For Splunk Enterprise/Cloud
- **ELK KQL** - For Elasticsearch/Kibana
- **Microsoft Sentinel KQL** - For Azure Sentinel
- **Sigma** - Generic detection format for rule conversion

### Example: Running DR-001 in Splunk

Use the SIEM query definitions from `demo/config/siem_queries.yaml` and keep them synchronized with detection rules:

```bash
# Validate SIEM query sync contract
make validate-siem-queries

# Inspect DR-001 query definitions
grep -n "DR-001" demo/config/siem_queries.yaml
```

Splunk SPL example:

```spl
index=ai_gateway sourcetype=litellm_audit
| eval approved_models="{{APPROVED_MODELS_SPLUNK}}"
| where NOT match(model_id, approved_models)
| where _time > relative_time(now(), "-24h")
| table request_id, model_id, key_alias, user_id, _time, status
| sort - _time
```

### Example: Running DR-002 in Sentinel KQL

```kusto
let TokenThreshold = 100000;
let TimeWindow = 24h;
AIGatewayLogs
| where TimeGenerated > ago(TimeWindow)
| summarize TotalTokens = sum(PromptTokens + CompletionTokens) by UserId
| where TotalTokens > TokenThreshold
| order by TotalTokens desc
```

### Correlation Across Sources

For unified governance across API key mode and subscription mode:

```spl
index=ai_gateway (sourcetype=litellm_audit OR sourcetype=otel_logs)
| eval normalized_user=coalesce(user_id, principal.id)
| eval normalized_model=coalesce(model_id, ai.model.id)
| stats values(sourcetype) as sources by normalized_user
| where mvcount(sources) > 1
```

### Cross-Reference Documentation

- [SIEM_INTEGRATION.md](SIEM_INTEGRATION.md) - Complete SIEM integration guide
- [Normalized Schema](../../demo/config/normalized_schema.yaml) - Field definitions
- [SIEM Queries](../../demo/config/siem_queries.yaml) - Platform-specific queries

---

## Detection Rules Authoring Guide

For comprehensive documentation on authoring detection rules, including SQL query requirements, parameter handling, and validation procedures, see:

**[Detection Rules Authoring Guide](../DETECTION_RULES.md)**

This guide covers:
- Complete YAML schema reference
- SQL pattern requirements and best practices
- Database table references and JOIN patterns
- psql variable substitution for secure parameters
- Step-by-step testing procedures
- Troubleshooting common issues

---

## Rule Customization

### Modifying Thresholds

Edit `demo/config/detection_rules.yaml` to adjust detection thresholds:

```yaml
- rule_id: DR-002
  name: "Token Usage Spike"
  parameters:
    threshold_tokens: 100000  # Change to 50000 for more sensitive detection
    window_hours: 24
```

### Adding Custom Rules

To add a new detection rule, follow the detailed authoring guide in [DETECTION_RULES.md](../DETECTION_RULES.md). Here's a quick start:

1. Add a new rule block to `detection_rules.yaml`:

```yaml
  - rule_id: DR-007
    name: "My Custom Rule"
    description: "Detects specific pattern"
    severity: "medium"
    category: "custom"
    enabled: true
    parameters:
      # Define parameters here
    sql_query: |
      SELECT * FROM "LiteLLM_SpendLogs"
      WHERE your_condition_here;
    remediation: "Describe how to respond"
```

2. Test the rule:

```bash
# Preview the rule (dry-run)
make validate-detections

# Run the rule
make validate-detections

# Validate SQL syntax
make validate-detections
```

**For comprehensive authoring guidelines, see [DETECTION_RULES.md](../DETECTION_RULES.md).**

### Disabling Rules

To temporarily disable a rule without removing it:

```yaml
  - rule_id: DR-002
    enabled: false  # Set to false to disable
```

---

## Troubleshooting

### No results returned

- Verify services are running: `make health`
- Check if audit log table exists: `make db-status`
- Ensure API requests have been made (logs populate on activity)
- Verify the database connection string is correct

### SQL query errors

- Validate YAML syntax: `yamllint demo/config/detection_rules.yaml`
- Test queries manually: `make db-shell` then run SQL directly
- Check that column names match your LiteLLM version
- Validate SQL syntax without execution: `make validate-detections`
- See [DETECTION_RULES.md](../DETECTION_RULES.md) for SQL pattern requirements

### Container not found

- Start services: `make up`
- Check container status: `docker compose ps`
- Verify Docker Compose is installed and working

---

## SIEM Query Sync Validation

### Overview

The detection rules (`detection_rules.yaml`) and SIEM query mappings (`siem_queries.yaml`) must stay in sync. The SIEM Query Sync Validator ensures that:

1. Every rule_id in `detection_rules.yaml` has a matching entry in `siem_queries.yaml`
2. Every rule_id in `siem_queries.yaml` has a matching entry in `detection_rules.yaml`
3. No duplicate rule IDs exist in either file
4. Rule IDs follow the DR-### format
5. Enabled rules have required vendor queries (Splunk, ELK, Sentinel, Sigma)

### Running the Validator

```bash
# Run SIEM query sync validation (included in make lint)
make validate-siem-queries

# Include schema validation checks
make validate-siem-schema
```

Custom rule/query path overrides are not part of the public-snapshot command surface; update tracked config files in `demo/config/` and rerun validation.

### Validation Checks

The validator performs the following checks:

| Check | Description | Failure Action |
|-------|-------------|----------------|
| Bidirectional sync | All rule IDs must exist in both files | Report missing/extra IDs |
| Duplicate detection | No duplicate rule IDs allowed | Report duplicate IDs |
| ID format | Rule IDs must match DR-### | Report invalid formats |
| Vendor queries | Enabled rules require all vendor sections | Report missing sections |

### Sync Contract

**For Enabled Rules:**
- `enabled: true` in both files
- Required vendor sections:
  - `splunk.query` - Splunk SPL query
  - `elk_kql.query` OR `elk_kql.aggregation` - ELK KQL query
  - `sentinel_kql.query` - Microsoft Sentinel KQL query
  - `sigma.detection` - Sigma detection section

**For Disabled Rules:**
- `enabled: false` in detection_rules.yaml
- SIEM entry may be `enabled: false` or omitted
- Vendor sections are optional for disabled rules

### Add New Rule Checklist

When adding a new detection rule:

1. [ ] Add rule to `detection_rules.yaml` with unique DR-### ID
2. [ ] Add corresponding entry to `siem_queries.yaml`
3. [ ] Add schema mapping to `normalized_schema.yaml` (if new fields)
4. [ ] Include all required vendor queries for enabled rules
5. [ ] Run validator: `make validate-siem-queries`
6. [ ] Run detection validation: `make validate-detections`
7. [ ] Run SIEM sync validation: `make validate-siem-queries`
8. [ ] Run full validation gate: `make lint`
9. [ ] Update this documentation with rule description

**Detailed authoring guide:** [DETECTION_RULES.md](../DETECTION_RULES.md)

### CI Integration

The validator is automatically run as part of `make lint`:

```bash
# Full lint suite includes SIEM sync validation
make lint

# CI gate includes all validations
make ci
```

Validation failures will cause the CI gate to fail, preventing drift between detection rules and SIEM queries.

---

## Rapid Response Mapping

This section maps detection rules to rapid response playbooks. When high-severity detections trigger, follow the linked runbook procedures.

Map these detections to customer-specific framework controls during pilot or production validation; this repository does not publish a normative framework crosswalk.

### Detection-to-Response Mapping

| Detection Rule | Severity | Response Action | Runbook Reference |
|----------------|----------|-----------------|-------------------|
| **DR-001** Non-Approved Model Access | High | Revoke key immediately; investigate intent | [RUNBOOK.md section 9.6](../RUNBOOK.md#96-rapid-response-containment--key-lifecycle) |
| **DR-002** Token Usage Spike | Medium | Verify key holder identity; rotate if compromised | [RUNBOOK.md Key Rotation](../RUNBOOK.md#key-rotation-procedure) |
| **DR-003** High Block/Error Rate | Medium | Check key validity; rotate if expired | [RUNBOOK.md Key Lifecycle](../RUNBOOK.md#62-key-lifecycle) |
| **DR-006** Failed Authentication Attempts | High | Revoke key; investigate source IPs | [RUNBOOK.md Security Incidents](../RUNBOOK.md#93-security-incidents) |
| **DR-005** Rapid Request Rate | Medium | Quarantine key pending investigation | [RUNBOOK.md First 15 Minutes](../RUNBOOK.md#first-15-minutes-checklist) |
| **DR-008** DLP Block Event Detected | High | Review content; educate or revoke key | [RUNBOOK.md Data Protection](../RUNBOOK.md#data-protection) |
| **DR-009** Repeated PII Submission Attempts | High | Auto-suspended; investigate immediately | [RUNBOOK.md Insider Threat](../RUNBOOK.md#insider-threat) |
| **DR-010** Prompt Injection Attempt | Medium | Review content; restrict key if needed | [RUNBOOK.md Policy Violations](../RUNBOOK.md#policy-violations) |

### Automated Response Integration

Detection validation output can feed response automation:

```bash
# Capture detection validation output for triage
make validate-detections 2>&1 | tee /tmp/detections-$(date +%Y%m%d-%H%M).log

# Revoke affected key aliases after operator review
make key-revoke ALIAS=<alias>
```

### Evidence Preservation

For each high-severity detection, capture:

```bash
# 1. Detection validation output (stderr + stdout)
make validate-detections 2>&1 | tee incident-$(date +%Y%m%d-%H%M)-detections.log

# 2. Database state at time of detection
make db-status > incident-$(date +%Y%m%d-%H%M)-dbstate.txt

# 3. Key-specific audit trail
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
  SELECT 
    TO_CHAR(s.\"startTime\", 'YYYY-MM-DD HH24:MI:SS') AS timestamp,
    v.key_alias, s.model, s.status, s.spend
  FROM \"LiteLLM_SpendLogs\" s
  JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE v.key_alias = 'suspected-key'
  ORDER BY s.\"startTime\" DESC;
" > incident-$(date +%Y%m%d-%H%M)-audit.csv
```

See the [Operational Runbook](../RUNBOOK.md) for complete incident response procedures including dual-control/break-glass processes.

---

## Related Documentation

- [SIEM_INTEGRATION.md](SIEM_INTEGRATION.md) - SIEM integration guide and query examples
- [DATABASE.md](../DATABASE.md) - Database schema and queries
- [LOGGING.md](../../demo/LOGGING.md) - Log configuration and rotation
- [DEPLOYMENT.md](../DEPLOYMENT.md) - Deployment architecture
- [AGENTS.md](../../AGENTS.md) - Repository conventions
