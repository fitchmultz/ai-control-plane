# Financial Governance and Chargeback

**Purpose:** Define the finance-ready showback/chargeback model for AI Control Plane usage, including attribution conventions, reporting workflows, and reconciliation guidance.

**Audience:** FinOps, Procurement, Security Operations, Platform Engineering

**Last Updated:** February 2026

---

## 1. Scope and Non-Scope

### What This Document Covers

- **Showback vs chargeback definitions** and organizational workflows
- **Attribution model** for mapping AI usage to teams/cost centers
- **Data sources** and accuracy considerations for financial reporting
- **Reporting recipes** (SQL queries, CLI pipelines, templates)
- **Reconciliation guidance** for matching internal records to provider invoices
- **Operational cadence** and role responsibilities

### What This Document Does NOT Cover

- **Provider invoice ingestion automation** — We document how to reconcile, not how to build vendor billing APIs
- **Real-time billing alerts** — See `BUDGETS_AND_RATE_LIMITS.md` for budget enforcement mechanics
- **Enterprise seat/license inventory pipelines** — We document required inputs and templates, not proprietary inventory systems
- **Prompt/response content** — Financial governance uses metadata only (who, what model, when, tokens, cost)

### Invariants

- **LiteLLM schema is authoritative** — No manual database migrations; LiteLLM manages schema internally
- **Estimated usage cost ≠ Billable cost** — See Section 3 for accuracy considerations
- **Attribution requires trusted identity inputs** — Use parseable `key_alias` and/or trusted user context (`LiteLLM_SpendLogs.user`) with explicit fallback monitoring

---

## 2. Definitions

| Term | Definition | Example |
|------|------------|---------|
| **Showback** | Informational reporting: "Here's what you used" without internal billing | Monthly usage report by team |
| **Chargeback** | Internal billing: Allocating costs to cost centers and potentially invoicing internal teams | Cross-charging AI costs to departmental budgets |
| **Budget** | Guardrails for spend control — prevents usage, does not allocate costs | Per-key $50/month budget with hard stop |
| **Forecasting** | Predicting future spend based on historical trends | "Based on current usage, Q2 spend will be $12K" |
| **Attribution** | Mapping usage records to organizational entities (team, cost center, service) | `svc-api-gateway__team-platform__cc-12345` |
| **Principal** | The entity making API requests (user, service, team) identified by resolved attribution precedence | `alice@example.com` or `svc-api-gateway__team-platform__cc-12345` |

### Budget vs Chargeback Relationship

```
┌─────────────────────────────────────────────────────────────────┐
│                    Financial Governance                          │
├─────────────────────────────┬───────────────────────────────────┤
│      Budget Enforcement      │         Chargeback Allocation     │
│      (Preventive Control)    │         (Allocative Process)      │
├─────────────────────────────┼───────────────────────────────────┤
│ • Prevents overspend         │ • Allocates actual costs          │
│ • Gateway-enforced           │ • Finance-managed                 │
│ • Real-time blocking         │ • Monthly cadence                 │
│ • Hard limits                │ • Invoice reconciliation          │
└─────────────────────────────┴───────────────────────────────────┘
```

**Key Point:** Budget enforcement prevents surprise bills. Chargeback allocates the actual costs that were incurred within budget limits.

---

## 3. Billing Surfaces

### 3.1 Usage-Based Billing (API Keys)

**Billing Model:**
- Enterprise is billed by model providers (OpenAI API platform, Anthropic API, etc.) based on actual token consumption
- Provider invoices show aggregate usage by API key or organization

**Chargeback Mechanism:**
- AI Gateway enables internal chargeback by principal:
  - **User** — via individual virtual keys
  - **Team** — via team-scoped keys
  - **Service** — via service account keys

**Data Source:**
- `LiteLLM_SpendLogs` — granular usage records with spend attribution
- `LiteLLM_VerificationToken` — key metadata including `key_alias`
- Consistent logs from gateway provide granular usage attribution

