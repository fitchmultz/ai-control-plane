# Upgrade And Migration

The supported upgrade story is host-first and explicit.

## Supported Contract

- Upgrades are supported only when an explicit release-to-release edge exists in the typed upgrade catalog.
- No backward-compatibility shims are provided.
- No direct skip-version cutovers are supported unless every intermediate edge exists and is executed in order.
- `upgrade execute` and `upgrade rollback` are supported only for the embedded database mode in the validated host-first path.
- Upgrade execution runs from the **target release checkout** on the managed host.
- Rollback runs from the **previous release checkout** recorded by the upgrade run metadata.

## Current Release Status

The current framework release (`0.1.0`) establishes the typed upgrade contract, rollback artifact format, and operator workflow.

Supported in-place edges today:

- none

That means:

- pre-framework releases -> `0.1.0`: **fresh install + restore only**
- future releases must add explicit typed upgrade edges before they claim in-place support

## Operator Workflow

1. Check out the **target release** on the managed host.
2. Confirm the release notes and compatibility matrix for the requested path.
3. Run:
   - `make upgrade-plan FROM_VERSION=X.Y.Z INVENTORY=... SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
   - `make upgrade-check FROM_VERSION=X.Y.Z INVENTORY=... SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
4. Execute:
   - `make upgrade-execute FROM_VERSION=X.Y.Z INVENTORY=... SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
5. Review the generated upgrade run under `demo/logs/upgrades/upgrade-<timestamp>/`.
6. Keep the previous release checkout until post-upgrade validation is complete.

## What `upgrade check` Validates

- explicit release-edge support for `FROM_VERSION -> target VERSION`
- checkout/version alignment with the target release
- config migrations applied to a temporary env snapshot
- production config contract validation against the migrated snapshot
- embedded database accessibility
- host-first Ansible convergence in check mode with upgrade metadata

## What `upgrade execute` Does

- snapshots the canonical env/secrets file
- creates a compressed embedded PostgreSQL backup
- persists rollback metadata under `demo/logs/upgrades/`
- applies explicit config migrations
- applies explicit database migrations
- runs `host apply` in upgrade mode
- leaves a rollback-ready run directory even if a later step fails

## Upgrade Run Artifacts

Each successful or partially completed execute attempt writes a private run directory under `demo/logs/upgrades/` containing:

- `config.before.env`
- `database.before.sql.gz`
- `summary.json`
- `inventory.txt`

The latest run pointer is stored at:

- `demo/logs/upgrades/latest-upgrade-run.txt`

## Rollback Workflow

1. Keep the upgrade run directory created during `upgrade execute`.
2. Check out the **previous release** on the managed host.
3. Run:
   - `make upgrade-rollback UPGRADE_RUN_DIR=demo/logs/upgrades/upgrade-<timestamp> INVENTORY=... SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`
4. The rollback workflow restores:
   - the pre-upgrade config snapshot
   - the pre-upgrade embedded database snapshot
   - previous-release host convergence via `host apply`

Rollback is blocked when the current checkout does not match the recorded previous release.

## Database And Config Migration Rules

- Config migrations may mutate only the canonical production env/secrets file.
- Database migrations may run only through the typed upgrade catalog and typed DB migration service.
- Untracked shell or Ansible migration logic is unsupported.
- If a path is not declared explicitly, the operator must use fresh install + restore instead of an in-place upgrade.

## Validation Commands

- `make upgrade-plan ...`
- `make upgrade-check ...`
- `make health`
- `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`
- `make dr-drill`

## Release Discipline

Every release that claims in-place upgrade support must update:

- `internal/upgrade` release-edge catalog
- [`UPGRADE_COMPATIBILITY_MATRIX.md`](UPGRADE_COMPATIBILITY_MATRIX.md)
- [`RELEASE_NOTES.md`](../../RELEASE_NOTES.md)
- [`CHANGELOG.md`](../../CHANGELOG.md)
