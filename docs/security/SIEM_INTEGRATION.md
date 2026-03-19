# AI Control Plane - SIEM Integration Guide

## Overview

This guide describes how to integrate the AI Control Plane with Security Information and Event Management (SIEM) systems for unified governance and security monitoring.

Per the [Enterprise AI Control Plane Strategy](../ENTERPRISE_STRATEGY.md) (Section 4.2), the AI Control Plane implements a **normalized evidence pipeline** that unifies telemetry from multiple sources into a single schema for SIEM ingestion.

Map this evidence pipeline to customer-specific control frameworks during pilot or production validation; the canonical buyer-safe mapping now lives in [../COMPLIANCE_CROSSWALK.md](../COMPLIANCE_CROSSWALK.md), but customer-specific implementation and auditor interpretation still remain environment-dependent.

## Architecture

### Evidence Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AI Control Plane Sources                             │
├──────────────────────────┬─────────────────────┬────────────────────────────┤
│  API Key Mode (Gateway)  │ Subscription Paths  │  SaaS Compliance           │
│  ┌────────────────────┐  │ ┌─────────────────┐ │  ┌────────────────────┐   │
│  │ LiteLLM Gateway    │  │ │ Codex/Claude    │ │  │ OpenAI Compliance  │   │
│  │                    │  │ │ Code            │ │  │ API                │   │
│  │ - Virtual keys     │  │ │ (Routed/Direct) │ │  │                    │   │
│  │ - Budget tracking  │  │ │                 │ │  │ - Audit exports    │   │
│  │ - Policy enforce   │  │ │ - Routed:       │ │  │ - User attribution │   │
│  │                    │  │ │   gateway logs  │ │  │ - Enterprise logs  │   │
│  │                    │  │ │ - Direct: OTEL  │ │  │                    │   │
│  └─────────┬──────────┘  │ └───────┬─────────┘ │  └─────────┬──────────┘   │
└────────────┼─────────────┴─────────┼───────────┴────────────┼──────────────┘
             │                       │                        │
             v                       v                        v
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Normalized Evidence Schema                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │  principal.id      │  Who initiated the request                        │  │
│  │  ai.model.id       │  Model + provider identification                  │  │
│  │  ai.tokens.*       │  Token consumption metadata                       │  │
│  │  ai.cost.*         │  Cost tracking                                    │  │
│  │  policy.action     │  Enforcement outcome (allowed/blocked/error)      │  │
│  │  correlation.*     │  Trace IDs for investigations                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────┬───────────────────────────────────────┘
                                      │
                                      v
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SIEM Ingestion Methods                               │
├──────────────┬──────────────┬──────────────┬──────────────┬─────────────────┤
│   Splunk     │    ELK       │   Sentinel   │   Chronicle  │  Custom HTTP    │
│   Enterprise │   Stack      │   (Azure)    │   (Google)   │   Endpoints     │
└──────────────┴──────────────┴──────────────┴──────────────┴─────────────────┘
                                      ▲
                                      │ Unified Evidence Feed
                                      │ demo/logs/normalized/evidence.jsonl
                                      │ (Recommended)
```

### Data Sources

| Source | Type | Schema | Content | Enforcement |
|--------|------|--------|---------|-------------|
| **PostgreSQL** (Gateway) | Structured SQL | `LiteLLM_SpendLogs` | Usage/cost metadata (no prompt bodies) | Active (inline) |
| **OTEL Collector** | JSON Lines | Normalized OTEL | Telemetry for direct/bypass paths and optional client-side correlation | Detective (post-hoc) |
| **Compliance API** | JSON Lines | Normalized Compliance | Audit exports (vendor-redacted) | Detective (post-hoc) |
| **Unified Evidence** | JSON Lines | Normalized Schema | All sources merged (metadata-only) | Detective (post-hoc) |

### Content Handling and Retention

**Metadata-First Default:**
The AI Control Plane's evidence pipeline is intentionally **metadata-first**—capturing who used what model, when, and at what cost—without prompt/response content. This minimizes compliance risk and data residency concerns.

| Data Type | Default Handling | Notes |
|-----------|-----------------|-------|
| **Usage metadata** | Included | Principal, model, tokens, cost, timestamps |
| **Policy outcomes** | Included | Allowed/blocked/error actions |
| **Correlation IDs** | Included | Trace IDs for cross-source investigation |
| **Prompt/response content** | **Excluded** | Not captured in default pipeline |

**Transcript Content (if required):**
Some compliance exports may include transcript content depending on vendor tier and endpoints. If transcripts are ingested:
- Route to a **separate restricted store** with strict access controls
- Apply **shorter retention periods** than metadata
- Implement **additional encryption at rest**
- Document **data residency** and processing locations

**Recommended SIEM Feed:**
The unified evidence JSONL (`demo/logs/normalized/evidence.jsonl`) is the recommended SIEM feed—pre-redacted, normalized, and ready for ingestion without content moderation concerns.

## Quick Start

### 1. View Normalized Schema

```bash
cat demo/config/normalized_schema.yaml
```

### 2. Generate Unified Evidence Feed

```bash
# Run the complete pipeline
make release-bundle

