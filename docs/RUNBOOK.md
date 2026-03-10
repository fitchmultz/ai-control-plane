# AI Control Plane - Operational Runbook

## Purpose

This runbook provides day-to-day operational procedures, incident response guidance, and maintenance workflows for the AI Control Plane **demo environment**. It serves as a navigation index to existing operational resources rather than duplicating content.

> **Important — Demo Environment Focus**: This runbook is designed for local development and demo deployments. For **production deployments**, see the [Production Handoff Runbook](./deployment/PRODUCTION_HANDOFF_RUNBOOK.md).

**Relationship to other documentation:**
- **demo/README.md** - Quick start, environment setup, and basic troubleshooting
- **docs/DEPLOYMENT.md** - Deployment architecture and network configuration
- **docs/DATABASE.md** - Deep-dive database schema and advanced troubleshooting
- **docs/security/DETECTION.md** - Detection rule details and SIEM integration
- **docs/security/SIEM_INTEGRATION.md** - SIEM integration methods and query examples
- **docs/deployment/PRODUCTION_HANDOFF_RUNBOOK.md** - Production deployment operations and handoff procedures
- **This runbook** - Demo environment operational procedures, incident response, and maintenance workflows

---

## Quick Reference Card

| Task | Command |
|------|---------|
| Start services | `make up` |
| Stop services | `make down` |
| Restart services | `make restart` |
| Health check | `make health` |
| View logs | `make logs` |
| Container status | `make ps` |
| Database status | `make db-status` |
| Create backup | `make db-backup` |
| Restore backup | `make db-restore` |
| Validate detection rules | `make validate-detections` |
| Generate release evidence bundle | `make release-bundle` |
| Generate virtual key | `make key-gen ALIAS=foo BUDGET=10.00` |
| TLS mode start | `make up-tls` |
| TLS health check | `make tls-health` |

> **Open-source release note:** legacy scorecard targets (`make governance-report*`) remain compatibility stubs only. Use `make chargeback-report` for the public chargeback/showback workflow.

---

## Network Hardening Prerequisite

For production-grade bypass prevention, runbook operations assume customer-managed controls are already deployed:

- Default-deny egress so only approved gateway paths can reach provider APIs
- SWG/CASB policies for browser-based AI governance
- MDM/endpoint controls enforcing managed tool configuration

See `docs/DEPLOYMENT.md` (Network Configuration) and `docs/demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md` for required control patterns and ownership boundaries.

---

## Routine Maintenance Procedures

### 3.1 Daily Operations

**Health Verification:**
```bash
# Quick health check (recommended daily)
make health

# Detailed diagnostics
make doctor
```

**Log Inspection:**
```bash
# Follow all service logs
make logs

# View specific service logs
docker compose -f demo/docker-compose.yml logs litellm
docker compose -f demo/docker-compose.yml logs postgres
```

> **Security Warning:** When using subscription mode (OAuth), tokens may appear in logs. Review logs before sharing and never commit log files.

**Container Status:**
```bash
make ps
# Or: docker compose -f demo/docker-compose.yml ps
```

### 3.2 Weekly Operations

**Database Status Review:**
```bash
make db-status
```

This displays:
- Database size and connection count
- Table statistics and row counts
- Virtual keys and budget usage
- Recent audit log entries

**Detection Rule Execution:**
```bash
# Run all detection rules
make validate-detections

# Focus on high-severity findings
make validate-detections
```

**Budget Review:**
```bash
# Check budget status for all keys
make db-status
# Look for section 4: "Budget Usage"
```

**Backup Verification:**
```bash
# List recent backups
ls -la demo/backups/

# Verify at least one backup exists from the past week
find demo/backups/ -name "litellm-backup-*.sql.gz" -mtime -7
```

### 3.3 Monthly Operations

**Full Database Backup:**
```bash
make db-backup
```

**Audit Log Review for Trends:**
```bash
# Connect to database and analyze trends
docker exec -it $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm -c "
    SELECT DATE(\"startTime\") as day,
           COUNT(*) as requests,
           SUM(spend) as total_spend
    FROM \"LiteLLM_SpendLogs\"
    WHERE \"startTime\" > NOW() - INTERVAL '30 days'
    GROUP BY day ORDER BY day DESC;
  "
```

**Key Rotation Assessment:**
- Review all active keys via `make db-status`
- Identify keys approaching expiration
- Rotate keys that have been active > 90 days

**Storage Usage Check:**
```bash
# Check Docker volume usage
docker system df -v | grep ai-control-plane

# Check log file sizes
du -sh demo/logs/
```

---

## Governance Reporting

Governance scorecard automation is **not included** in this open-source release. Compatibility targets are retained so older workflows fail fast with a clear message.

### Quick Start

**Compatibility targets (expected to exit code `2` in open-source release):**
```bash
# Legacy markdown scorecard target (stub)
make chargeback-report

# Legacy JSON scorecard target (stub)
make chargeback-report OUTPUT_FORMAT=json

# Legacy 7-day scorecard target (stub)
make chargeback-report REPORT_MONTH=YYYY-MM

# Canonical readiness evidence flow
make release-bundle
make release-bundle-verify
```

### Scorecard Metrics Explained

The governance scorecard provides the following metrics:

#### Executive Summary
| Metric | Description | Interpretation |
|--------|-------------|----------------|
| **Total Requests** | Number of API requests processed | Baseline activity level |
| **Total Spend** | Cost of AI usage in USD | Budget consumption tracking |
| **Governance Posture** | Overall health indicator (HEALTHY/AT_RISK/CRITICAL) | Action priority level |
| **Report Period** | Time window for the data | Context for trend analysis |

