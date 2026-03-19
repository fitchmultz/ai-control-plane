# Operations And Deployment

This document describes the supported host-first deployment and operations path.

## Runtime Profiles

- Base runtime: `make up`
- Base plus DLP overlay: `make up-dlp`
- Base plus managed UI overlay: `make up-ui`
- Base plus both overlays: `make up-full`
- Offline deterministic runtime: `make up-offline`
- TLS overlay: `make up-tls`
- Production OTEL + TLS profile: `make up-production`

## Local Workflow

```bash
make install
make up
make health
make prod-smoke
```

Use `demo/.env` for local runs. It is local-only and not part of the host-production secret contract.

If you use the managed UI overlay, populate the LibreChat-specific keys documented in [`demo/.env.example`](../demo/.env.example) before running `make up-ui` or `make up-full`.

## Supported Host Boundary

The supported host-first production path now assumes:

- Debian 12+ or Ubuntu 24.04+
- systemd
- apt-based package management
- Docker + Docker Compose already installed
- canonical secrets at `/etc/ai-control-plane/secrets.env`
- Ansible SSH host-key verification enabled

The host playbook hardens the host baseline by enforcing:

- safe apt upgrades during convergence
- installed baseline packages (`ufw`, `unattended-upgrades`, and tracked operator utilities)
- automatic security updates
- private secrets-file permissions
- SSH hardening (no password auth, no root login)
- UFW defaults: `deny incoming`, `allow outgoing`, `deny routed`
- Docker-compatible sysctl hardening
- automated database backup timer installation with tracked retention defaults

## Host-First Production Workflow

Canonical order:

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
make host-service-status
```

Rules:

- `/etc/ai-control-plane/secrets.env` is the canonical production env source.
- Production workflows do not sync secrets into `demo/.env`.
- Compose-driven host operations use `COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`.
- The base runtime remains LiteLLM plus PostgreSQL unless an overlay is explicitly selected.
- Supported host overlays are selected through `acp_runtime_overlays` in the Ansible inventory. Allowed values are `tls`, `ui`, `dlp`, and `offline`.
- Base host deployment remains the default, but without `tls` the supported `acp_public_url` stays loopback-only (`http://127.0.0.1:4000`).
- Remote non-loopback ingress requires the `tls` overlay and an `https://...` `acp_public_url`.
- Host overlay runs always execute `make health` and `make prod-smoke`, then run overlay-specific postchecks for `ui` (`make librechat-health`), `tls` (`make tls-health` plus `./scripts/acpctl.sh cert check`), and `dlp` (`make dlp-health`).
- `host apply` installs and enables the automated `ai-control-plane-backup.timer` by default and installs `ai-control-plane-cert-renewal.timer` whenever the `tls` overlay is enabled. Local `host install` still renders the runtime service and backup timer contract, and can optionally install the same certificate timer contract.
- Timer defaults come from tracked inventory variables: `acp_backup_timer_on_calendar: daily`, `acp_backup_timer_randomized_delay_sec: 15m`, `acp_backup_retention_keep: 7`, `acp_cert_renewal_timer_on_calendar: daily`, `acp_cert_renewal_timer_randomized_delay_sec: 30m`, and `acp_cert_renewal_threshold_days: 30`.
- Host firewall posture is host-level ingress hardening only. Customer-owned perimeter controls still own outbound allow-listing, SWG/CASB policy, and broader network enforcement.

## Topology Limits And HA Expectations

The supported production topology today is a **single-node** host-first deployment. The tracked Ansible playbook, `deploy/ansible/playbooks/gateway_host.yml`, converges one gateway host running LiteLLM, PostgreSQL, and any selected overlays on the same machine.

Truthful availability boundary:

- The current contract supports **recovery**, not automatic **failover**.
- Scheduled backups, restore drills, and typed re-apply workflows reduce recovery risk, but they do not create host-level HA.
- A host failure, local storage failure, or database failure can still take down the entire deployment because those components share one failure domain.
- Customer-owned DNS, load balancers, and network controls determine any external traffic failover behavior.