**Accuracy:**
- LiteLLM `spend` field closely approximates provider usage-billed invoices
- Still requires reconciliation (timing differences, currency conversion, rounding)

### 3.2 Seat-Based Billing (Subscriptions)

**Billing Model:**
- Enterprise is billed per seat (ChatGPT Business/Enterprise, Claude Team, Copilot, Cursor, etc.)
- Fixed per-user cost regardless of usage volume

**Chargeback Mechanism:**
- **Primary:** Seat assignment (who has a license)
- **Secondary:** Usage telemetry where available (vendor analytics, compliance exports)

**Data Sources:**
| Source | Coverage | Use Case |
|--------|----------|----------|
| Vendor seat management | Assigned licenses | Primary chargeback baseline |
| Gateway logs (if routed) | Token usage via gateway | Usage optimization signal |
| OTEL telemetry | Direct subscription usage | Showback/optimization only |
| Compliance exports | Vendor-verified audit logs | Audit-grade attribution |

**Accuracy:**
- **Seat cost** is the actual billable amount
- **Usage telemetry** (OTEL, gateway logs) is for showback and optimization — it does NOT equal the vendor invoice
- Token-based `spend` in OTEL may be missing or estimated

### 3.3 Mixed Mode (Subscription Routed Through Gateway)

**Billing Model:**
- Upstream: Vendor subscription (seat-based or metered)
- Downstream: Gateway tracks usage via virtual keys

**Chargeback Mechanism:**
- Combine seat-based chargeback with usage-based optimization insights
- Gateway provides granular attribution; subscription provides the actual invoice

**Data Sources:**
- PostgreSQL gateway logs for attribution
- Vendor seat roster for primary chargeback

---

## 4. Data Sources and Accuracy

### 4.1 Primary Data Sources

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `LiteLLM_SpendLogs` | Request-level usage/cost | `api_key`, `model`, `spend`, `prompt_tokens`, `completion_tokens`, `startTime`, `status` |
| `LiteLLM_VerificationToken` | Virtual key metadata | `token`, `key_alias`, `spend`, `budget_id`, `user_id` |
| `LiteLLM_BudgetTable` | Budget records | `budget_id`, `max_budget`, `budget_duration` |

### 4.2 Accuracy Considerations

**Usage-Based (API-Key Mode):**
- ✓ LiteLLM `spend` is close to provider usage-billed invoices
- ⚠ Reconcile for: timing differences (request time vs billing period), currency conversion, provider-specific rounding

**Subscription / Seat-Based:**
- ⚠ LiteLLM `spend` (when routed) is usage-cost **estimate** for showback/optimization
- ⚠ OTEL cost fields may be missing or estimated
- ✓ **Actual billable cost** is the vendor invoice (seat-based or metered subscription)

**Key Principle:**
> Always distinguish "estimated usage cost" from "billable cost" when reporting to finance. Usage-based billing reconciles closely; subscription billing reconciles by seat assignment.

---

## 5. Attribution Model

### 5.1 Parseable `key_alias` Convention

To enable automated attribution without database schema changes, adopt a parseable `key_alias` convention using the `__` (double underscore) delimiter.

**Constraints:**
- Alias validator allows: `A-Z a-z 0-9 . _ -` (max 64 characters)
- Use `__` as delimiter (two underscores to distinguish from single underscore in names)

### 5.2 Recommended Alias Patterns

| Principal Type | Pattern | Example |
|----------------|---------|---------|
| **Service** | `svc-<service>__team-<team>__cc-<costcenter>` | `svc-api-gateway__team-platform__cc-12345` |
| **User** | `usr-<employeeid>__team-<team>__cc-<costcenter>` | `usr-jdoe123__team-eng__cc-54321` |
| **Team** | `team-<team>__cc-<costcenter>` | `team-data-science__cc-99999` |
| **Unknown** | `unknown` or `cc-unknown` | `cc-unknown` |

**Privacy Note:**
- Avoid using email addresses in aliases (PII exposure risk)
- Use employee IDs or team keys for user-level chargeback
- `key_alias` appears in logs and reports