#### Usage Statistics
| Metric | Description | Action Trigger |
|--------|-------------|----------------|
| **Total Tokens** | Input + output tokens consumed | Capacity planning |
| **Successful Requests** | Requests completed successfully | Availability metric |
| **Blocked/Failed** | Policy violations or errors | Security/policy review |

#### Detection Findings
| Severity | Meaning | Response Time |
|----------|---------|---------------|
| **High** | Security incidents, policy violations | Immediate (hours) |
| **Medium** | Anomalies, unusual patterns | Within 24 hours |
| **Low** | Budget warnings, minor issues | Next business day |

**Detection Categories:**
- **Policy Violation**: Non-approved models, sensitive data patterns
- **Anomaly**: Token spikes, rapid request rates
- **Availability**: High error rates, auth failures
- **Cost Management**: Budget exhaustion warnings
- **Security**: Failed authentication attempts

#### Top Principals
- **By Spend**: Identify highest-cost users/services
- **By Requests**: Identify highest-volume consumers

**Actions Enabled:**
- Budget reallocation discussions
- Usage optimization opportunities
- Access pattern analysis

#### Top Models
Shows which AI models are most heavily used, enabling:
- Model cost optimization
- Approved model list validation
- Capacity planning

#### Budget Risk Alerts
Identifies keys approaching budget limits (<20% remaining):
- Proactive budget increase decisions
- Usage review triggers
- Key rotation planning

### Interpreting the Scorecard

#### HEALTHY Posture
```
✓ No high-severity detections
✓ No blocked policy violations
✓ Budgets within limits
✓ Normal request patterns
```
**Action**: Continue regular monitoring

#### AT_RISK Posture
```
⚠ Blocked requests detected (policy violations)
⚠ Medium-severity anomalies present
⚠ Keys approaching budget limits
```
**Action**: Review blocked requests, investigate anomalies

#### CRITICAL Posture
```
✗ High-severity security detections
✗ Failed authentication patterns
✗ Sensitive data policy violations
```
**Action**: Immediate investigation required, consider key revocation

### CI Integration

In the open-source release, governance scorecard targets are compatibility stubs and intentionally exit with code `2`.

```bash
# Assert expected stub behavior
make chargeback-report

# Use canonical readiness artifacts in CI
make release-bundle
make release-bundle-verify
```

### Monthly Governance Review Checklist

- [ ] Run `make release-bundle` and review readiness evidence (`docs/release/PRESENTATION_READINESS_TRACKER.md`)
- [ ] Review governance posture trend (improving/stable/degrading)
- [ ] Check top 5 principals for budget allocation accuracy
- [ ] Review high-severity detection findings
- [ ] Validate top models against approved model list
- [ ] Address keys with <20% budget remaining
- [ ] Document lessons learned and policy adjustments

### Demo Scorecard Data

For demonstration without live traffic, fixture data is available:

```bash
# Load fixture data into evidence pipeline
cp demo/fixtures/scorecard_fixture_data.jsonl demo/logs/normalized/evidence.jsonl

# Run detection against fixtures
make validate-detections
```

---

## Financial Governance and Chargeback

This section covers the operational workflow for monthly showback/chargeback processes.

### Overview

Financial governance has two components:
1. **Budget Enforcement** — Real-time gateway controls (covered in Detection section)
2. **Chargeback/Showback** — Monthly cost allocation to teams/cost centers

See [FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md](policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md) for complete policy details.

### Monthly Chargeback Cycle

#### Week 1: Data Collection

```bash
# Capture current governance state and evidence bundle
make db-status
make release-bundle
make release-bundle-verify

# Legacy scorecard command remains a compatibility stub in open-source release
make chargeback-report

# Export full usage data for finance analysis
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
COPY (
  SELECT
    s.\"startTime\" AS timestamp,
    COALESCE(v.key_alias, 'unknown') AS principal,
    s.model,
    s.\"prompt_tokens\",
    s.\"completion_tokens\",
    s.spend,
    CASE WHEN v.key_alias LIKE '%__team-%' THEN substring(v.key_alias from '__team-([^_]+)') ELSE 'unknown' END AS team,
    CASE WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)') ELSE 'unknown' END AS cost_center
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" > NOW() - INTERVAL '30 days'
  ORDER BY s.\"startTime\" DESC
) TO STDOUT WITH CSV HEADER;
" > ai_usage_$(date +%Y-%m).csv
```

#### Week 2: Attribution and Allocation

Parse `key_alias` for cost-center mapping:

```sql
-- Spend by cost center (for chargeback)
SELECT
  CASE
    WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)')
    ELSE 'unknown-cc'
  END AS cost_center,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  COUNT(*) AS request_count
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '30 days'
GROUP BY cost_center
ORDER BY SUM(s.spend) DESC;
```

#### Week 3: Reconciliation

Compare internal totals to provider invoices:

```bash
# Get total spend from gateway
docker exec $(docker compose ps -q postgres) psql -U litellm -d litellm -c "
SELECT ROUND(SUM(spend)::numeric, 4) AS total_spend
FROM \"LiteLLM_SpendLogs\"
WHERE \"startTime\" > NOW() - INTERVAL '30 days';
"

# Compare to provider invoice total
# Acceptable variance: <5%
```

