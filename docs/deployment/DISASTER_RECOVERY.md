# Disaster Recovery

The supported disaster-recovery story is the host-first Docker reference implementation.

## Supported Recovery Inputs

- Canonical secrets file: `/etc/ai-control-plane/secrets.env`
- Database backups created through the typed DB workflow under `demo/backups/`
- The tracked automated backup timer contract:
  - `ai-control-plane-backup.timer`
  - `ai-control-plane-backup.service`
  - retention cleanup via `acpctl db backup-retention --apply`
- Release bundle, readiness evidence, and pilot closeout artifacts when applicable

## Supported Automation Contract

The supported host-first path makes backup automation part of the deployment contract:

- `host apply` renders and enables the automated backup timer by default.
- `host install` renders the same backup timer for local systemd-managed hosts.
- Default tracked settings:
  - schedule: `daily`
  - randomized delay: `15m`
  - retention: keep newest `7` backups
- `make host-service-status` reports both the runtime service and backup timer state.

## Supported Recovery Flow

1. Restore host access and the canonical secrets file.
2. Confirm the repository checkout and host deployment assets are present.
3. Restore the database using the typed DB workflow if required.
4. Re-apply the host deployment with `./scripts/acpctl.sh host apply --inventory ...`.
5. Verify runtime readiness with `make health` and `make prod-smoke`.
6. Confirm the automated backup timer is active with `make host-service-status`.

## Backup And Restore Operations

```bash
# Create a fresh manual backup
make db-backup

# Check for stale backups against the tracked retention contract
make db-backup-retention KEEP=7

# Apply retention cleanup
make db-backup-retention APPLY=1 KEEP=7

# Restore the latest backup
make db-restore

# Restore a specific backup artifact
./scripts/acpctl.sh db restore demo/backups/<backup>.sql.gz
```

## Restore Verification Drill

The supported restore-verification proof point is a real scratch restore, not a documentation-only exercise.

```bash
# Create a fresh backup, restore it into a scratch database,
# verify the LiteLLM core schema, and clean up the scratch DB
make dr-drill
```

Successful drill output proves that the current host can:

- create a backup from the embedded PostgreSQL instance
- restore that backup into a scratch database without touching production
- verify the expected LiteLLM core schema
- clean up the scratch database afterward

## Drill Cadence

Recommended minimum cadence for the supported host-first path:

- automated scheduled backup: daily
- retention review: whenever schedule or storage policy changes
- restore-verification drill: at least monthly and before major upgrades

Incubating deployment tracks are intentionally excluded from the supported recovery contract.
