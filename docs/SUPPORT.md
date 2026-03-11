# Support

The supported product surface is the host-first Docker reference implementation. That means:

- `make up` starts the supported base runtime only: LiteLLM plus PostgreSQL.
- `make up-dlp`, `make up-ui`, and `make up-full` are explicit supported overlays.
- `make up-offline` is the supported deterministic offline path.
- `make up-tls` is the supported TLS ingress overlay.
- `make ci`, `make ci-pr`, `make prod-smoke`, and the typed `acpctl` host/runtime workflows are the supported validation surface.

Support levels are defined in [support-matrix.yaml](support-matrix.yaml) and rendered in [reference/support-matrix.md](reference/support-matrix.md).

## Operator Contract

- Use `make` for day-to-day operations.
- Use `./scripts/acpctl.sh` for typed workflows and machine-oriented tasks.
- Use `demo/.env` for local-only runs.
- Use `/etc/ai-control-plane/secrets.env` for host-production workflows.
- Select supported host overlays through `acp_runtime_overlays` in the Ansible inventory.

## Migration Notes

- Removed public `acpctl` groups for demo, incubating deployment tracks, and bridge delegation.
- Removed `host secrets-refresh`; production reads `/etc/ai-control-plane/secrets.env` directly.
- Moved incubating deployment assets into `deploy/incubating/`.

## Not Part Of The Supported Surface

Anything not listed as supported in the support matrix is not part of the primary operator UX or default validation contract.