### 5.3 Handling Unknown Attribution

When `key_alias` is missing or doesn't follow the convention:

| Scenario | Action | Report Entry |
|----------|--------|--------------|
| No alias | Use `'unknown'` | Allocate to `cc-unknown` for manual review |
| Partial alias (no cc) | Extract available tokens | Allocate to known team, flag for cc assignment |
| Invalid format | Log exception | Include in exceptions list for finance review |

**Always explicitly show unattributed usage** in reports — hidden unknowns undermine trust in chargeback accuracy.

### 5.4 SQL Attribution Helpers

Extract attribution tokens from `key_alias`:

```sql
-- Extract team from key_alias
SELECT
  key_alias,
  CASE
    WHEN key_alias LIKE '%__team-%' THEN
      substring(key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  CASE
    WHEN key_alias LIKE '%__cc-%' THEN
      substring(key_alias from '__cc-([0-9]+)')
    ELSE 'unknown'
  END AS cost_center
FROM "LiteLLM_VerificationToken"
WHERE key_alias IS NOT NULL;
```

### 5.5 Shared-Key LibreChat Attribution Precedence

When LibreChat uses a shared gateway key, chargeback still requires per-user evidence. The normalized gateway export resolves `principal.id` with this precedence:

1. Valid `LiteLLM_SpendLogs.user` (trusted LibreChat-authenticated user context)
2. Valid `LiteLLM_VerificationToken.user_id`
3. Valid `LiteLLM_VerificationToken.key_alias`
4. `unknown` (fail-safe)

Operational guidance:
- Treat `principal.identity_source = "spendlogs_user"` as the expected steady state for shared-key user attribution.
- Track fallback volume (`token_user_id`, `key_alias`, `unknown`) as a governance quality metric.
- Route `identity_missing_or_invalid` records to manual finance/security review before monthly allocation close.

---

## 6. Budgeting vs Chargeback

### 6.1 Key Differences

| Aspect | Budget | Chargeback |
|--------|--------|------------|
| **Purpose** | Prevent overspend | Allocate actual costs |
| **Timing** | Real-time enforcement | Post-hoc (monthly) |
| **Mechanism** | Gateway blocks over-budget requests | Finance process allocates costs |
| **Flexibility** | Hard limits | Adjustable allocations |
| **Owner** | Platform/SecOps | FinOps/Procurement |

### 6.2 How They Interact

```
Usage Request → Gateway Check Budget ─┬─> Within Budget → Execute → Log Spend
                                      │
                                      └─> Over Budget → Block (no spend logged)
                                                          ↓
                                               No chargeback (no cost incurred)
```

**Budget as Chargeback Input:**
- Budgets inform expected spend for forecasting
- Budget alerts trigger proactive chargeback adjustments
- Budget overruns (if any) appear as exceptions in chargeback reports

---

## 7. Operational Workflow

### 7.1 Roles and Responsibilities

| Role | Responsibilities | Deliverables |
|------|------------------|--------------|
| **FinOps** | Monthly chargeback calculations, cost allocations, variance analysis | Chargeback reports, journal entries |
| **Procurement** | Provider invoice management, seat license reconciliation, contract management | Reconciled invoices, seat rosters |
| **SecOps** | Gateway access, detection rule monitoring, policy enforcement support | Security findings, policy compliance reports |
| **Platform** | Key provisioning, alias convention enforcement, infrastructure health | Key inventory, system uptime reports |

### 7.2 Daily/Weekly: Budget Risk Monitoring

Platform/SecOps monitor for budget risks:

```bash
# Quick budget risk check
make validate-detections  # Includes DR-007 budget-threshold validation

# Or query directly
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
SELECT
  v.key_alias,
  ROUND(v.spend::numeric, 4) AS spent,
  ROUND(b.max_budget::numeric, 4) AS maximum,
  ROUND(((b.max_budget - v.spend) / NULLIF(b.max_budget, 0) * 100)::numeric, 2) AS percent_remaining
FROM \"LiteLLM_VerificationToken\" v
JOIN \"LiteLLM_BudgetTable\" b ON v.budget_id = b.budget_id
WHERE b.max_budget > 0
  AND ((b.max_budget - v.spend) / NULLIF(b.max_budget, 0)) < 0.2
ORDER BY percent_remaining ASC;
"
```