#### Week 4: Reporting

Use the [Financial Showback/Chargeback Report Template](templates/FINANCIAL_SHOWBACK_CHARGEBACK_REPORT.md):

1. Fill in reporting period and data sources
2. Complete allocation summary by cost center
3. Document exceptions and unattributed usage
4. Complete reconciliation checklist
5. Obtain approvals from Finance/Procurement

### Key Alias Convention

For chargeback to work, keys must follow the attribution convention:

| Pattern | Example | Use For |
|---------|---------|---------|
| Service | `svc-<name>__team-<team>__cc-<cc>` | `svc-api-gateway__team-platform__cc-12345` |
| User | `usr-<id>__team-<team>__cc-<cc>` | `usr-jdoe123__team-eng__cc-54321` |
| Team | `team-<team>__cc-<cc>` | `team-data-science__cc-99999` |

**Verification:**
```bash
# Check for keys missing attribution
make db-status | grep -v "__team-" | grep -v "__cc-"
```

### CSV Export for Finance Systems

```bash
# Governance JSON export targets are stubs in the open-source release.
# Use SQL export (see Week 1 data-collection query above) to produce CSV artifacts.
make db-status
```

### Reconciliation Checklist

```markdown
## Monthly Reconciliation

### Gateway vs Provider Invoice
- [ ] Record internal total spend
- [ ] Record provider invoice total
- [ ] Record variance percentage (<5% target)
- [ ] Document variance explanation

### Attribution Coverage
- [ ] Record usage with valid cost-center mapping
- [ ] Record unattributed usage percentage
- [ ] Confirm exceptions are documented

### Approvals
- [ ] Reconciliation prepared and recorded in monthly report
- [ ] FinOps review completed (name and date captured in report)
- [ ] Finance approval completed (name and date captured in report)
```

---

## Service Lifecycle Management

### 4.1 Starting Services

**Standard Startup:**
```bash
make up
```

Expected startup sequence:
1. PostgreSQL container starts first
2. LiteLLM waits for PostgreSQL to be healthy
3. LiteLLM initializes database schema (first startup only)

**TLS Mode:**
```bash
make up-tls
```

**Verification:**
```bash
# Wait 10-15 seconds for services to initialize, then:
make health
```

### 4.2 Stopping Services

**Preserve Data (Recommended):**
```bash
make down
```
- Stops containers but preserves volumes
- Data remains intact for next startup

**Complete Cleanup (Destructive):**
```bash
# ALWAYS backup first!
make db-backup

# Interactive shells: prompts for confirmation (Continue? [y/N])
make clean

# For scripts/automation:
make clean-force
```

**Warning:** `make clean` deletes all volumes including database data. This action cannot be undone. Interactive shells are prompted for confirmation; non-interactive automation should use `make clean-force`.

### 4.3 Restarting Services

**Standard Restart:**
```bash
make restart
```

**TLS Mode:**
```bash
make restart-tls
```

---

## Database Operations

### 5.1 Checking Database Status

**Quick Status:**
```bash
make db-status
```

**What to Look For:**
- All expected tables exist (section 2: "Schema Verification")
- Recent virtual keys are visible (section 3)
- Budget usage is within expected ranges (section 4)
- Detection summary shows recent spend-log activity (section 5)

**Manual Verification:**
```bash
# Test database connectivity
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  pg_isready -U litellm

# Get PostgreSQL version
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm -c "SELECT version();"
```

See also: [docs/DATABASE.md](DATABASE.md) section 5

### 5.2 Backup Procedures

**Automated Backup:**
```bash
make db-backup
```

Creates: `demo/backups/litellm-backup-YYYYMMDD-HHMMSS.sql.gz`

**Custom Name:**
```bash
./scripts/acpctl.sh db backup my-custom-backup
```

**Retention Cleanup (7 days):**
```bash
find demo/backups/ -name "litellm-backup-*.sql.gz" -mtime +7 -delete
```

**Pre-Clean Backup:**
```bash
# ALWAYS backup before make clean!
make db-backup

# Interactive shells: prompts for confirmation
# Non-interactive scripts: use make clean-force
make clean
```

### 5.3 Restore Procedures

**Restore Command (Current Open-Source Release Behavior):**
```bash
# Attempt restore using latest backup (auto-detected)
make db-restore

# Restore a specific backup file
./scripts/acpctl.sh db restore demo/backups/litellm-backup-20240128-143052.sql.gz
```

Current typed restore flow validates prerequisites and backup input, then prints guided manual restore instructions.

**Manual Restore Path (when required):**
```bash
gunzip < demo/backups/litellm-backup-20240128-143052.sql.gz \
  | docker exec -i $(docker compose -f demo/docker-compose.yml ps -q postgres) \
      psql -U litellm -d litellm
```

**Safety Notes:**
- Restore overwrites ALL existing data
- LiteLLM is stopped during restore
- Always backup current state before restoring
- Validate backup integrity before restore (e.g., `gzip -t <backup.sql.gz`)

### 5.4 Direct Database Access

**Interactive psql:**
```bash
docker exec -it $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm
```

**Single Query:**
```bash
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm -c "SELECT * FROM \"LiteLLM_VerificationToken\" LIMIT 5;"
```

See also: [docs/DATABASE.md](DATABASE.md) section 7

---

## Key Management

### 6.1 Generating Virtual Keys

Use the canonical key lifecycle surface:

