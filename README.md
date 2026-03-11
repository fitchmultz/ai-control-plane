# AI Control Plane

AI Control Plane is a host-first Docker reference implementation for enterprise AI governance. The supported product surface is the minimal LiteLLM plus PostgreSQL runtime, backed by a typed operator core, truthful runtime gates, validation/security checks, and local evidence workflows.

## Supported Surface

- Host-first Docker deployment
- `make` as the primary operator interface
- `acpctl` for typed workflows: status, health, smoke, doctor, validate, db, key, host, chargeback, onboarding, and evidence artifacts
- Optional supported overlays:
  - Managed browser UI via `make up-ui`
  - DLP via `make up-dlp`
  - Both overlays together via `make up-full`
- Offline deterministic runtime via `make up-offline`

The machine-readable support contract lives in [support-matrix.yaml](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/support-matrix.yaml). The generated public view lives in [support-matrix.md](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/reference/support-matrix.md).

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
```

Host-first production path:

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
```

## Canonical Docs

- [Support](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/SUPPORT.md)
- [Architecture](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/technical-architecture.md)
- [Operations And Deployment](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/DEPLOYMENT.md)
- [Security And Governance](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/SECURITY_GOVERNANCE.md)

Generated references:

- [ACPCTL Reference](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/reference/acpctl.md)
- [Approved Models](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/reference/approved-models.md)
- [Detection Rules](/Users/mitchfultz/Projects/AI/ai-control-plane/docs/reference/detections.md)

## Local Env Files

- `demo/.env` is local-demo only.
- `/etc/ai-control-plane/secrets.env` is the canonical host-production env source.
- Production workflows do not sync secrets back into the repository tree.
