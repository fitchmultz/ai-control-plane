# Financial Showback/Chargeback Report Template

**Reporting Period:** [YYYY-MM-DD to YYYY-MM-DD]  
**Generated:** [YYYY-MM-DD HH:MM UTC]  
**Prepared By:** [Name/Team]  
**Report Type:** Showback / Chargeback

---

## Executive Summary

| Metric | Value |
|--------|-------|
| **Total AI Spend** | $[AMOUNT] |
| **Total Requests** | [NUMBER] |
| **Total Tokens** | [NUMBER] |
| **Attribution Coverage** | [PERCENT]% |
| **Unattributed Usage** | [PERCENT]% |

### Key Findings

- [Key finding 1: e.g., "Platform Engineering represents 45% of total usage"]
- [Key finding 2: e.g., "15% of usage unattributed due to missing cost center mapping"]
- [Key finding 3: e.g., "Claude-sonnet-4-5 accounts for 60% of costs"]

---

## Data Sources

| Source | Coverage | Records | Confidence |
|--------|----------|---------|------------|
| Gateway Logs (PostgreSQL) | API-key traffic | [NUMBER] | High |
| OTEL Telemetry | Direct subscriptions | [NUMBER] | Medium |
| Compliance Exports | Vendor-verified | [NUMBER] | High |
| Seat Rosters | Subscription licenses | [NUMBER] | High |

### Data Quality Notes

- [Note on any data gaps, delays, or quality issues]
- [Any assumptions made in attribution]

---

## Allocation Summary

### By Cost Center (Chargeback View)

| Cost Center | Spend | % of Total | Seats (if applicable) |
|-------------|-------|------------|----------------------|
| [CC-12345] | $[AMOUNT] | [%] | [NUMBER] |
| [CC-54321] | $[AMOUNT] | [%] | [NUMBER] |
| [CC-99999] | $[AMOUNT] | [%] | [NUMBER] |
| **Unattributed** | $[AMOUNT] | [%] | — |
| **Total** | **$[AMOUNT]** | **100%** | **[NUMBER]** |

### By Team (Showback View)

| Team | Spend | Requests | Tokens | Top Model |
|------|-------|----------|--------|-----------|
| [platform] | $[AMOUNT] | [NUMBER] | [NUMBER] | [model] |
| [data-science] | $[AMOUNT] | [NUMBER] | [NUMBER] | [model] |
| [engineering] | $[AMOUNT] | [NUMBER] | [NUMBER] | [model] |
| [unknown] | $[AMOUNT] | [NUMBER] | [NUMBER] | — |
| **Total** | **$[AMOUNT]** | **[NUMBER]** | **[NUMBER]** | — |

---

## Usage-Based Billing (API Keys)

### Top Principals by Spend

| Principal | Team | CC | Spend | Requests | Avg $/Request |
|-----------|------|----|-------|----------|---------------|
| [svc-api-gateway] | [platform] | [CC-12345] | $[AMOUNT] | [NUMBER] | $[AMOUNT] |
| [usr-jdoe123] | [engineering] | [CC-54321] | $[AMOUNT] | [NUMBER] | $[AMOUNT] |
| [team-data-science] | [data-science] | [CC-99999] | $[AMOUNT] | [NUMBER] | $[AMOUNT] |

### Spend by Model

| Model | Spend | Requests | Tokens | % of Total |
|-------|-------|----------|--------|------------|
| [claude-sonnet-4-5] | $[AMOUNT] | [NUMBER] | [NUMBER] | [%] |
| [openai-gpt5.2] | $[AMOUNT] | [NUMBER] | [NUMBER] | [%] |
| [claude-haiku-4-5] | $[AMOUNT] | [NUMBER] | [NUMBER] | [%] |

---

## Seat-Based Billing (Subscriptions)

### Seat Assignment Summary

| Vendor | Seats Assigned | Cost/Seat | Total Cost | Chargeback Basis |
|--------|---------------|-----------|------------|------------------|
| [ChatGPT Enterprise] | [NUMBER] | $[AMOUNT] | $[AMOUNT] | Seat assignment |
| [Claude Team] | [NUMBER] | $[AMOUNT] | $[AMOUNT] | Seat assignment |
| [Copilot] | [NUMBER] | $[AMOUNT] | $[AMOUNT] | Seat assignment |
| **Total** | **[NUMBER]** | — | **$[AMOUNT]** | — |

