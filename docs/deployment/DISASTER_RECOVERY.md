# Disaster Recovery

The supported disaster-recovery story is the **single-node** host-first Docker reference implementation.

This document covers **recovery after failure**. It does **not** describe automatic failover, clustered high availability, or promotion to a secondary node. Scheduled backups, scratch-restore drills, and typed restore workflows reduce recovery risk, but they do not remove the single-host failure domain.

For topology limits, failure domains, and the validated customer-operated active-passive reference pattern, see [HA And Failover Topology](HA_FAILOVER_TOPOLOGY.md). For manual active-passive failover proof, see [Active-Passive HA Failover Runbook](HA_FAILOVER_RUNBOOK.md).

## Recovery Boundary

- Backup + restore is **disaster recovery**.
- Restarting services or re-running `host apply` on the same host is **recovery inside the same failure domain**.
- Automatic traffic cutover to a secondary host is **not** part of the current supported contract.
- Default backup cadence and retention influence recovery point expectations, but they do not create high availability.

## Supported Recovery Inputs

- Canonical secrets file: `/etc/ai-control-plane/secrets.env`
- Customer-owned off-host copy of a typed ACP database backup, staged onto the recovery host at an absolute path outside `demo/backups/`
- Off-host recovery manifest describing:
  - staged backup file path
  - off-host source URI
  - backup SHA256
  - inventory path
  - secrets env file path
  - expected repo version when pinned
- The tracked automated backup timer contract:
  - `ai-control-plane-backup.timer`
  - `ai-control-plane-backup.service`
  - retention cleanup via `acpctl db backup-retention --apply`
- Release bundle and tracked host deployment assets for the target version
- Certificate renewal rollback artifacts under `demo/logs/cert-renewals/` when controlled renewal has been run
- Readiness evidence and pilot closeout artifacts when applicable

## Supported Automation Contract

The supported host-first path makes backup automation part of the deployment contract:

- `host apply` renders and enables the automated backup timer by default.
- `host apply` also renders and enables the certificate renewal timer whenever the `tls` overlay is enabled.
- `host install` renders the same backup timer for local systemd-managed hosts.
- Default tracked settings:
  - schedule: `daily`
  - randomized delay: `15m`
  - retention: keep newest `7` backups
- `make host-service-status` reports both the runtime service and backup timer state.

## Off-Host Backup-Copy Contract

The repo does **not** automate replication into customer storage. That transport remains customer-owned.

The supported contract is:

1. ACP creates canonical local backups under `demo/backups/`.
2. The operator copies a selected `.sql.gz` backup into customer-owned off-host storage.
3. For recovery proof, the operator stages that off-host copy onto the host being used for the drill or rebuild.
4. The staged file must live at an absolute path **outside** the repo's canonical `demo/backups/` directory.
5. The operator records the drill mode, drill host, staged path, off-host source URI, SHA256 digest, inventory path, and secrets env file path in an off-host recovery manifest.
6. `./scripts/acpctl.sh db off-host-drill --manifest ...` validates the manifest, digest, and scratch restore.

Supported drill modes are:

- `staged-local` — truthful single-machine staged validation only
- `separate-host` — truthful separate-host or separate-VM replacement-host proof

Supported off-host destinations are customer-owned patterns such as `rsync`/`scp` targets, object storage, or mounted network storage. ACP validates the recovered copy after staging; it does not implement the transport.

## Replacement-Host Recovery Workflow

Recommended supported sequence for a replacement host:

1. Provision the replacement host and restore operator access.
2. Restore `/etc/ai-control-plane/secrets.env`.
3. Stage the selected off-host backup copy onto the replacement host at an absolute path outside `demo/backups/`.
4. Check out the matching repo version or release bundle.
5. Prepare or update the inventory so it targets the replacement host that will run the rebuild.
6. Run an initial converge without smoke gating:
   - `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env --skip-smoke-tests`