# View the unified feed
make release-bundle-verify
```

### 3. View Sample Telemetry Data

```bash
# Individual sources
cat demo/logs/otel/telemetry.jsonl | head -5
cat demo/logs/compliance/compliance_events.jsonl | head -5
cat demo/logs/gateway/gateway_events.jsonl | head -5

# Unified feed
cat demo/logs/normalized/evidence.jsonl | head -5
```

### 4. View SIEM Query Examples

```bash
cat demo/config/siem_queries.yaml | less
```

### 5. Run Detection Rules

```bash
make detection
```

**Rule quality gating:** Detection outputs now include `operational_status` and `coverage_tier`. Route only `validated` + `decision-grade` findings to paging workflows by default; keep `example` + `demo` findings in analyst-review channels until tuned for production.

### 6. Demo Walkthrough

For a comprehensive step-by-step demonstration of the SIEM integration including:
- Three evidence sources (gateway, OTEL, compliance exports)
- Evidence pipeline execution (export, merge, validate)
- Cross-source correlation examples
- Executive reporting

See: **[SaaS/Subscription Governance Demo](../demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md)**

## Integration Methods

### Method 1: PostgreSQL Database Connector (Recommended for Gateway Mode)

Connect your SIEM directly to the PostgreSQL database for real-time query access.

> **Important:** By default, PostgreSQL is not published to the host network in this repo.
> If you want your SIEM to connect directly, you must intentionally publish port 5432
> (or use SSH port forwarding). See `../DEPLOYMENT.md`.

**Splunk DB Connect:**
```
1. Install Splunk DB Connect app
2. Add PostgreSQL connection:
   - Host: your-docker-host
   - Port: 5432
   - Database: litellm
   - User: litellm
3. Create input using SQL queries from demo/config/siem_queries.yaml
```

**ELK JDBC Input:**
```yaml
# logstash.conf
input {
  jdbc {
    # If Postgres is published/tunneled to the host:
    jdbc_connection_string => "jdbc:postgresql://localhost:5432/litellm"
    # If Logstash runs in Docker on the same network as Postgres, use:
    # jdbc_connection_string => "jdbc:postgresql://postgres:5432/litellm"
    jdbc_user => "litellm"
    jdbc_password => "${DB_PASSWORD}"
	    jdbc_driver_library => "/path/to/postgresql.jar"
	    jdbc_driver_class => "org.postgresql.Driver"
	    schedule => "*/5 * * * *"
	    statement => "SELECT * FROM \"LiteLLM_SpendLogs\" WHERE \"startTime\" > NOW() - INTERVAL '5 minutes'"
	  }
	}