### Seat Utilization (if available)

| Vendor | Active Users | Assigned Seats | Utilization % |
|--------|--------------|----------------|---------------|
| [ChatGPT Enterprise] | [NUMBER] | [NUMBER] | [%] |
| [Claude Team] | [NUMBER] | [NUMBER] | [%] |

---

## Exceptions and Anomalies

### Unattributed Usage

| Category | Spend | % of Total | Root Cause | Action Item |
|----------|-------|------------|------------|-------------|
| Missing key_alias | $[AMOUNT] | [%] | [reason] | [action] |
| Missing cc mapping | $[AMOUNT] | [%] | [reason] | [action] |
| Invalid alias format | $[AMOUNT] | [%] | [reason] | [action] |

### Anomalies Detected

| Anomaly | Severity | Details | Investigated |
|---------|----------|---------|--------------|
| [Spike in usage] | [High/Med/Low] | [description] | [Y/N] |
| [Budget overrun] | [High/Med/Low] | [description] | [Y/N] |
| [Unusual model usage] | [High/Med/Low] | [description] | [Y/N] |

---

## Reconciliation

### Internal Totals vs Provider Invoices

| Provider | Internal Total | Invoice Total | Variance | Variance % | Explained |
|----------|---------------|---------------|----------|------------|-----------|
| [OpenAI] | $[AMOUNT] | $[AMOUNT] | $[AMOUNT] | [%] | [Y/N] |
| [Anthropic] | $[AMOUNT] | $[AMOUNT] | $[AMOUNT] | [%] | [Y/N] |
| **Total** | **$[AMOUNT]** | **$[AMOUNT]** | **$[AMOUNT]** | **[%]** | — |

### Reconciliation Notes

- [Explanation of any variances >5%]
- [Timing differences, currency conversions, etc.]

### Reconciliation Checklist

- [ ] Gateway totals match provider invoices within 5%
- [ ] Seat counts match vendor admin consoles
- [ ] All variances >5% explained and documented
- [ ] Finance/Procurement reviewed and approved

---

## Appendices

### A. Methodology

**Attribution Logic:**
- `key_alias` parsed using `__` delimiter
- Team extracted from `__team-<name>` segment
- Cost center extracted from `__cc-<number>` segment
- Unparseable aliases mapped to `unknown`

**Data Query:**
```sql
-- Primary allocation query
SELECT
  COALESCE(v.key_alias, 'unknown') AS principal,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  CASE
    WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)')
    ELSE 'unknown'
  END AS cost_center,
  SUM(s.spend) AS total_spend
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" BETWEEN '[START]' AND '[END]'
GROUP BY v.key_alias;
```

### B. Key Aliases with Missing Attribution

| Alias | Spend | Issue | Recommended Action |
|-------|-------|-------|-------------------|
| [example-legacy-key] | $[AMOUNT] | No team segment | Update alias or manual map |
| [temp-test-key] | $[AMOUNT] | No cc segment | Add cc or use default |

### C. Glossary

| Term | Definition |
|------|------------|
| **Showback** | Informational reporting without internal billing |
| **Chargeback** | Internal cost allocation to cost centers |
| **Attribution** | Mapping usage to organizational entities |
| **Principal** | Entity making API requests (user, service, team) |

---

## Approvals

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Prepared By | [NAME] | _________________ | [DATE] |
| Reviewed By (FinOps) | [NAME] | _________________ | [DATE] |
| Approved By (Procurement/Finance) | [NAME] | _________________ | [DATE] |

---

## Distribution

- [ ] Finance (chargeback processing)
- [ ] Procurement (invoice reconciliation)
- [ ] Team Leads (showback)
- [ ] Platform Engineering (infrastructure costs)
- [ ] SecOps (anomaly review)

---

*Template Version: 1.0*  
*Based on: AI Control Plane Financial Governance Policy*