```bash
# Standard key
make key-gen ALIAS=my-key BUDGET=10.00

# Role-shaped presets
make key-gen-dev ALIAS=my-dev-key
make key-gen-lead ALIAS=my-lead-key

# Revoke a key
make key-revoke ALIAS=<alias>

# acpctl equivalents
./scripts/acpctl.sh key gen my-key --budget 10.00
./scripts/acpctl.sh key gen-dev my-dev-key
./scripts/acpctl.sh key gen-lead my-lead-key
./scripts/acpctl.sh key revoke <alias>
```

**Required environment:**

```bash
# Read LITELLM_MASTER_KEY as data only (never source or grep demo/.env)
export LITELLM_MASTER_KEY="$(./scripts/acpctl.sh env get LITELLM_MASTER_KEY)"
```

> Approval-queue and dry-run key-generation script flows from older private iterations are not part of the open-source release operator interface.

### 6.2 Key Lifecycle

**Creating Keys with Expiry:**
```bash
# Generate key that expires in 30 days
curl -s -X POST "http://127.0.0.1:4000/key/generate" \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "key_alias": "temp-key",
    "max_budget": 5.00,
    "expires": "2026-03-01T00:00:00Z"
  }'
```

**Revoking Keys:**
```bash
# List keys first
make db-status

# Revoke via LiteLLM API
curl -s -X POST "http://127.0.0.1:4000/key/delete" \
  -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key_alias": "key-to-revoke"}'
```

**Key Rotation:**
1. Generate new key with same permissions
2. Update clients to use new key
3. Verify old key is no longer in use
4. Revoke old key

### 6.3 Key Monitoring

**View Active Keys:**
```bash
make db-status
# Check section 3: "Virtual Keys"
```

**Check Budget Usage:**
```bash
# Check section 4 in db-status output
make db-status | grep -A 20 "Budget Usage"
```

**Detection Rules for Keys:**
- **DR-004**: Budget Exhaustion Warning
- **DR-007**: Budget Threshold Alert

---

## Health Verification

### 7.1 Automated Health Checks

**Full Check:**
```bash
make health
```

**Operator Command:**
```bash
make health
```

**Exit Codes:**
- `0`: All services healthy
- `1`: One or more services unhealthy

### 7.2 Manual Verification

**Container Status:**
```bash
docker compose -f demo/docker-compose.yml ps
```

**LiteLLM Health Endpoint:**
```bash
curl http://127.0.0.1:4000/health
```

**PostgreSQL Connectivity:**
```bash
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  pg_isready -U litellm
```

### 7.3 Health Check Components

The health check script verifies:

1. **Docker Container Status**
   - PostgreSQL container running
   - LiteLLM container running
   - OTEL collector running (optional)

2. **LiteLLM Gateway Health Endpoint**
   - HTTP 200 or 401 response

3. **LiteLLM Models Endpoint**
   - HTTP 200 or 401 response

4. **PostgreSQL Connectivity**
   - Database accepting connections
   - Tables exist

5. **OTEL Collector Health** (optional)
   - HTTP 200 on port 4318

---

## Security Monitoring & Detection

### 8.1 Running Detection Rules

**All Rules:**
```bash
make validate-detections
```

**Specific Rule:**
```bash
make validate-detections
```

**By Severity:**
```bash
# High severity only
make validate-detections

# Medium severity
make validate-detections
```

**JSON Output (for SIEM):**
```bash
make validate-detections
```

**Dry-Run Mode (Preview Before Executing):**
```bash
# Preview all enabled rules and their SQL queries
make validate-detections

# Preview high-severity rules only
make validate-detections

# Preview a specific rule with full details
make validate-detections

# Preview as JSON (useful for programmatic validation)
make validate-detections
```

The `--dry-run` flag:
- Lists selected rules based on filters (--rule, --severity)
- Shows rule metadata (ID, name, severity, category, description)
- Displays SQL queries that would be executed
- Shows rule parameters (thresholds, windows, etc.)
- Does not require Docker, PostgreSQL, jq, or evidence files
- Validates configuration and exits with code 0

**Recommended workflow:** Use `--dry-run` to validate detection rule changes before deploying them, especially when modifying `demo/config/detection_rules.yaml` or testing new filter combinations.

### 8.2 Detection Rules Reference

| Rule ID | Name | Severity | Category | Description |
|---------|------|----------|----------|-------------|
| DR-001 | Non-Approved Model Access | High | Policy Violation | Requests to models not in approved list |
| DR-002 | Token Usage Spike | Medium | Anomaly Detection | Unusual token consumption per key |
| DR-003 | High Block/Error Rate | Medium | Availability | Keys with >10% error rate |
| DR-004 | Budget Exhaustion Warning | Low | Cost Management | Keys with <20% budget remaining |
| DR-005 | Rapid Request Rate | Medium | Anomaly Detection | >60 requests/minute |
| DR-006 | Failed Authentication Attempts | High | Security | ≥5 failed auth attempts |
| DR-007 | Budget Threshold Alert | Medium | Cost Management | Keys with ≥80% budget spent |

**Note:** Content-based DLP detection is not included in this repo. See
[docs/security/DETECTION.md](security/DETECTION.md) for DLP alternatives
(Presidio guardrails, external telemetry).

See [docs/security/DETECTION.md](security/DETECTION.md) for detailed SQL queries and remediation steps.

### 8.3 SIEM Integration

**Demo Commands:**
```bash
make validate-detections       # Run interactive SIEM demo
make validate-siem-schema     # View normalized schema
make validate-siem-queries    # View SIEM query examples
make detection-normalized     # View recent telemetry snapshot
```

