# Operations And Deployment

This document describes the supported host-first deployment and operations path.

## Runtime Profiles

- Base runtime: `make up`
- Base plus DLP overlay: `make up-dlp`
- Base plus managed UI overlay: `make up-ui`
- Base plus both overlays: `make up-full`
- Offline deterministic runtime: `make up-offline`
- TLS overlay: `make up-tls`

## Local Workflow

```bash
make install
make up
make health
make prod-smoke
```

Use `demo/.env` for local runs. It is local-only and not part of the host-production secret contract.

## Host-First Production Workflow

Canonical order:

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
```

Rules:

- `/etc/ai-control-plane/secrets.env` is the canonical production env source.
- Production workflows do not sync secrets into `demo/.env`.
- Compose-driven host operations use `COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env`.
- The base runtime remains LiteLLM plus PostgreSQL unless an overlay is explicitly selected.

## Typed Operational Checks

```bash
./scripts/acpctl.sh status
./scripts/acpctl.sh health
./scripts/acpctl.sh smoke
./scripts/acpctl.sh doctor
```

Use `make ci-pr` for fast deterministic checks and `make ci` for the full supported gate.

## References

- [Support](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/SUPPORT.md)
- [Architecture](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/technical-architecture.md)
- [Security And Governance](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/SECURITY_GOVERNANCE.md)
- [ACPCTL Reference](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/reference/acpctl.md)