```

### Method 2: File Export (OTEL Telemetry)

The OTEL collector writes normalized telemetry to:
```
demo/logs/otel/telemetry.jsonl
```

**Splunk Universal Forwarder:**
```ini
# inputs.conf
[monitor:///path/to/ai-control-plane/demo/logs/otel/telemetry.jsonl]
disabled = false
sourcetype = otel_ai_telemetry
index = ai_gateway
```

**Filebeat (ELK):**
```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /path/to/ai-control-plane/demo/logs/otel/telemetry.jsonl
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "ai-otel-telemetry-%{+yyyy.MM.dd}"
```

### Method 3: Compliance API Export (SaaS Governance)

For subscription mode usage (ChatGPT Enterprise, Codex via OAuth), OpenAI provides
Compliance APIs that export audit-grade logs with user attribution and metadata.

**Prerequisites:**
- OpenAI Enterprise workspace with Compliance API access
- `OPENAI_COMPLIANCE_API_KEY` environment variable set (optional - fixture mode works without it)
- `OPENAI_ORG_ID` configured (optional - fixture mode works without it)

**Pull Compliance Exports:**
```bash
make validate-detections
```

This generates normalized events to:
```
demo/logs/compliance/compliance_events.jsonl
```

**View Events:**
```bash
make db-status
```

**SIEM Integration:**

**Splunk Universal Forwarder:**
```ini
[monitor:///path/to/ai-control-plane/demo/logs/compliance/compliance_events.jsonl]
disabled = false
sourcetype = ai_compliance_api
index = ai_gateway
```

**Unified Evidence Pipeline:**

The compliance API events use the same normalized schema as gateway and OTEL sources,
enabling unified queries across all AI activity:

```spl
index=ai_gateway (sourcetype=litellm_audit OR sourcetype=otel_logs OR sourcetype=compliance_api)
| eval user=coalesce(user_id, principal.id)
| stats values(sourcetype) as sources by user
```

**Security Note:**
Compliance export data is pre-redacted by OpenAI (no prompt/response content).
The ingestion script performs additional redaction to ensure no auth tokens
are logged.

### Method 4: Unified Evidence Feed (Recommended)

The unified evidence feed can combine gateway, OTEL, and compliance API sources into a single normalized JSONL file for simplified SIEM ingestion. This is the recommended approach when multi-source telemetry is in scope.

**Benefits:**
- Single ingestion point for all AI activity across API key mode, subscription mode, and SaaS compliance exports
- Pre-sorted by timestamp (chronological order)
- Deduplicated by trace ID or request ID
- Consistent normalized schema across all sources
- No need to configure multiple input sources in SIEM

**Generate Unified Feed:**
```bash
# Run the full pipeline (export + merge + validate)
make release-bundle

# Or run individual steps:
# Export gateway logs from PostgreSQL
make release-bundle

# Merge all sources (gateway + OTEL + compliance)
make release-bundle

# Validate JSON structure
make release-bundle-verify

# View the unified feed
make release-bundle-verify
```

**Output:**
```
demo/logs/normalized/evidence.jsonl
```

**SIEM Integration:**

*Splunk Universal Forwarder:*
```ini
[monitor:///path/to/ai-control-plane/demo/logs/normalized/evidence.jsonl]
disabled = false
sourcetype = ai_normalized_evidence
index = ai_gateway
```

*Filebeat (ELK):*
```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /path/to/ai-control-plane/demo/logs/normalized/evidence.jsonl
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "ai-unified-evidence-%{+yyyy.MM.dd}"
```

*Microsoft Sentinel (Log Analytics Agent):*
```bash
# Using Azure Monitor Log Analytics agent
/opt/microsoft/omsagent/bin/omsadmin.sh -w <workspace_id> -s <shared_key>

# Configure custom log collection for evidence.jsonl
```

### Method 5: Release Bundle Evidence (Recommended for Audit Prep)

Use the canonical release-bundle flow for auditor-ready evidence packaging. This is the supported public-snapshot command surface.

**Quick Start:**

```bash
# Build release evidence bundle + checksum + install manifest
make release-bundle

# Verify latest bundle integrity
make release-bundle-verify
```

**Verify a specific bundle (optional):**

```bash
RELEASE_BUNDLE_PATH=demo/logs/release-bundles/ai-control-plane-deploy-<version>.tar.gz \
  make release-bundle-verify
```

**Output Location:**

```
demo/logs/release-bundles/
├── ai-control-plane-deploy-<version>.tar.gz
└── ai-control-plane-deploy-<version>.tar.gz.sha256
```

**Bundle Integrity Verification:**

```bash
# Canonical verify command
make release-bundle-verify

# Manual checksum verification (specific artifact)
sha256sum -c demo/logs/release-bundles/ai-control-plane-deploy-<version>.tar.gz.sha256
```

**Audit Handoff Package (recommended):**
- Verified release bundle tarball + `.sha256`
- Normalized SIEM feed: `demo/logs/normalized/evidence.jsonl`
- Detection summary from latest run: `make validate-detections`
- Supply-chain summary: `demo/logs/supply-chain/summary.json`

> **Note:** Period-filtered “compliance bundle” command variants from private iterations are retired in this public snapshot; use `make release-bundle` / `make release-bundle-verify`.

### Method 6: HTTP Event Collector (HEC)

**Splunk HEC:**
```bash
# Export detection results to Splunk
curl -s http://splunk-hec:8088/services/collector/event \
  -H "Authorization: Splunk $HEC_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sourcetype": "ai_detection",
    "event": {
      "rule_id": "DR-001",
      "severity": "high",
      "finding": "Non-approved model access detected"
    }
  }'
```

**OTEL Collector HTTP Exporter:**
```yaml
# Add to demo/config/otel-collector/config.yaml
exporters:
  otlphttp/splunk:
    endpoint: "http://splunk-hec:8088"
    headers:
      Authorization: "Splunk ${SPLUNK_HEC_TOKEN}"
```

### Method 6: Real-Time Webhook Events (Push-Based)

The AI Control Plane can push events to your SIEM in real-time via webhooks. This is the recommended approach for immediate alerting and response.

**Benefits:**
- Real-time notification (no polling delay)
- Reduced overhead (no periodic queries)
- HMAC signature verification for authenticity
- Fine-grained event subscriptions

**Configuration:**

1. Add endpoint to `demo/config/webhooks.yaml`:

```yaml
webhooks:
  enabled: true
  endpoints:
    - name: "siem-webhook"
      enabled: true
      url: "${SIEM_WEBHOOK_URL}"
      secret_env: "SIEM_WEBHOOK_SECRET"
      events:
        - "key.revoked"
        - "policy.violation"
        - "detection.triggered"
      headers:
        Authorization: "Bearer ${SIEM_API_TOKEN}"
```

2. Enable webhooks:

```bash
export WEBHOOKS_ENABLED=true
```

3. Configure your SIEM to receive the webhook:
   - Splunk: HTTP Event Collector (HEC)
   - Sentinel: Logic Apps webhook trigger
   - ELK: Elasticsearch ingest endpoint

**Splunk HEC Configuration:**

```yaml
endpoints:
  - name: "splunk-hec"
    url: "https://splunk:8088/services/collector/event"
    secret_env: "SPLUNK_HEC_TOKEN"
    events:
      - "key.*"
      - "detection.triggered"
    headers:
      Authorization: "Splunk ${SPLUNK_HEC_TOKEN}"
```

**Event Correlation Example:**

Correlate webhook events with gateway logs using `event_id`:

```spl
index=ai_gateway (sourcetype=ai_webhook_event OR sourcetype=litellm_audit)
| eval correlation_id=coalesce(event_id, request_id)
| stats values(sourcetype) as sources by correlation_id
```

**Event Types Available:**

| Event Type | Description |
|------------|-------------|
| `key.created` | New API key generated |
| `key.revoked` | API key revoked |
| `key.expired` | Key budget exhausted |
| `approval.requested` | New approval request |
| `approval.approved` | Request approved |
| `approval.rejected` | Request rejected |
| `approval.escalated` | Request SLA breached |
| `budget.threshold` | Budget threshold crossed |
| `policy.violation` | Policy violation detected |
| `detection.triggered` | Detection rule triggered |

**Webhook Payload Format:**

```json
{
  "event_id": "evt_abc123def456",
  "event_type": "key.revoked",
  "timestamp": "2026-02-16T04:49:26Z",
  "idempotency_key": "key.revoked:my-key:2026-02-16T04:49:26Z",
  "source": "ai-control-plane",
  "version": "1.0",
  "payload": {
    "alias": "my-key",
    "timestamp": "2026-02-16T04:49:26Z"
  }
}
```

**Testing Webhooks:**

```bash
# Trigger detection and key-lifecycle events
make validate-detections
make key-gen ALIAS=webhook-test BUDGET=1.00
make key-revoke ALIAS=webhook-test

# Verify receiver endpoint health
curl -i "$SIEM_WEBHOOK_URL"
```

See [WEBHOOK_EVENTS.md](WEBHOOK_EVENTS.md) for complete event taxonomy, signature verification examples, and integration guides for Slack, PagerDuty, and other platforms.

## Field Mapping Reference

### Normalized Schema to SIEM Fields

| Normalized Field | Splunk | ELK (ECS) | Sentinel |
|-----------------|--------|-----------|----------|
| `principal.id` | `user_id` | `user.id` | `UserId` |
| `principal.role` | `user_role` | `user.roles` | `UserRole` |
| `principal.role_source` | `role_source` | `user.roles_source` | `RoleSource` |
| `ai.model.id` | `model_id` | `ai.model.id` | `ModelId` |
| `ai.request.timestamp` | `_time` | `@timestamp` | `TimeGenerated` |
| `ai.tokens.total` | `total_tokens` | `ai.tokens.total` | `TotalTokens` |
| `ai.cost.amount` | `spend` | `ai.cost.amount` | `Spend` |
| `policy.action` | `action` | `event.action` | `Action` |
| `correlation.trace.id` | `trace_id` | `trace.id` | `TraceId` |
| `source.type` | `sourcetype` | `event.source` | `SourceSystem` |
| `principal.identity_source` | `identity_source` | `principal.identity_source` | `IdentitySource` |
| `principal.identity_reason` | `identity_reason` | `principal.identity_reason` | `IdentityReason` |

See [Enterprise Authentication Architecture](./ENTERPRISE_AUTH_ARCHITECTURE.md) for complete identity resolution and role mapping contracts.

### Gateway Attribution Precedence (Shared-Key Mode)

Gateway exports resolve `principal.id` with one deterministic chain:

1. Valid `LiteLLM_SpendLogs.user`
2. Valid `LiteLLM_VerificationToken.user_id`
3. Valid `LiteLLM_VerificationToken.key_alias`
4. `unknown`

For SIEM operations, monitor attribution quality with:
- `principal.identity_source = "spendlogs_user"` as expected steady state
- fallback sources (`token_user_id`, `key_alias`, `unknown`) as drift or incident signals
- `principal.identity_reason = "identity_missing_or_invalid"` as a high-priority review queue

### Compliance API Field Mapping

| Compliance API Field | Normalized Field | Description |
|---------------------|------------------|-------------|
| `request_id` | `ai.request.id` | Unique request identifier |
| `user.email` | `principal.id`, `principal.email` | User attribution |
| `model` | `ai.model.id` | Model identifier |
| `provider` | `ai.provider` | AI provider (e.g., openai) |
| `created_at` | `ai.request.timestamp` | Request timestamp |
| `usage.prompt_tokens` | `ai.tokens.prompt` | Input token count |
| `usage.completion_tokens` | `ai.tokens.completion` | Output token count |
| `usage.total_cost` | `ai.cost.amount` | Request cost in USD |
| `policy_action` | `policy.action` | Enforcement action |
| `session_id` | `correlation.session.id` | Session correlation ID |

## Detection Rules

### Available Detections

| Rule ID | Name | Severity | Category |
|---------|------|----------|----------|
| DR-001 | Non-Approved Model Access | High | Policy Violation |
| DR-002 | Token Usage Spike | Medium | Anomaly |
| DR-003 | High Block/Error Rate | Medium | Availability |
| DR-004 | Budget Exhaustion Warning | Low | Cost Management |
| DR-005 | Rapid Request Rate | Medium | Anomaly |
| DR-006 | Failed Authentication Attempts | High | Security |
| DR-007 | Budget Threshold Alert | Medium | Cost Management |
| DR-008 | DLP Block Event Detected | High | Security |
| DR-009 | Repeated PII Submission Attempts | High | Security |
| DR-010 | Potential Prompt Injection Attempt | Medium | Policy Violation |

### Canonical Detection Execution

```bash
# Compatibility alias
make detection

# Canonical validation target
make validate-detections

# acpctl equivalent
./scripts/acpctl.sh validate detections
```

### SIEM Export Model (Public Snapshot)

For SIEM ingestion, use the normalized evidence pipeline as the canonical export path.

```bash
# Build unified evidence feed
make release-bundle

# Validate merged evidence format
make release-bundle-verify

# Inspect current feed
make release-bundle-verify
```

Output path:
- `demo/logs/normalized/evidence.jsonl`

> In this public snapshot, detection execution is exposed as a gate/validation command surface. Use the normalized evidence feed for machine-ingest SIEM pipelines.

### Suggested Scheduling

```crontab
# Hourly evidence refresh + validation
0 * * * * cd /path/to/ai-control-plane && make release-bundle && make release-bundle-verify

# Daily detection gate
0 9 * * * cd /path/to/ai-control-plane && make validate-detections
```

### SIEM Field Mapping (Normalized Evidence)

| Normalized Field | Splunk | ELK (ECS) | Sentinel |
|-----------------|--------|-----------|----------|
| `ai.request.id` | `request_id` | `event.id` | `RequestId` |
| `ai.request.timestamp` | `_time` | `@timestamp` | `TimeGenerated` |
| `principal.id` | `user_id` | `user.id` | `UserId` |
| `ai.model.id` | `model_id` | `ai.model.id` | `ModelId` |
| `ai.tokens.total` | `total_tokens` | `ai.tokens.total` | `TotalTokens` |
| `ai.cost.amount` | `spend` | `ai.cost.amount` | `Spend` |
| `policy.action` | `action` | `event.action` | `Action` |

## SIEM Query Examples

### Splunk SPL

**Non-Approved Model Access (DR-001):**
```spl
index=ai_gateway sourcetype=litellm_audit
| eval approved_models="<approved models from litellm.yaml>"
| where NOT match(model_id, approved_models)
| where _time > relative_time(now(), "-24h")
| table request_id, model_id, key_alias, user_id, _time, status
| sort - _time
```

> **Note:** The approved_models list should match `demo/config/litellm.yaml` `model_list`. See `demo/config/siem_queries.yaml` for the current values.

**Token Usage Spike (DR-002):**
```spl
index=ai_gateway sourcetype=litellm_audit
| where _time > relative_time(now(), "-24h")
| stats sum(prompt_tokens + completion_tokens) as total_tokens by key_alias
| where total_tokens > 100000
| sort - total_tokens
```

### ELK KQL

**Non-Approved Model Access:**
```
sourcetype:litellm_audit AND 
NOT model_id:(<approved models from litellm.yaml>) AND
@timestamp >= now-24h
```

> **Note:** The approved models list should match `demo/config/litellm.yaml` `model_list`. See `demo/config/siem_queries.yaml` for the current values.

**Authentication Failures:**
```
sourcetype:litellm_audit AND status:failure AND @timestamp >= now-24h
```

### Microsoft Sentinel KQL

**Non-Approved Model Access:**
```kusto
let approved_models = dynamic(["<approved models from litellm.yaml>"]);
AIGatewayLogs
| where TimeGenerated > ago(24h)
| where ModelId !in (approved_models)
| project TimeGenerated, RequestId, ModelId, KeyAlias, UserId, Status
| order by TimeGenerated desc
```

> **Note:** The approved_models list should match `demo/config/litellm.yaml` `model_list`. See `demo/config/siem_queries.yaml` for the current values.

**Token Usage by User:**
```kusto
AIGatewayLogs
| where TimeGenerated > ago(24h)
| summarize TotalTokens = sum(PromptTokens + CompletionTokens) by UserId
| where TotalTokens > 100000
| order by TotalTokens desc
```

## Dashboard Examples

### AI Usage Overview (Splunk)

```spl
index=ai_gateway sourcetype=litellm_audit
| eval hour=strftime(_time, "%H")
| stats count as requests, sum(spend) as total_spend by hour
| sort hour
```

### Policy Violations (ELK Lens)

- **X-axis:** `@timestamp` (Date Histogram, 1 hour)
- **Y-axis:** Count of records
- **Break down by:** `model_id` or `policy.action`
- **Filter:** `policy.action:blocked OR status:failure`

### Cost Tracking (Sentinel Workbook)

```kusto
AIGatewayLogs
| where TimeGenerated > ago(7d)
| summarize 
    TotalSpend = sum(Spend),
    TotalTokens = sum(PromptTokens + CompletionTokens),
    RequestCount = count()
    by Day = bin(TimeGenerated, 1d)
| order by Day asc
| render timechart
```

## Alert Configuration

### Splunk Alerts

**High Severity Detection Alert:**
```spl
index=ai_gateway sourcetype=litellm_audit status=failure
| where _time > relative_time(now(), "-1h")
| stats count by key_alias
| where count >= 5
```

- **Trigger condition:** Number of results > 0
- **Throttle:** 1 hour
- **Alert action:** Email to security team

### ELK Alerting

Create a rule with:
- **Index pattern:** `litellm-audit-*`
- **Query:** `status:failure`
- **Group by:** `key_alias`
- **Threshold:** Count >= 5 in last 1 hour
- **Action:** Index to `.siem-signals-*` or webhook

### Sentinel Analytics Rules

**Failed Authentication Detection:**
```kusto
let FailureThreshold = 5;
AIGatewayLogs
| where TimeGenerated > ago(24h)
| where Status == "failure"
| summarize FailureCount = count() by KeyAlias, bin(TimeGenerated, 1h)
| where FailureCount >= FailureThreshold
| extend AccountCustomEntity = KeyAlias
```

- **Tactics:** Credential Access
- **Techniques:** T1110 (Brute Force)

## Correlation Use Cases

### Use Case 1: Cross-Source Attribution

**Scenario:** User uses API key mode for some requests, subscription mode for others.

**Correlation:** Match user-centric IDs across sources, but treat gateway rows with `principal.identity_source="key_alias"` as key-level attribution until user mapping is resolved.

```spl
index=ai_gateway (sourcetype=litellm_audit OR sourcetype=otel_logs OR sourcetype=compliance_api)
| eval identity_source=coalesce(principal.identity_source, "external")
| eval normalized_user=coalesce(user_id, principal.email, principal.id)
| where NOT (sourcetype="litellm_audit" AND identity_source="key_alias")
| stats values(sourcetype) as sources by normalized_user
| where mvcount(sources) > 1
```

### Use Case 2: Budget Exhaustion Investigation

**Scenario:** Key approaches budget limit, then switches to subscription mode.

**Correlation:** Link `key_alias` to `principal.id` via user mapping.

```kusto
// Combine budget table with usage logs
KeyBudgetTable
| where PercentRemaining < 10
| join kind=inner (
    AIGatewayLogs
    | where TimeGenerated > ago(24h)
) on KeyAlias
| project KeyAlias, UserId, PercentRemaining, ModelId, Spend
```

### Use Case 3: Token Spike Pattern Analysis

**Scenario:** Unusual token consumption from gateway, OTEL, and Compliance API sources.

**Correlation:** Aggregate tokens across all sources by time window.

```spl
index=ai_gateway (sourcetype=litellm_audit OR sourcetype=otel_logs OR sourcetype=compliance_api)
| eval tokens=coalesce(prompt_tokens + completion_tokens, ai.tokens.total)
| eval user=coalesce(key_alias, principal.id)
| bucket _time span=1h
| stats sum(tokens) as hourly_tokens by _time, user
| where hourly_tokens > 50000
```

## Best Practices

### 1. Field Normalization

Always map source fields to normalized schema before alerting:

```python
# Example normalization function
def normalize_event(event):
    return {
        "principal.id": event.get("user_id") or event.get("principal.id"),
        "ai.model.id": event.get("model_id") or event.get("ai.model.id"),
        "ai.request.timestamp": event.get("start_date") or event.get("timestamp"),
        "ai.cost.amount": event.get("spend") or event.get("ai.cost.amount"),
        "policy.action": normalize_action(event.get("status") or event.get("policy.action")),
    }
```

### 2. Time Synchronization

Ensure all sources use UTC timestamps:
- PostgreSQL: `start_date` is already UTC
- OTEL: Normalize to ISO 8601 UTC

### 3. Handling PII

- **Never log:** OAuth tokens, API keys, prompt/response content
- **Consider redacting:** Email addresses, user IDs in external systems
- **Review:** Log retention policies for compliance requirements

### 4. Alert Fatigue Prevention

- Use throttling (e.g., max 1 alert per hour per rule)
- Implement suppression for known maintenance windows
- Tune thresholds based on baseline usage patterns

## Operational Response

This section describes how detections feed into incident response and where key lifecycle management fits in the operational workflow.

### Detection-to-Response Pipeline

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Detection     │───▶│   SIEM Alert    │───▶│   Response      │
│   Rule Triggers │    │   Generated     │    │   Action        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                                           │
        ▼                                           ▼
┌─────────────────┐                        ┌─────────────────┐
│   Evidence      │                        │   Key Revoke    │
│   Preserved     │                        │   or Rotation   │
└─────────────────┘                        └─────────────────┘
```

### Response Triggers by Severity

| Severity | Response Time | Typical Actions |
|----------|---------------|-----------------|
| **High** | Immediate (minutes) | Key revocation, egress lockdown, incident ticket |
| **Medium** | Within 4 hours | Key rotation investigation, threshold tuning |
| **Low** | Next business day | Budget review, trend analysis |

### Key Revocation in Response Flow

When high-severity detections indicate key compromise:

1. **Automated Detection** → DR-001, DR-006 trigger
2. **Alert Routing** → SIEM alert to security team
3. **Evidence Collection** → Capture detection output and audit logs
4. **Key Revocation** → Execute `make key-revoke ALIAS=<alias>`
5. **Verification** → Confirm key no longer authenticates
6. **Documentation** → Update incident ticket with evidence

**Command Reference:**
```bash
# Immediate key revocation
make key-revoke ALIAS=<alias>

# Or capture detection validation evidence before response
make validate-detections 2>&1 | tee detections.log
make key-revoke ALIAS=<alias>
```

### Evidence for Incident Response

Preserve these artifacts for each security incident:

| Artifact | Source | Purpose |
|----------|--------|---------|
| Detection findings | `make validate-detections` | Shows what triggered the alert |
| Audit logs | `make db-status` + SQL queries | Complete activity history |
| Key metadata | `LiteLLM_VerificationToken` table | Key creation, budget, alias |
| Gateway logs | `docker compose logs litellm` | Raw request/response traces |

**Full procedures:** See [RUNBOOK.md section 9.6](../RUNBOOK.md#96-rapid-response-containment--key-lifecycle) for complete rapid response playbooks including dual-control processes.

---

## Troubleshooting

### No Data in SIEM

1. **Verify source is producing data:**
   ```bash
   make db-status  # Check PostgreSQL
   cat demo/logs/otel/telemetry.jsonl | wc -l  # Check OTEL
   cat demo/logs/compliance/compliance_events.jsonl | wc -l  # Check Compliance
   
   # Or use the unified feed (recommended)
   make release-bundle
   cat demo/logs/normalized/evidence.jsonl | wc -l  # Check unified feed
   ```

2. **Check SIEM connection:**
   ```bash
   # Test Splunk HEC
   curl -s http://splunk:8088/services/collector/health
   
   # Test Elasticsearch
   curl -s http://elasticsearch:9200/_cluster/health
   ```

3. **Verify field extraction:**
   - Check that sourcetype has correct field extraction rules
   - Verify JSON parsing is enabled for OTEL logs

### Duplicate Events

If seeing duplicates:
- Add deduplication by `ai.request.id` or `correlation.trace.id`
- Use Splunk's `dedup` command or ELK's `collapse` feature
- Ensure only one ingestion path per source

### Timestamp Issues

If events have wrong timestamps:
- Verify timezone settings in SIEM
- Check that `_time` (Splunk) or `@timestamp` (ELK) is correctly mapped
- Use `strftime` or date parsing functions as needed

## Related Documentation

- [Normalized Schema](../../demo/config/normalized_schema.yaml) - Complete schema definition
- [Detection Rules](../../demo/config/detection_rules.yaml) - SQL-based detection rules
- [SIEM Queries](../../demo/config/siem_queries.yaml) - Platform-specific queries
- [DATABASE.md](../DATABASE.md) - Database schema reference
- [OTEL_SETUP.md](../observability/OTEL_SETUP.md) - OTEL collector configuration
- [DETECTION.md](DETECTION.md) - Detection rule documentation (includes SIEM sync validator)

## Keeping Rules in Sync

### SIEM Query Sync Validator

The detection rules (`detection_rules.yaml`) and SIEM query mappings (`siem_queries.yaml`) must stay synchronized. Use the validator to ensure consistency:

```bash
# Validate sync between files
make validate-siem-queries

# Run with verbose output (via make)
make validate-siem-queries

# Run with verbose output (direct script)
make validate-siem-queries
```

### Sync Contract

**Bidirectional Requirement:**
- Every `rule_id` in `detection_rules.yaml` must exist in `siem_queries.yaml`
- Every `rule_id` in `siem_queries.yaml` must exist in `detection_rules.yaml`
- Rule IDs must follow the `DR-###` format (e.g., DR-001)

**For Enabled Rules:**
- `enabled: true` in both files
- Required vendor sections in `siem_queries.yaml`:
  - `splunk.query` - Splunk SPL query
  - `elk_kql.query` OR `elk_kql.aggregation` - ELK KQL query
  - `sentinel_kql.query` - Microsoft Sentinel KQL query
  - `sigma.detection` - Sigma detection section

**For Disabled Rules:**
- `enabled: false` in `detection_rules.yaml`
- SIEM entry may be `enabled: false` or omitted
- Vendor sections are optional for disabled rules

### Add New Rule Checklist

When adding a new detection rule with SIEM queries:

1. [ ] Add rule to `detection_rules.yaml` with unique DR-### ID
2. [ ] Add corresponding entry to `siem_queries.yaml`
3. [ ] Include all required vendor queries for enabled rules
4. [ ] Run validator: `make validate-siem-queries`
5. [ ] Run detection validation: `make validate-detections`
6. [ ] Update documentation

### CI Integration

The validator runs automatically during `make lint`:

```bash
# Full lint includes SIEM sync validation
make lint

# CI gate includes all validations
make ci
```

Validation failures will cause the CI gate to fail, preventing drift between detection rules and SIEM queries.

## References

- [Enterprise AI Control Plane Strategy](../ENTERPRISE_STRATEGY.md) - Strategic context
- [Splunk Documentation](https://docs.splunk.com/)
- [Elasticsearch Documentation](https://www.elastic.co/guide/index.html)
- [Microsoft Sentinel KQL](https://docs.microsoft.com/azure/sentinel/kusto-overview)
- [Sigma Specification](https://github.com/SigmaHQ/sigma-specification)