See [docs/security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md) for complete SIEM integration guide.

---

## Incident Response Procedures

### 9.1 Service Outage

**Symptoms:**
- Health check fails
- Clients cannot connect to gateway
- High error rates

**Immediate Actions:**
```bash
# 1. Check service status
make ps

# 2. View recent logs
make logs

# 3. Check specific service logs
docker compose -f demo/docker-compose.yml logs --tail=50 litellm
```

**Recovery Steps:**
```bash
# Restart services
make restart

# Verify recovery
make health
```

**Escalation Criteria:**
- Multiple restart attempts fail
- Database corruption suspected
- Network connectivity issues persist

### 9.2 Database Issues

**Symptoms:**
- Connection errors
- Slow queries
- LiteLLM cannot start

**Diagnosis:**
```bash
# Check database status
make db-status

# Check PostgreSQL logs
docker compose -f demo/docker-compose.yml logs postgres

# Test connectivity
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  pg_isready -U litellm
```

**Recovery:**
```bash
# Option 1: Restart PostgreSQL
docker compose -f demo/docker-compose.yml restart postgres

# Option 2: Restore from backup (if corruption suspected)
make db-backup  # Backup current state first
make db-restore
```

### 9.3 Security Incidents

**Detection:**
```bash
# Run all detection rules
make validate-detections

# Focus on high-severity rules
make validate-detections
```

**Response Steps:**

1. **Run Detection Rules:**
   ```bash
   make validate-detections
   ```

2. **Identify Affected Keys/Users:**
   ```bash
   # Review detection output for key_alias and user_id
   make db-status
   ```

3. **Revoke Compromised Keys:**
   ```bash
   curl -X POST "http://127.0.0.1:4000/key/delete" \
     -H "Authorization: Bearer $LITELLM_MASTER_KEY" \
     -H "Content-Type: application/json" \
     -d '{"key_alias": "compromised-key"}'
   ```

4. **Review Audit Logs:**
   ```bash
   # Query recent suspicious activity
   docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
     psql -U litellm -d litellm -c "
      SELECT s.* FROM \"LiteLLM_SpendLogs\" s
      JOIN \"LiteLLM_VerificationToken\" v
        ON s.api_key = v.token
      WHERE v.key_alias = 'compromised-key'
      ORDER BY s.\"startTime\" DESC LIMIT 50;
     "
   ```

5. **Implement Preventive Measures:**
   - Generate new keys for affected users
   - Adjust rate limits or budgets
   - Update detection rule thresholds

See [docs/security/DETECTION.md](security/DETECTION.md) section on remediation for detailed guidance.

### 9.4 Log Safety During Incident Response

**Secure Logging Defaults:**

Demo scenario scripts automatically redact secrets in terminal output:
- Key previews shown as `sk-abc123def...` (12 chars + ellipsis)
- Full-key diagnostic lines rendered as `sk-[REDACTED]`
- Bearer tokens replaced with `[REDACTED]`
- Export commands show previews, not full values

**Safe Output (Default):**
```bash
./scenario_1_api_path.sh --verbose
# Output: export OPENAI_API_KEY="virtual-key-abc123..."
```

**When You Need Full Secrets (Debugging Only):**
```bash
# Explicit opt-in required
./scenario_1_api_path.sh --verbose --reveal-secrets

# Or via environment variable
DEMO_REVEAL_SECRETS=1 ./scenario_1_api_path.sh --verbose
```

**Incident Response Protocol:**

1. **Never commit terminal transcripts with revealed secrets**
2. **Never paste `--reveal-secrets` output into tickets/chat**
3. **Always validate before sharing:**
   ```bash
   # Check tracked repository content for leaked secrets before sharing logs
   make secrets-audit
   
   # If audit finds leaks, redact them:
   # - Replace tokens with [REDACTED]
   # - Replace API keys with sk-...[REDACTED]
   ```

4. **If secrets were accidentally exposed:**
   - Rotate exposed keys immediately: `make key-revoke ALIAS=<alias>`
   - Revoke at provider console (OpenAI, Anthropic, etc.)
   - Document incident with rotated key IDs

**For Incident Documentation:**
```bash
# Safe to include in tickets (redacted by default)
./scenario_1_api_path.sh --verbose > incident-repro.txt

# Verify no tracked repository content leaked before attaching
LOGS_DIR=./ make secrets-audit
```

### 9.5 Budget Exhaustion

**Detection:**
- DR-004: Budget Exhaustion Warning
- DR-007: Budget Threshold Alert

**Response:**
```bash
# 1. Identify affected keys
make validate-detections

# 2. Check current budget status
make db-status

# 3. Option A: Increase budget for existing key
# (via LiteLLM API or regenerate with higher limit)

# 3. Option B: Generate new key
make key-gen ALIAS=new-key BUDGET=20.00
```

### 9.6 Authentication Failures

**Detection:**
- DR-006: Failed Authentication Attempts
- Health check shows 401 errors

**Response:**
```bash
# 1. Check detection output for source patterns
make validate-detections

# 2. Verify key validity
make db-status

# 3. Rotate compromised keys
make key-gen ALIAS=replacement-key BUDGET=10.00

# 4. Investigate source IP patterns in logs
docker compose -f demo/docker-compose.yml logs litellm | grep "401"
```

### 9.7 Rapid Response (Containment & Key Lifecycle)

