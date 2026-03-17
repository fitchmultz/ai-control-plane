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
- Host overlay runs always execute `make health` and `make prod-smoke`, then run overlay-specific postchecks for `ui` (`make librechat-health`), `tls` (`make tls-health`), and `dlp` (`make dlp-health`).
- `host apply` installs and enables the automated `ai-control-plane-backup.timer` by default. Local `host install` renders the same timer contract and `host service-status` reports both the runtime service and the backup timer.
- Timer defaults come from tracked inventory variables: `acp_backup_timer_on_calendar: daily`, `acp_backup_timer_randomized_delay_sec: 15m`, and `acp_backup_retention_keep: 7`.
- Host firewall posture is host-level ingress hardening only. Customer-owned perimeter controls still own outbound allow-listing, SWG/CASB policy, and broader network enforcement.

## Inventory Guidance

Use `deploy/ansible/inventory/hosts.example.yml` as the starting point.

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
- [ACPCTL Reference](reference/acpctl.md)