### 7.3 Monthly: Showback/Chargeback Cycle

**Week 1: Data Collection**
- [ ] Capture gateway usage evidence (`make db-status`, `make release-bundle`, `make release-bundle-verify`)
- [ ] Pull provider invoices (Procurement)
- [ ] Export seat roster from vendor admin consoles (Procurement)
- [ ] Collect OTEL/compliance exports (SecOps)

**Week 2: Attribution and Allocation**
- [ ] Parse `key_alias` for team/cost center mapping
- [ ] Allocate gateway usage to cost centers
- [ ] Reconcile seat assignments to organizational units
- [ ] Identify and categorize unattributed usage

**Week 3: Reconciliation**
- [ ] Compare internal totals to provider invoices
- [ ] Investigate variances >5%
- [ ] Document exceptions and adjustments
- [ ] Finalize allocation percentages

**Week 4: Reporting and Posting**
- [ ] Generate final chargeback report (see templates)
- [ ] Deliver to finance for journal entry posting (if chargeback)
- [ ] Distribute showback reports to teams (if showback only)
- [ ] Archive data and documentation

### 7.4 Reconciliation Checklist

```markdown
## Monthly Reconciliation Checklist

### Gateway Usage vs Provider Invoices
- [ ] Record total spend from gateway logs
- [ ] Record total amount on provider invoice
- [ ] Record variance percentage (target: <5%)
- [ ] Document variance explanation

### Seat-Based Billing
- [ ] Record seats assigned per vendor
- [ ] Record seat cost per vendor
- [ ] Record total seat cost
- [ ] Confirm seat roster reconciled to HRIS

### Attribution Coverage
- [ ] Record usage with valid cost-center mapping
- [ ] Record usage with team mapping but missing cost center
- [ ] Record unattributed usage percentage
- [ ] Confirm exceptions are documented

### Approvals
- [ ] Reconciliation prepared and attached to monthly report
- [ ] FinOps review completed (name/date recorded)
- [ ] Finance approval completed (name/date recorded)
```

---

## 8. Reporting Recipes

### 8.1 SQL Queries for Finance

**Top N Principals by Spend (Attribution Test):**

```sql
SELECT
  COALESCE(v.key_alias, 'unknown') AS principal,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  COUNT(*) AS request_count,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN
      substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  CASE
    WHEN v.key_alias LIKE '%__cc-%' THEN
      substring(v.key_alias from '__cc-([0-9]+)')
    ELSE 'unknown'
  END AS cost_center
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY COALESCE(v.key_alias, 'unknown')
ORDER BY SUM(s.spend) DESC
LIMIT 20;
```

**Spend by Team/Cost Center (Allocation Report):**

```sql
WITH attribution AS (
  SELECT
    s.spend,
    COALESCE(v.key_alias, 'unknown') AS key_alias,
    CASE
      WHEN v.key_alias LIKE '%__team-%' THEN
        substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown-team'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN
        substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center
  FROM "LiteLLM_SpendLogs" s
  LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
  WHERE s."startTime" > NOW() - INTERVAL '30 days'
)
SELECT
  cost_center,
  team,
  COUNT(*) AS request_count,
  ROUND(SUM(spend)::numeric, 4) AS total_spend,
  ROUND((SUM(spend) / NULLIF((SELECT SUM(spend) FROM attribution), 0) * 100)::numeric, 2) AS percent_of_total
FROM attribution
GROUP BY cost_center, team
ORDER BY SUM(spend) DESC;
```

**Spend by Model + Team (Optimization Signal):**

```sql
SELECT
  s.model,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN
      substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  COUNT(*) AS requests,
  SUM(s."prompt_tokens" + s."completion_tokens") AS total_tokens,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY s.model, team
ORDER BY SUM(s.spend) DESC;
```