This section provides **operational procedures** for rapid incident response, including key revocation, rotation, and emergency containment actions.

#### Operational Terminology

| Term | Definition | Default Action |
|------|------------|----------------|
| **Quarantine** | Immediate revocation of a key alias in the gateway | Recommended first response for suspected compromise |
| **Rotation** | Issue replacement key, update clients, verify old key unused, revoke old | Standard key lifecycle management |
| **Tool Restriction** | MDM/managed configuration enforcement | Endpoint-level containment |
| **Egress Lockdown** | Default-deny provider endpoints except from gateway | Network-level emergency containment |

#### First 15 Minutes Checklist

When a security incident is detected (high-severity detection rule triggered):

**Minutes 0-5: Assess**
```bash
# 1. Run high-severity detections
make validate-detections
# Or: make validate-detections

# 2. Identify affected key aliases and users
make db-status
# Look for unusual patterns in section 3 (Virtual Keys) and 5 (Detection Summary)
```

**Minutes 5-10: Contain**
```bash
# 3. Revoke compromised key(s)
make key-revoke ALIAS=<alias>
# Or: make key-revoke ALIAS=compromised-key-alias

# 4. Verify key is revoked (should fail authentication)
make db-status | grep compromised-key-alias
# Key should no longer appear or show as revoked
```

**Minutes 10-15: Document**
```bash
# 5. Query SpendLogs for forensics evidence
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm -c "
    SELECT 
      TO_CHAR(s.\"startTime\", 'YYYY-MM-DD HH24:MI:SS') AS timestamp,
      COALESCE(v.key_alias, 'unknown') AS key_alias,
      s.model,
      s.status,
      ROUND(s.spend::numeric, 6) AS spend,
      s.\"prompt_tokens\" + s.\"completion_tokens\" AS total_tokens
    FROM \"LiteLLM_SpendLogs\" s
    LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
    WHERE v.key_alias = 'compromised-key-alias'
       OR s.\"startTime\" > NOW() - INTERVAL '1 hour'
    ORDER BY s.\"startTime\" DESC
    LIMIT 50;
  "

# 6. Capture detection output for incident ticket
make validate-detections
```

#### Key Revocation Commands

**Single Key Revocation:**
```bash
# Revoke a specific key by alias
make key-revoke ALIAS=<alias>

# Dry run (preview without executing)
make key-revoke ALIAS=my-compromised-key
```

**Bulk Revocation (from detection findings):**
```bash
# Export high-severity findings
make validate-detections

# Preview bulk revoke (safe-by-default)
make key-revoke ALIAS=<alias>

# Execute bulk revoke (requires explicit confirmation)
make key-revoke ALIAS=<alias>
```

#### Automated Key Suspension

The system can automatically suspend keys when high-severity detection rules trigger. This reduces MTTR from minutes/hours to seconds.

**Default Auto-Response Rules:**

| Rule | Trigger | Action | Grace Period |
|------|---------|--------|--------------|
| DR-001 | Non-approved model access | Immediate suspension | None |
| DR-006 | Failed authentication (≥5 attempts) | Suspension after 5min | 5 minutes |

**Break-Glass Override:**

If you need to run detections without triggering auto-response:

```bash
# Option 1: Environment variable
export ACP_DISABLE_AUTO_RESPONSE=1
make validate-detections

# Option 2: Command-line flag
make validate-detections
```

**Verifying Auto-Response Status:**

```bash
# Check if auto-response is globally disabled
echo "ACP_DISABLE_AUTO_RESPONSE=${ACP_DISABLE_AUTO_RESPONSE:-<not set>}"

# Check recent auto-response actions
tail -20 demo/logs/auto_response.log

# View all successful auto-suspensions
cat demo/logs/auto_response.log | jq 'select(.status == "SUCCESS")'
```

**Manual Recovery After Auto-Suspension:**

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

# 3. If key was suspended in error, generate a replacement
make key-gen ALIAS=replacement-key BUDGET=10.00

# 4. Document the incident (auto-suspension is logged)
```

**Audit Trail for Automated Actions:**

All automated suspensions are logged with:
- Detection ID and rule ID
- Target key alias
- Timestamp (UTC)
- Action status (SUCCESS/FAILED)
- Actor (always "auto_response_system")

This provides the same audit trail as manual actions for compliance purposes:

```bash
# View complete audit log
cat demo/logs/auto_response.log | jq .

# Generate summary for incident report
cat demo/logs/auto_response.log | jq -s '
  group_by(.rule_id) | 
  map({
    rule: .[0].rule_id, 
    total: length, 
    successful: map(select(.status == "SUCCESS")) | length
  })
'
```

#### Key Rotation Procedure

Standard key rotation sequence (non-emergency):

```bash
# Step 1: Generate new key with same permissions
make key-gen ALIAS=my-service-key-v2 BUDGET=10.00

# Step 2: Capture the generated key value from command output
# (store it in your secret manager / deployment system before rollout)

# Step 3: Update clients to use new key (application-specific)
# ... deploy new configuration ...

# Step 4: Verify old key is no longer in use
make db-status
# Check that old key shows no recent activity in SpendLogs

# Step 5: Revoke old key
make key-revoke ALIAS=my-service-key-v1
```

#### Emergency Egress Lockdown

For severe compromises requiring immediate network containment:

**Prerequisites:**
- Network egress controls must be in place (see [Network Endpoint Enforcement Demo](./demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md))
- Default-deny posture configured at firewall/proxy level

**Lockdown Procedure:**
```bash
# Emergency egress lockdown checklist:
# 1. Enable default-deny rule for AI provider endpoints
# 2. Ensure only gateway IP can reach provider APIs
# 3. Block direct outbound connections from endpoints

