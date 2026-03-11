# AI Control Plane

AI Control Plane is a host-first Docker reference implementation for enterprise AI governance. The supported product surface is the minimal LiteLLM plus PostgreSQL runtime, backed by a typed operator core, truthful runtime gates, validation/security checks, and local evidence workflows.

## Supported Surface

- Host-first Docker deployment
- `make` as the primary operator interface
- `acpctl` for typed workflows: status, health, smoke, doctor, validate, db, key, host, chargeback, onboarding, and evidence artifacts
- Optional supported overlays:
  - Managed browser UI via `make up-ui`
  - DLP via `make up-dlp`
  - TLS ingress via `make up-tls`
  - Both overlays together via `make up-full`
- Offline deterministic runtime via `make up-offline`

The machine-readable support contract lives in [docs/support-matrix.yaml](docs/support-matrix.yaml). The generated public view lives in [docs/reference/support-matrix.md](docs/reference/support-matrix.md).

## Quick Start

Local host-first baseline:

```bash
make install
make up
make health
make prod-smoke
```

Optional overlays:

```bash
make up-dlp
make up-ui
make up-full
make up-offline
make up-tls
```

Managed browser UI requires the LibreChat keys documented in [`demo/.env.example`](demo/.env.example). Host-first overlay selection is driven by the Ansible inventory variable `acp_runtime_overlays`.

Host-first production path:

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
```

## Canonical Docs

- [Support](docs/SUPPORT.md)
- [Architecture](docs/technical-architecture.md)
- [Operations And Deployment](docs/DEPLOYMENT.md)
- [Security And Governance](docs/SECURITY_GOVERNANCE.md)

Generated references:

- [ACPCTL Reference](docs/reference/acpctl.md)
- [Approved Models](docs/reference/approved-models.md)
- [Detection Rules](docs/reference/detections.md)

## Cutover Notes

- Removed public `acpctl` roots for demo, incubating deployment tracks, and bridge mirroring.
- Removed `host secrets-refresh`; host production now reads `/etc/ai-control-plane/secrets.env` directly.
- Moved incubating deployment assets under `deploy/incubating/`.
- Overlay mapping is explicit: `make up-dlp`, `make up-ui`, `make up-full`, `make up-offline`, and `make up-tls`.

## Local Env Files

- `demo/.env` is local-demo only.
- `/etc/ai-control-plane/secrets.env` is the canonical host-production env source.
- Production workflows do not sync secrets back into the repository tree.
