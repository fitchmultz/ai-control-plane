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
- Supported host overlays are selected through `acp_runtime_overlays` in the Ansible inventory. Allowed values are `tls`, `ui`, `dlp`, and `offline`.
- Base host deployment remains the default. Add overlays only when the host contract explicitly requires them.

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