# See detailed procedures in:
# docs/demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md
```

**Rollback:**
```bash
# To lift egress lockdown:
# 1. Verify threat has been contained
# 2. Document findings in incident ticket
# 3. Re-enable normal egress with monitoring
# 4. Rotate all keys that were active during incident
```

#### Evidence to Capture

For incident documentation and forensics:

```bash
# 1. Detection findings
make validate-detections

# 2. Database state (keys, budgets, recent activity)
make db-status > incident-db-state.txt

# 3. Gateway logs (recent)
docker compose -f demo/docker-compose.yml logs --since=1h litellm > incident-gateway-logs.txt

# 4. Audit log query
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  psql -U litellm -d litellm -c "
    COPY (
      SELECT 
        TO_CHAR(s.\"startTime\", 'YYYY-MM-DD HH24:MI:SS') AS timestamp,
        COALESCE(v.key_alias, 'unknown') AS key_alias,
        s.model, s.status, s.spend
      FROM \"LiteLLM_SpendLogs\" s
      LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
      WHERE s.\"startTime\" > NOW() - INTERVAL '24 hours'
      ORDER BY s.\"startTime\" DESC
    ) TO STDOUT WITH CSV HEADER;
  " > incident-audit-log.csv
```

#### Dual-Control / Break-Glass Process

For emergency actions requiring elevated authorization:

**Roles:**
| Role | Responsibility |
|------|----------------|
| **Incident Commander** | Declares incident, decides on break-glass actions |
| **Approver** | Authorizes emergency changes (separate from operator) |
| **Operator** | Executes commands, captures evidence |

**Approval and Audit Trail Requirements:**
- Ticket ID: Reference incident/case number
- Timestamp: Record all actions in UTC
- Reason: Document why break-glass was required
- Command Log: Capture exact commands executed (redact secrets)
- Outputs: Screenshot or save command outputs
- Rollback Plan: Document how to revert if needed

**Break-Glass Key Handling:**
- Stored in vault (never printed or hardcoded)
- Injected via environment variable at runtime
- Rotated immediately after emergency use
- Usage logged with incident ticket reference

```bash
# Example break-glass workflow:
# 1. Incident commander declares incident #INC-2026-0001
# 2. Approver authorizes emergency key revocation
# 3. Operator executes with incident reference:

INCIDENT_ID="INC-2026-0001"
make key-revoke ALIAS=<alias>

# 4. Capture command output and incident reference in ticket:
#    Emergency: $INCIDENT_ID - suspected credential compromise
# 5. Break-glass credentials rotated post-incident
```

---

## Disaster Recovery

### 10.1 Backup Strategy

**Automated Backups:**
```bash
# Create timestamped backup
make db-backup
```

**Retention Policy (Recommended):**
- Demo environments: 7 days
- Production environments: 30+ days with offsite storage

**Offsite Backup (Production):**
```bash
# Create backup
make db-backup

# Copy to offsite storage (example: S3)
aws s3 cp demo/backups/litellm-backup-*.sql.gz s3://my-backup-bucket/ai-control-plane/
```

### 10.2 Recovery Procedures

**Service Recovery (Infrastructure Issues):**
```bash
# After infrastructure is restored
make up
make health
```

**Data Recovery:**
```bash
# Restore from backup
make db-restore

# Verify restore
make db-status
```

**Full Rebuild (Data Loss Scenario):**
```bash
# WARNING: This results in complete data loss
# Interactive: prompts for confirmation; Non-interactive: requires FORCE=1
make clean
make up
make health
```

### 10.3 Testing Recovery

**Monthly Test Procedure:**

1. **Create Test Backup:**
   ```bash
   make db-backup
   ```

2. **Verify Backup Integrity:**
   ```bash
   gunzip -t demo/backups/litellm-backup-*.sql.gz
   ```

3. **Test Restore (on separate instance or after backup):**
   ```bash
   make db-restore
   ```

4. **Verify Data:**
   ```bash
   make db-status
   # Compare key counts, budget data, recent audit logs
   ```

---

## Log Management

### 11.1 Viewing Logs

**All Services:**
```bash
make logs
```

**Specific Service:**
```bash
docker compose -f demo/docker-compose.yml logs <service>
# Services: litellm, postgres, otel-collector, caddy
```

**Tail Mode (last N lines):**
```bash
docker compose -f demo/docker-compose.yml logs --tail=100 litellm
```

### 11.2 Log Locations

| Source | Location | Access Method |
|--------|----------|---------------|
| Docker Container Logs | Docker daemon | `docker compose logs` |
| Mounted Application Logs | `demo/logs/` | Direct file access |
| OTEL Telemetry | `demo/logs/otel/telemetry.jsonl` | Direct file access |
| PostgreSQL Logs | Inside container | `docker exec` |

### 11.3 Log Safety

**OAuth Token Warning:**
- When using subscription mode (Claude Code OAuth), tokens may appear in logs
- Never commit logs to version control
- Review logs before sharing
- Use `make secrets-audit` to check tracked repository content for token leakage before sharing

**Secrets Audit:**
```bash
# Run the tracked-file audit before sharing repository content
make secrets-audit

# Note:
# - make tls-verify is a manual-check stub in this open-source release (exit code 2)
# - make secrets-audit applies docs/policy/SECRET_SCAN_POLICY.json to tracked files only
```

**Log Cleanup:**
```bash
# Remove mounted log files
make clean
```

See [demo/LOGGING.md](../demo/LOGGING.md) for detailed log configuration.

---

## TLS/HTTPS Operations

### 12.1 Starting with TLS

**Quick Start:**
```bash
make up-tls
```

**Verification:**
```bash
# Check TLS health
make tls-health

# Manual reminder for OAuth token safety (non-automated in open-source release)
make tls-verify
```

### 12.2 Certificate Management

**Self-Signed Certificates (Development):**
- Automatically generated by Caddy
- Stored in Docker volume
- Shows browser security warnings (expected)

**Let's Encrypt (Production):**
- Configure in `demo/config/caddy/Caddyfile.prod`
- Requires valid domain name
- Automatic certificate renewal

See [docs/deployment/TLS_SETUP.md](deployment/TLS_SETUP.md) for complete TLS documentation.

---

## Troubleshooting Quick Reference

### 13.1 Services Won't Start

**Check Port Conflicts:**
```bash
lsof -i :4000  # LiteLLM port
lsof -i :5432  # PostgreSQL port
```

**Check Logs:**
```bash
make logs
```

**Verify Environment:**
```bash
# Check master key can be read through the typed accessor
./scripts/acpctl.sh env get LITELLM_MASTER_KEY

# Verify .env exists
ls -la demo/.env
```

### 13.2 Database Connection Issues

**Check Container:**
```bash
docker compose -f demo/docker-compose.yml ps postgres
```

**Test Connection:**
```bash
docker exec $(docker compose -f demo/docker-compose.yml ps -q postgres) \
  pg_isready -U litellm
```

**Verify Credentials:**
```bash
# Check database credentials and connection settings without grepping .env
./scripts/acpctl.sh env get POSTGRES_USER
./scripts/acpctl.sh env get POSTGRES_DB
./scripts/acpctl.sh env get DATABASE_URL
```

### 13.3 Authentication Failures

**Verify Master Key:**
```bash
# Read the current master key without executing demo/.env
./scripts/acpctl.sh env get LITELLM_MASTER_KEY
```

**Restart After .env Changes:**
```bash
# Required after editing .env
make down
make up
```

### 13.4 Network Issues (Remote Mode)

**Test Connectivity:**
```bash
# From client machine
make health
```

**Check Firewall:**
```bash
# Ubuntu/Debian
sudo ufw status

# RHEL/CentOS
sudo firewall-cmd --list-all
```

See [docs/DEPLOYMENT.md](DEPLOYMENT.md) section 9 for detailed network troubleshooting.

---

## Related Documentation Index

| Document | Purpose |
|----------|---------|
| [demo/README.md](../demo/README.md) | Quick start, environment setup, basic troubleshooting |
| [docs/DEPLOYMENT.md](DEPLOYMENT.md) | Deployment architecture, network configuration |
| [docs/DATABASE.md](DATABASE.md) | Database schema, deep troubleshooting |
| [docs/security/DETECTION.md](security/DETECTION.md) | Detection rules in detail |
| [docs/security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md) | SIEM integration methods |
| [demo/LOGGING.md](../demo/LOGGING.md) | Log configuration and rotation |
| [docs/deployment/TLS_SETUP.md](deployment/TLS_SETUP.md) | HTTPS/TLS configuration |
| [AGENTS.md](../AGENTS.md) | Repository conventions |

---

## Operational Checklists

### 15.1 Daily Checklist

- [ ] Run `make health` and verify all services pass
- [ ] Check logs for errors: `make logs` (brief review)
- [ ] Review high-severity detection rules: `make validate-detections` or check for alerts

### 15.2 Weekly Checklist

- [ ] Run high-severity detection rules: `make validate-detections`
- [ ] Check budget status: `make db-status` (section 4)
- [ ] Verify backup exists from past week: `ls -la demo/backups/`
- [ ] Review audit log for anomalies

### 15.3 Monthly Checklist

- [ ] Full database backup: `make db-backup`
- [ ] Audit log trend analysis
- [ ] Key rotation assessment
- [ ] Storage usage check
- [ ] Test restore procedure on non-production instance

### 15.4 Incident Response Checklist

- [ ] Identify incident type (outage, security, database, budget)
- [ ] Run appropriate detection rules
- [ ] Check service status: `make health`
- [ ] Review relevant logs
- [ ] Execute recovery procedure from this runbook
- [ ] Document findings and lessons learned

---

## Appendix: Environment Variables

### Required Variables

| Variable | Purpose | Location |
|----------|---------|----------|
| `LITELLM_MASTER_KEY` | Admin key for key generation | `demo/.env` |
| `LITELLM_SALT_KEY` | Salt for key encryption | `demo/.env` |
| `DATABASE_URL` | PostgreSQL connection string | `demo/.env` |
| `POSTGRES_USER` | Database user | `demo/.env` |
| `POSTGRES_PASSWORD` | Database password | `demo/.env` |
| `POSTGRES_DB` | Database name | `demo/.env` |

### Optional Provider Keys

| Variable | Provider |
|----------|----------|
| `OPENAI_API_KEY` | OpenAI |
| `ANTHROPIC_API_KEY` | Anthropic |
| `GEMINI_API_KEY` | Google Gemini |

---

*Last Updated: 2026-03-05*
*Version: 1.0*
