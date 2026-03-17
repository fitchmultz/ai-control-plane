# Single-Tenant Production Contract

The production contract is the host-first Docker deployment path described in [DEPLOYMENT.md](../DEPLOYMENT.md).

## Canonical Contract

1. Validate production config against `/etc/ai-control-plane/secrets.env`.
2. Run `./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env`.
3. Run `./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml`.
4. Run `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml`.
5. Run `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`.
6. Run `./scripts/acpctl.sh cert check --threshold-days 30`.
7. Confirm `systemctl status ai-control-plane-cert-renewal.timer` is healthy when the `tls` overlay is enabled.
8. Confirm `make host-service-status` shows the runtime service and automated backup timer, plus the certificate renewal timer where applicable.

## Invariants

- `/etc/ai-control-plane/secrets.env` is the canonical production env source.
- `demo/.env` is local-demo only.
- Production workflows do not sync secrets back into the repository tree.
- The supported runtime remains the host-first Docker baseline, with overlays only when explicitly selected.
- Without the `tls` overlay, the supported `acp_public_url` remains loopback-only.
- Remote non-loopback ingress requires the `tls` overlay and an `https://...` public URL.
- When the `tls` overlay is enabled, Caddy owns certificate issuance and storage, and the supported host-first path installs the certificate renewal timer.
- The supported host boundary is Debian 12+ or Ubuntu 24.04+ with systemd, apt, Docker, and Docker Compose.
- The tracked host playbook enforces baseline package/update posture, SSH hardening, private secrets-file permissions, explicit UFW defaults, and automated backup-timer installation.
- The supported recovery contract includes scheduled backups, tracked retention, a repeatable scratch-restore verification drill, and typed rollback artifacts for future explicit upgrade edges.
- In-place upgrades are supported only through the typed `acpctl upgrade` workflow when an explicit release edge exists.