7. Restore the database from the staged off-host copy:
   - `./scripts/acpctl.sh db restore /var/tmp/ai-control-plane-recovery/<backup>.sql.gz`
8. Re-run host convergence normally:
   - `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env`
9. Verify:
   - `make health COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`
   - `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`
   - `make host-service-status`

## Separate-Host Or Separate-VM Proof

Use this path when you need truthful evidence that a customer-owned off-host copy can rebuild ACP on a replacement host or replacement VM.

1. Provision the replacement host or replacement VM.
2. Restore `/etc/ai-control-plane/secrets.env` on that host.
3. Check out the matching repo version or unpack the matching release bundle on that host.
4. Stage the selected customer-owned off-host backup copy onto that host at an absolute path outside `demo/backups/`.
5. Prepare an inventory that targets the replacement host.
6. Copy `demo/config/off_host_recovery.separate_host.yaml` to a host-local manifest path and update:
   - `drill_mode: separate-host`
   - `drill_host: <replacement-hostname>`
   - `backup_file`
   - truthful `backup_source_uri`
   - `backup_sha256`
   - `inventory_path`
   - `secrets_env_file`
7. Run `./scripts/acpctl.sh db off-host-drill --manifest <path>` on the replacement host or replacement VM.
8. Capture the generated evidence bundle and handoff notes with the drill labeled as `separate-host`.

When no second host or real customer storage is available, operators may still run a truthful `staged-local` drill by copying a fresh ACP backup to an absolute path outside `demo/backups/` and using a truthful `file://...` source URI in the manifest. ACP validates the recovered copy after staging, but a staged-local drill does not prove customer transport or separate-host replacement.

## Off-Host Recovery Evidence

Prepare the manifest for the drill mode you are running:

```bash
mkdir -p demo/logs/recovery-inputs
cp demo/config/off_host_recovery.example.yaml demo/logs/recovery-inputs/off_host_recovery.yaml
# or start from demo/config/off_host_recovery.separate_host.yaml on the replacement host
sha256sum /var/tmp/ai-control-plane-recovery/<backup>.sql.gz
# update the manifest with the real digest, drill_mode, drill_host, and paths
```

Run the non-destructive proof drill:

```bash
make db-off-host-drill OFF_HOST_RECOVERY_MANIFEST=demo/logs/recovery-inputs/off_host_recovery.yaml
```

A successful run writes a timestamped evidence bundle under:

- `demo/logs/evidence/replacement-host-recovery/`

That evidence proves the staged backup copy came from outside the canonical same-host backup directory and can restore into a scratch database with the expected core schema. The generated JSON and Markdown summaries record `drill_mode`, `drill_host`, and an explicit claim-boundary statement so staged-local evidence and separate-host evidence cannot be conflated. When the manifest uses a truthful local `file://...` provenance and the drill runs on the same machine that created the backup, treat the result as **single-machine staged off-host validation only**. It does **not** prove real customer transport or real replacement-host recovery on separate hardware.

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

## Upgrade Rollback

Version rollback is part of the supported host-first recovery story.

Use the typed upgrade framework:

1. Keep the previous release checkout until the upgrade is accepted.
2. Use the generated upgrade run directory under `demo/logs/upgrades/`.
3. Run `./scripts/acpctl.sh upgrade rollback --run-dir ... --inventory ... --env-file /etc/ai-control-plane/secrets.env` from the previous release checkout.
4. The rollback flow restores the pre-upgrade config snapshot, restores the pre-upgrade embedded database backup, and re-runs host convergence for the previous release.

See [UPGRADE_MIGRATION.md](UPGRADE_MIGRATION.md) for the full operator workflow.

## Drill Cadence

Recommended minimum cadence for the supported host-first path:

- automated scheduled backup: daily
- retention review: whenever schedule or storage policy changes
- restore-verification drill: at least monthly and before major upgrades

Incubating deployment tracks are intentionally excluded from the supported recovery contract.