### 8.3 Canonical Chargeback Evidence Workflow

In this open-source release, legacy scorecard targets (`make governance-report*`) remain compatibility stubs only. Use `make chargeback-report` as the canonical public entrypoint for monthly chargeback/showback artifacts.

Use the canonical workflow below for monthly chargeback evidence:

```bash
# 1) Capture current platform state and usage totals
make db-status

# 2) Build and verify deterministic evidence package
make release-bundle
make release-bundle-verify

# 3) Export finance-ready CSV with the SQL query in Section 8.4
```

If older automation still invokes the legacy scorecard command, keep it stub-safe:

```bash
make chargeback-report
```

This preserves compatibility while keeping operator workflows on maintained targets.

### 8.4 CLI Pipelines

**Generate Validation Evidence and Export CSV:**

```bash
# Governance scorecard targets are compatibility stubs in the open-source release
make chargeback-report

# Capture current database status for audit evidence
make db-status

# Use the monthly SQL export query below to produce finance-ready CSV
```

**Monthly Data Export:**

```bash
# Export full month for finance analysis
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
COPY (
  SELECT
    s.\"startTime\" AS timestamp,
    COALESCE(v.key_alias, 'unknown') AS principal,
    s.model,
    s.\"prompt_tokens\",
    s.\"completion_tokens\",
    s.spend,
    CASE
      WHEN v.key_alias LIKE '%__team-%' THEN substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown'
    END AS cost_center
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" > NOW() - INTERVAL '30 days'
  ORDER BY s.\"startTime\" DESC
) TO STDOUT WITH CSV HEADER;
" > ai_usage_$(date +%Y-%m).csv
```

### 8.5 Spend Forecasting

The chargeback report includes optional spend forecasting to enable proactive budget management.

**Methodology:**

- **Algorithm:** Linear regression on the last 6 months of historical spend data
- **Output:** 3-month forward projection (Month +1, Month +2, Month +3)
- **Confidence:** Estimates may vary +/- 20% from actual spend due to usage volatility
- **Limitations:** Assumes historical trend continues; does not account for planned changes (new projects, team growth, etc.)

**Burn Rate Calculation:**

- Daily average = Total month-to-date spend / Days elapsed
- Days until exhaustion = (Total budget - Current spend) / Daily average
- Requires active budgets in `LiteLLM_BudgetTable`

**Budget Risk Assessment:**

| Risk Level | Criteria |
|------------|----------|
| Low | 3-month forecast < 50% of total budget |
| Medium | 3-month forecast >= 50% of total budget |
| High | 3-month forecast >= threshold (default: 80%) |

**CLI Usage:**

```bash
# Canonical monthly evidence flow
make db-status
make release-bundle
make release-bundle-verify

# Legacy scorecard compatibility stub (expected exit code 2)
NO_FORECAST=1 BUDGET_ALERT_THRESHOLD=75 make chargeback-report
```

> Forecasting-specific CLI automation from private iterations is not included in this open-source release.

**JSON Output Schema:**

```json
{
  "forecast": {
    "enabled": true,
    "methodology": "linear_regression",
    "confidence_note": "Estimates based on historical trends; actual spend may vary +/- 20%",
    "predictions": {
      "month_1": 1234.56,
      "month_2": 1300.12,
      "month_3": 1365.67
    },
    "burn_rate": {
      "daily_average": 41.15,
      "days_until_exhaustion": 45,
      "exhaustion_date": "2026-04-01"
    },
    "budget_analysis": {
      "total_budget": 5000.00,
      "risk_assessment": {
        "risk_level": "medium",
        "budget_percent": 78.01,
        "threshold_exceeded": false
      }
    }
  }
}
```

**When Forecasting is Unavailable:**

Forecasting returns `N/A` values when:
- Less than 2 months of historical data exists
- No spend recorded in historical periods
- Budget table has no active budgets (burn rate only)

---

## 9. Templates