See [deployment/HA_FAILOVER_TOPOLOGY.md](deployment/HA_FAILOVER_TOPOLOGY.md) for the full failure-domain model, RPO/RTO truth, and the next credible active-passive pattern. See [deployment/HA_FAILOVER_RUNBOOK.md](deployment/HA_FAILOVER_RUNBOOK.md) for the customer-operated failover drill workflow and evidence contract. See [deployment/DISASTER_RECOVERY.md](deployment/DISASTER_RECOVERY.md) for the supported restore workflow after failure.

## Certificate Lifecycle Workflow

For TLS-enabled host-first deployments, use the typed certificate lifecycle surface:

```bash
make cert-status DOMAIN=gateway.example.com
make cert-renew DOMAIN=gateway.example.com THRESHOLD_DAYS=30
sudo make cert-renew-install SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

The supported path assumes Caddy owns certificate issuance and storage. ACP validates live certificate state through `acpctl cert check` and preserves rollback artifacts for controlled renewals under `demo/logs/cert-renewals/`.

See [deployment/CERTIFICATE_LIFECYCLE.md](deployment/CERTIFICATE_LIFECYCLE.md) for the full runbook.

## Host-First Upgrade Workflow

When a release explicitly ships a supported in-place edge, run the upgrade from the managed host checkout for the **target release**:

```bash
make upgrade-plan FROM_VERSION=X.Y.Z INVENTORY=deploy/ansible/inventory/hosts.yml SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make upgrade-check FROM_VERSION=X.Y.Z INVENTORY=deploy/ansible/inventory/hosts.yml SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make upgrade-execute FROM_VERSION=X.Y.Z INVENTORY=deploy/ansible/inventory/hosts.yml SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

If the release does not declare an explicit upgrade edge, do **not** perform an in-place cutover. Use fresh install + restore instead.

Rollback runs from the **previous release checkout** using the saved upgrade run directory:

```bash
make upgrade-rollback UPGRADE_RUN_DIR=demo/logs/upgrades/upgrade-<timestamp> INVENTORY=deploy/ansible/inventory/hosts.yml SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

See [deployment/UPGRADE_MIGRATION.md](deployment/UPGRADE_MIGRATION.md) for the full contract and [deployment/UPGRADE_COMPATIBILITY_MATRIX.md](deployment/UPGRADE_COMPATIBILITY_MATRIX.md) for the current supported-path matrix.

## Inventory Guidance

Use `deploy/ansible/inventory/hosts.example.yml` as the starting point for the single-host supported baseline.
Use `deploy/ansible/inventory/hosts.ha.example.yml` when preparing a customer-operated active-passive failover drill.

Recommended patterns:

- Base host-only contract:
  - `acp_runtime_overlays: []`
  - `acp_public_url: http://127.0.0.1:4000`
- Remote ingress contract:
  - `acp_runtime_overlays: [tls]`
  - `acp_public_url: https://gateway.example.com`
  - tighten `acp_host_firewall_tls_allowed_cidrs` to the intended client ranges whenever possible

## Typed Operational Checks

```bash
./scripts/acpctl.sh status
./scripts/acpctl.sh health
./scripts/acpctl.sh smoke
./scripts/acpctl.sh doctor
```

Use `make ci-pr` for fast deterministic checks and `make ci` for the full supported gate.

## References

- [Support](SUPPORT.md)
- [Architecture](technical-architecture.md)
- [Security And Governance](SECURITY_GOVERNANCE.md)
- [HA And Failover Topology](deployment/HA_FAILOVER_TOPOLOGY.md)
- [Active-Passive HA Failover Runbook](deployment/HA_FAILOVER_RUNBOOK.md)
- [Disaster Recovery](deployment/DISASTER_RECOVERY.md)
- [ACPCTL Reference](reference/acpctl.md)
