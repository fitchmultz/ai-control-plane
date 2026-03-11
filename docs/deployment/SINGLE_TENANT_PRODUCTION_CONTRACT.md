# Single-Tenant Production Contract

The production contract is the host-first Docker deployment path described in [DEPLOYMENT.md](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/DEPLOYMENT.md).

## Canonical Contract

1. Validate production config against `/etc/ai-control-plane/secrets.env`.
2. Run `./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env`.
3. Run `./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml`.
4. Run `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml`.
5. Run `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`.

## Invariants

- `/etc/ai-control-plane/secrets.env` is the canonical production env source.
- `demo/.env` is local-demo only.
- Production workflows do not sync secrets back into the repository tree.
- The supported runtime remains the host-first Docker baseline, with overlays only when explicitly selected.