See [`docs/templates/FINANCIAL_SHOWBACK_CHARGEBACK_REPORT.md`](../templates/FINANCIAL_SHOWBACK_CHARGEBACK_REPORT.md) for a reusable monthly report template suitable for finance consumption.

For automated generation in this open-source release, use the canonical workflow in section 8.3 (`make db-status`, `make release-bundle`, `make release-bundle-verify`) plus the SQL export query.

---

## 10. Automated Reporting

### Kubernetes CronJob

For production deployments, enable the chargeback CronJob for automated monthly report generation:

```yaml
# values.yaml
chargeback:
  enabled: true
  schedule: "0 9 1 * *"  # 9 AM on 1st of each month
  varianceThreshold: 15   # Alert on >15% variance
  anomalyThreshold: 200   # Flag >200% spend spikes
  notifications:
    enabled: true
  persistence:
    enabled: true
    size: 5Gi
```

**Host-First Execution:**

```bash
make chargeback-report
make chargeback-report REPORT_MONTH=2026-02 OUTPUT_FORMAT=all
```

**Retrieving Reports:**

Reports are written to the configured local archive/output path for the host-first runtime and evidence workflows.

# Copy reports from pod to local machine
kubectl cp -n acp <pod-name>:/reports/ ./chargeback-reports/

# Or mount PVC to a debug pod for inspection
kubectl run -n acp debug --rm -it \
  --overrides='{"spec":{"volumes":[{"name":"reports","persistentVolumeClaim":{"claimName":"acp-chargeback-reports"}}],"containers":[{"name":"debug","image":"alpine","volumeMounts":[{"name":"reports","mountPath":"/reports"}]}]}}' \
  --image=alpine -- /bin/sh
```

**Integration with Object Storage:**

For long-term retention, configure a sidecar container or external sync job to upload reports to object storage:

```yaml
# Example: Add to values.yaml for S3 sync sidecar
chargeback:
  extraContainers:
    - name: s3-sync
      image: amazon/aws-cli:latest
      command:
        - /bin/sh
        - -c
        - |
          while true; do
            aws s3 sync /reports/ s3://my-bucket/chargeback-reports/
            sleep 3600
          done
      volumeMounts:
        - name: reports
          mountPath: /reports
```

### Finance System Integration

CSV exports follow a standard format compatible with major ERP systems:

```csv
CostCenter,Team,SpendAmount,RequestCount,TokenCount,PercentOfTotal,ReportMonth
12345,platform,500.00,25000,5000000,40.50,2026-01
54321,engineering,300.00,15000,3000000,24.30,2026-01
```

**Supported Integrations:**
- **SAP ERP**: Direct GL posting via batch input or IDoc
- **Oracle Financials**: Journal import via GL_INTERFACE table
- **Workday**: Supplier invoice import or custom integration
- **NetSuite**: CSV import via file cabinet or REST API
- **Custom GL systems**: Generic webhook POST with JSON payload

**Webhook Integration:**

Configure webhooks for real-time finance system integration:

```bash
# Generic webhook for ERP integration
export GENERIC_WEBHOOK_URL="https://erp.company.com/api/chargeback"

# Slack for notifications
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."

# Generate and verify evidence package (for downstream notifications)
make release-bundle
make release-bundle-verify
```

---

## 11. Cross-References

- **Budget Enforcement:** [`BUDGETS_AND_RATE_LIMITS.md`](BUDGETS_AND_RATE_LIMITS.md) — Gateway-level spend controls
- **API-Key Governance:** [`../demos/API_KEY_GOVERNANCE_DEMO.md`](../demos/API_KEY_GOVERNANCE_DEMO.md) — Usage-based billing walkthrough
- **SaaS Governance:** [`../demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md`](../demos/SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md) — Seat-based billing walkthrough
- **Operational Runbook:** [`../RUNBOOK.md`](../RUNBOOK.md) — Monthly governance checklist
- **Service Offerings:** [`../SERVICE_OFFERINGS.md`](../SERVICE_OFFERINGS.md) — Managed operations deliverables

---

*Generated for AI Control Plane — Financial Governance Reference*
