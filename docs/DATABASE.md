# AI Control Plane - Database Documentation

This document provides comprehensive information about the PostgreSQL database used by the AI Control Plane LiteLLM gateway, including initialization, schema, persistence, backup, and restore procedures.

## Table of Contents

1. [Database Overview](#1-database-overview)
2. [Automatic Schema Initialization](#2-automatic-schema-initialization)
3. [Database Schema](#3-database-schema)
4. [Volume Persistence](#4-volume-persistence)
5. [Database Health Verification](#5-database-health-verification)
6. [Backup and Restore](#6-backup-and-restore)
7. [Database Maintenance](#7-database-maintenance)

---

## 1. Database Overview

The AI Control Plane uses PostgreSQL 18 as its persistent storage layer for the LiteLLM gateway.

### Connection Details

| Property | Value |
|----------|-------|
| **Database Engine** | PostgreSQL 18 |
| **Host** | `postgres` (Docker network) or `127.0.0.1:5432` (host, only if published/tunneled) |
| **Database Name** | `${POSTGRES_DB}` (default: `litellm`) |
| **User** | `${POSTGRES_USER}` (default: `litellm`) |
| **Password** | `${POSTGRES_PASSWORD}` (default: `litellm` - change for production) |
| **Connection URL** | `postgresql://litellm:litellm@postgres:5432/litellm` |

**Note:** Database credentials are configured via environment variables in `demo/.env`:
- `POSTGRES_USER` (default: `litellm`) - Database user
- `POSTGRES_PASSWORD` (default: `litellm`) - Database password (change for production)
- `POSTGRES_DB` (default: `litellm`) - Database name
- `DATABASE_URL` - Connection string that must match the values above

See `demo/.env.example` for the credential template. The defaults are suitable for demo
environments but should be changed for production deployments.

### Purpose

The database stores:
- **Virtual Keys**: Encrypted API keys for client authentication
- **User Information**: User identities and roles
- **Budget Tracking**: Per-key and per-proxy budget limits and usage
- **Audit Logs**: Request/response logging for compliance and debugging

---

## 2. Automatic Schema Initialization

LiteLLM automatically creates and initializes the database schema on first startup when `database_url` is configured.

### How It Works

1. **First Startup**: When LiteLLM starts with a valid `DATABASE_URL`, it checks for required tables
2. **Table Creation**: If tables don't exist, LiteLLM creates them automatically
3. **Schema Validation**: On subsequent startups, LiteLLM validates the schema exists

### No Manual Migrations Required

- **Initial setup**: No manual SQL scripts or migration tools required
- **Version management**: LiteLLM handles schema versioning internally
- **Idempotent**: Safe to restart LiteLLM multiple times

### Verification

After starting services, verify schema initialization:

```bash
# Using the database status script
make db-status

# Or manually
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c "\dt"
```

Expected output:
```
          List of relations
 Schema |         Name          | Type  |  Owner
--------+-----------------------+-------+----------
 public | LiteLLM_BudgetTable          | table | litellm
 public | LiteLLM_ProxyModelTable      | table | litellm
 public | LiteLLM_SpendLogs            | table | litellm
 public | LiteLLM_UserTable            | table | litellm
 public | LiteLLM_VerificationToken    | table | litellm
```

**Important:** The exact schema is an implementation detail of the LiteLLM container image
used by this repo. Make-driven workflows default to the locally built
`ai-control-plane/litellm-hardened:local` image from
`demo/images/litellm-hardened/Dockerfile`, while direct compose usage can still
override to a pinned registry image via `LITELLM_IMAGE`.
If you change the pinned image, treat it as a schema upgrade: take a backup (`make db-backup`),
restart services, and re-verify tables via `make db-status`.

`make db-status` now reports these sections directly from the typed CLI:
1. Runtime Summary
2. Schema Verification
3. Virtual Keys
4. Budget Usage
5. Detection Summary

---

## 3. Database Schema

LiteLLM creates and manages its database schema automatically. In this repo, the
scripts and detections assume the following core tables exist:

### `"LiteLLM_VerificationToken"` (Virtual keys)

Stores virtual key metadata such as `key_alias`, budgets, and expiration.

**How to create a key (recommended):**

```bash
make key-gen ALIAS=my-key BUDGET=10.00
```

### `"LiteLLM_BudgetTable"` (Budgets)

Stores budget records referenced by virtual keys.

### `"LiteLLM_SpendLogs"` (Gateway “audit log” / usage logs)

Stores request-level usage and cost metadata for gateway traffic. This repo’s
export and detection scripts intentionally treat this as the “audit log” source.

**Security note:** Prompt/response bodies are not part of the evidence pipeline
in this repo; exports focus on metadata (principal, model, tokens, spend, status).

### `"LiteLLM_UserTable"` (Users)

Stores user identities/metadata when used by LiteLLM.

### `"LiteLLM_ProxyModelTable"` (Proxy/global budgets)

Stores proxy-level controls/budgets (if configured).

### Approval Queue Storage (External to LiteLLM Schema)

The approval workflow system stores request data **outside** the LiteLLM-managed database schema, in JSON Lines format files. This separation ensures:

- **LiteLLM schema integrity**: No custom tables in LiteLLM-managed database
- **Portability**: Queue data can be moved/archived independently
- **Audit compliance**: Immutable audit log format (append-only)

**Storage Locations:**

| File | Purpose | Format | Permissions |
|------|---------|--------|-------------|
| `demo/logs/approval_queue.jsonl` | Active approval queue (pending/approved/rejected/provisioned) | JSON Lines | 600 (owner rw) |
| `demo/logs/approval_audit.log` | Immutable audit trail of all approval actions | JSON Lines | 600 (owner rw) |

**JSON Structure (Request):**
```json
{
  "request_id": "req-20260207-abc123def",
  "status": "pending",
  "submitted_at": "2026-02-07T10:30:00Z",
  "submitted_by": "alice@example.com",
  "approved_at": null,
  "approved_by": null,
  "sla_deadline": "2026-02-07T14:30:00Z",
  "escalated": false,
  "request": {
    "alias": "prod-api-key",
    "budget": "50.00",
    "rpm": "100",
    "tpm": "100000",
    "parallel": "20",
    "duration": "30d",
    "justification": "Production service API access"
  },
  "provisioned_key": null,
  "audit_log": []
}
```

**JSON Structure (Audit Entry):**
```json
{
  "timestamp": "2026-02-07T12:00:00Z",
  "request_id": "req-20260207-abc123def",
  "action": "approve",
  "actor": "bob@example.com",
  "details": {}
}
```

**Querying Approval Data (File-Based):**

```bash
# List pending requests from queue file
jq 'select(.status == "pending")' demo/logs/approval_queue.jsonl

# Get one request by ID
jq 'select(.request_id == "req-20260207-abc123def")' demo/logs/approval_queue.jsonl

# Search audit log for specific actor
jq 'select(.actor == "bob@example.com")' demo/logs/approval_audit.log
```

> This public snapshot does not expose a dedicated approval-queue CLI command surface; inspect queue/audit files directly when these artifacts are present.

**Backup Considerations:**

The approval queue files are stored in `demo/logs/` which is gitignored. Include them in your backup strategy:

```bash
# Backup queue and audit log
cp demo/logs/approval_queue.jsonl demo/backups/approval-queue-$(date +%Y%m%d).jsonl
cp demo/logs/approval_audit.log demo/backups/approval-audit-$(date +%Y%m%d).log
```

**Retention Policy:**

- **Approval Queue**: Clean up provisioned/rejected/cancelled requests older than 90 days
- **Audit Log**: Retain indefinitely for compliance (append-only, never delete)

For the authoritative list and row counts, use:

```bash
make db-status
```

---

## 4. Volume Persistence

The PostgreSQL data is persisted in a Docker named volume.

### Volume Configuration

From `demo/docker-compose.yml`:

```yaml
volumes:
  pgdata:
    driver: local
```

```yaml
services:
  postgres:
    volumes:
      - pgdata:/var/lib/postgresql/data
```

### Volume Location

On the host system, the volume is stored in Docker's volume directory:

```bash
# List Docker volumes
docker volume ls | grep pgdata

# Inspect volume to find actual location
docker volume inspect ai-control-plane_pgdata
```

Typical location: `/var/lib/docker/volumes/ai-control-plane_pgdata/_data`

### Volume Lifecycle

| Operation | Command | Effect on Data |
|-----------|---------|----------------|
| **Stop services** | `make down` | Preserved |
| **Restart services** | `make up` | Preserved |
| **Recreate containers** | `make down && make up` | Preserved |
| **Clean all artifacts** | `make clean` | **DELETED** |

### Backup Before Cleaning

Always back up before running `make clean`. In interactive shells, `make clean` prompts for confirmation; for scripts/automation, use `make clean-force`:

```bash
# Backup first
make db-backup

# Then clean (prompts for confirmation in interactive shells)
make clean

# For scripts/automation (skips confirmation)
make clean-force
```

---

## 5. Database Health Verification

### Using the Database Status Script

The easiest way to check database health:

```bash
make db-status
```

This displays:
- Database size
- Table row counts
- Recent virtual keys
- Budget usage summary
- Recent audit log entries

### Manual Verification Queries

Connect to the database:

```bash
docker exec -it $(docker compose ps -q postgres) \
  psql -U litellm -d litellm
```

#### Check Database Version

```sql
SELECT version();
```

#### List All Tables

```sql
\dt
```

#### Check Row Counts

```sql
SELECT 'LiteLLM_VerificationToken' AS table_name, COUNT(*) FROM "LiteLLM_VerificationToken"
UNION ALL
SELECT 'LiteLLM_UserTable', COUNT(*) FROM "LiteLLM_UserTable"
UNION ALL
SELECT 'LiteLLM_BudgetTable', COUNT(*) FROM "LiteLLM_BudgetTable"
UNION ALL
SELECT 'LiteLLM_SpendLogs', COUNT(*) FROM "LiteLLM_SpendLogs"
UNION ALL
SELECT 'LiteLLM_ProxyModelTable', COUNT(*) FROM "LiteLLM_ProxyModelTable";
```

#### Check Database Size

```sql
SELECT pg_size_pretty(pg_database_size('litellm')) AS database_size;
```

#### View Recent Virtual Keys

```sql
SELECT key_alias, max_budget, user_id, created_at
FROM "LiteLLM_VerificationToken"
ORDER BY created_at DESC
LIMIT 10;
```

#### Check Budget Usage

```sql
SELECT
  v.key_alias,
  ROUND(v.spend::numeric, 4) AS spent,
  ROUND(b.max_budget::numeric, 4) AS maximum,
  ROUND((v.spend / NULLIF(b.max_budget, 0) * 100)::numeric, 2) AS percent_used
FROM "LiteLLM_VerificationToken" v
JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id
ORDER BY v.created_at DESC;
```

#### Open a Database Shell

```bash
make db-shell
```

- Embedded mode connects to the repo-local Compose `postgres` container.
- External mode requires `DATABASE_URL` and a local `psql` binary.

---

## 6. Backup and Restore

### Backup Workflow

Create timestamped backups of the embedded PostgreSQL database through the typed operator surface.

#### Usage

```bash
# Create a timestamped backup
make db-backup

# Create a custom-named backup
./scripts/acpctl.sh db backup my-custom-backup
```

#### Backup Location

Backups are stored in `demo/backups/` by default:

```
demo/backups/
  litellm-backup-20260317-113000.sql.gz
  litellm-backup-20260316-020000.sql.gz
  my-custom-backup.sql.gz
```

The backup directory stays local-only and private. Backup files are written with `0600` permissions and the containing directory with `0700` permissions.

#### Environment Variable

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_DIR` | `demo/backups/` | Override the canonical backup directory |

#### Backup Format

Backups are compressed SQL dumps created with `pg_dump -c -C --no-owner --no-acl`:

```bash
# Validate gzip integrity
gzip -t demo/backups/litellm-backup-20260317-113000.sql.gz

# View backup contents (decompress to stdout)
gunzip -c demo/backups/litellm-backup-20260317-113000.sql.gz | less
```

### Backup Retention Workflow

The supported retention contract is now typed and deterministic.

```bash
# Check for stale backups (default keep=7)
make db-backup-retention

# Check with a custom keep count
make db-backup-retention KEEP=14

# Apply cleanup
make db-backup-retention APPLY=1 KEEP=14
```

Behavior:

1. Collect canonical `.sql.gz` backup artifacts from the configured backup directory.
2. Sort them newest-first by modification time.
3. Keep the newest `N` artifacts.
4. Either fail in check mode or delete stale artifacts in apply mode.

### Restore Workflow

Restore from the latest backup or a specific backup file.

#### Usage

```bash
# Restore latest backup in demo/backups/
make db-restore

# Restore specific backup (typed CLI path)
./scripts/acpctl.sh db restore demo/backups/litellm-backup-20260317-113000.sql.gz
```

#### What Happens During Restore

1. Resolve the backup file (latest if none is provided).
2. Validate that the archive exists and is readable.
3. Stream the decompressed SQL into `psql` against the embedded PostgreSQL instance.
4. Report success/failure using the standard exit-code contract.

#### Safety Warnings

- **Data loss**: Restore overwrites existing embedded database data.
- **Downtime**: Gateway may experience errors during restore.
- **Backup first**: Always run `make db-backup` before restore.
- **Integrity check**: Validate the archive before restore (`gzip -t demo/backups/<backup>.sql.gz`).
- **Mode restriction**: Backup and restore are unsupported for external database mode.

### Restore Verification Drill

The supported DR drill is now a real restore-verification workflow, not a stub.

```bash
# Create a fresh backup, restore it into a scratch DB,
# verify the LiteLLM core schema, and clean up the scratch DB
make dr-drill
```

What the drill proves:

1. The current embedded PostgreSQL instance can be backed up.
2. The resulting backup can be rewritten safely for a scratch database.
3. The scratch database restore succeeds.
4. The expected LiteLLM core schema is present after restore.
5. The scratch database is dropped after verification.

### Host-First Backup Automation

On the supported host-first path, backup automation is part of the deployment contract:

- `host apply` installs and enables `ai-control-plane-backup.timer` by default.
- `host install` renders the same timer contract for local systemd-managed hosts.
- Default tracked settings:
  - schedule: `daily`
  - randomized delay: `15m`
  - retention keep count: `7`
- `make host-service-status` reports both the runtime service and the backup timer.

---

## 7. Database Maintenance

### Regular Health Checks

Run health checks regularly to ensure database integrity:

```bash
# Full service health check
make health

# Database-specific status
make db-status
```

### Log Rotation

PostgreSQL logs can grow over time. Monitor log file sizes:

```bash
# Check PostgreSQL log size
docker exec $(docker compose ps -q postgres) \
  du -sh /var/lib/postgresql/data/log/

# Rotate logs if needed
docker exec $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c "SELECT pg_logfile_rotate();"
```

### Database Vacuuming

PostgreSQL automatically handles vacuuming, but you can run manual maintenance:

```bash
# Analyze tables for query optimization
docker exec $(docker compose ps -q postgres) \
  psql -U litellm -d litellm -c "VACUUM ANALYZE;"
```

### Audit Log Cleanup

Audit logs can grow large over time. Implement a retention policy:

```sql
-- Delete audit logs older than 90 days
DELETE FROM "LiteLLM_SpendLogs"
WHERE "startTime" < NOW() - INTERVAL '90 days';

-- Confirm deletion
SELECT COUNT(*) FROM "LiteLLM_SpendLogs";
```

### Performance Monitoring

Monitor database performance:

```sql
-- Check active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'litellm';

-- Check table sizes
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## Additional Resources

- **LiteLLM Database Documentation**: <https://docs.litellm.ai/>
- **PostgreSQL Documentation**: <https://www.postgresql.org/docs/16/>
- **Deployment Guide**: See [DEPLOYMENT.md](DEPLOYMENT.md) for full deployment instructions
- **Demo Environment**: See [demo/README.md](../demo/README.md) for demo-specific information
