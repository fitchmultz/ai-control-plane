# Operator Signal Reference

This document is the canonical operator-facing map of what ACP exposes today for runtime visibility on the supported host-first Docker surface.

## Support boundary

ACP ships **local operator views and evidence artifacts**. It does **not** claim a built-in Prometheus, Grafana, Loki, Tempo, or Jaeger stack as part of the supported topology.

Supported dashboard and reporting surfaces today:

1. **Live terminal status** — `./scripts/acpctl.sh status --wide --watch`
2. **Canonical report snapshot** — `make operator-report`
3. **Static HTML dashboard snapshot** — `make operator-dashboard`
4. **LiteLLM native admin UI** — `${GATEWAY_URL}/ui` for keys and budgets
5. **Customer-owned SIEM dashboards** — built from the normalized evidence and query packs in `docs/security/SIEM_INTEGRATION.md`

## Signal inventory

| Signal domain | ACP source of truth | Primary operator surface | Notes |
| --- | --- | --- | --- |
| Gateway reachability, auth, model-list access | `internal/gateway` via `gateway` status collector | `status`, `operator-report`, `operator-dashboard` | Fast truth for ingress and master-key posture |
| Database reachability and schema readiness | `internal/db` runtime summary via `database` collector | `status`, `operator-report`, `operator-dashboard` | Includes DB mode, schema count, size, and connection count |
| Request volume, token volume, spend, non-success rate (24h) | `LiteLLM_SpendLogs` via `traffic` collector | `status`, `operator-report`, `operator-dashboard` | Routed gateway traffic only |
| Enforcement and anomaly findings | `LiteLLM_SpendLogs` + detection thresholds via `detections` collector | `status`, `operator-report`, `operator-dashboard`, SIEM | Use SIEM for longer retention and alert routing |
| Key inventory | `LiteLLM_VerificationToken` via `keys` collector | `status`, `operator-report`, `operator-dashboard`, LiteLLM UI | Counts total, active, expired |
| Budget posture | `LiteLLM_BudgetTable` via `budget` collector | `status`, `operator-report`, `operator-dashboard`, LiteLLM UI | Counts total, high-utilization, exhausted |
| Certificate lifecycle | `internal/certlifecycle` via `certificate` collector | `status`, `operator-report`, `operator-dashboard` | Only meaningful when TLS overlay is active |
| Local backup freshness | `demo/backups/*.sql.gz` via `backup` collector | `status`, `operator-report`, `operator-dashboard` | Reports last backup path, size, timestamp, age |
| Readiness evidence freshness | `demo/logs/evidence/latest-run.txt` + `summary.json` via `readiness` collector | `status`, `operator-report`, `operator-dashboard` | Reports latest run ID, overall result, age, and failing/skipped gates |
| Direct/bypass traces | `demo/logs/otel/telemetry.jsonl` via OTEL collector | `make otel-health`, OTEL trace workflow in `OTEL_SETUP.md` | Only for direct-to-vendor or correlation telemetry |
| Historical cost allocation | chargeback workflow | `make chargeback-report` | Month-scoped showback/chargeback, not a live dashboard |

## Recommended operator workflow

### 1. Current runtime snapshot

```bash
make operator-report WIDE=1
```

Use this when you need the current typed truth in markdown or JSON.

### 2. Static dashboard snapshot

```bash
make operator-dashboard
open demo/logs/observability/operator-dashboard.html
```

Use this for handoff, screenshots, ops review, or a quick browser-based dashboard without introducing more infrastructure.

### 3. Live watch mode

```bash
./scripts/acpctl.sh status --wide --watch
```

Use this during rollout, key rotation, incident response, restore drills, or failover exercises.

### 4. OTEL trace inspection for bypass telemetry

See [OTEL_SETUP.md](OTEL_SETUP.md) for the direct-subscription trace workflow.

## Signal interpretation

### Traffic

- `traffic` is derived from routed gateway request logs in the last 24 hours.
- A warning appears when non-success traffic exceeds the documented DR-003 threshold: **>10% over at least 10 requests**.
- No traffic is not automatically unhealthy.

### Backup

- `backup` is healthy when the newest local backup is fresh.
- It warns when the newest backup is aging and becomes unhealthy when it is materially stale.
- This is a **local recovery-point signal**, not proof of off-host retention by itself.

### Readiness

- `readiness` reflects the latest generated readiness-evidence pack, not a background daemon.
- A passing but old run is shown as stale because readiness evidence is only trustworthy when current.
- A failed run blocks truthful external reuse of that evidence pack.

### Enforcement

- `detections` is the fast operator summary.
- `docs/security/SIEM_INTEGRATION.md` plus `demo/config/siem_queries.yaml` remain the source of truth for downstream dashboarding and alert rules.

## What ACP does not claim here

- A built-in long-retention metrics stack
- A shipped Grafana/Prometheus/Loki/Tempo deployment surface
- OTEL traces for every gateway-routed request path
- Backup signals as proof of customer off-host copy or multi-host HA

Use the ACP built-in views for the supported host-first runtime, then layer customer-owned SIEM or observability systems on top when longer retention, alerting, or enterprise dashboard ownership is required.
