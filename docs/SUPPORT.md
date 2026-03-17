# Support

The supported product surface is the host-first Docker reference implementation. That means:

- `make up` starts the supported base runtime only: LiteLLM plus PostgreSQL.
- `make up-dlp`, `make up-ui`, and `make up-full` are explicit supported overlays.
- `make up-offline` is the supported deterministic offline path.
- `make up-tls` is the supported TLS ingress overlay.
- `make ci`, `make ci-pr`, `make prod-smoke`, and the typed `acpctl` host/runtime workflows are the supported validation surface.

Support levels are defined in [support-matrix.yaml](support-matrix.yaml) and rendered in [reference/support-matrix.md](reference/support-matrix.md).

Start with [README.md](../README.md) for the public repo overview, [troubleshooting/README.md](troubleshooting/README.md) for failure-mode triage, and [../examples/README.md](../examples/README.md) for curated operator examples.

## Operator Contract

- Use `make` for day-to-day operations.
- Use `./scripts/acpctl.sh` for typed workflows and machine-oriented tasks.
- Use `demo/.env` for local-only runs.
- Use `/etc/ai-control-plane/secrets.env` for host-production workflows.
- Select supported host overlays through `acp_runtime_overlays` in the Ansible inventory.
- Keep `acp_public_url` loopback-only unless the `tls` overlay is enabled.
- Expect the supported host path to verify SSH host keys and enforce baseline host hardening (UFW defaults, unattended security updates, SSH hardening, private secrets-file permissions).
- Treat outbound allow-listing, SWG/CASB policy, and broader perimeter controls as customer-owned responsibilities outside the host playbook.

## Migration Notes

- Removed public `acpctl` groups for demo, incubating deployment tracks, and bridge delegation.
- Removed `host secrets-refresh`; production reads `/etc/ai-control-plane/secrets.env` directly.
- Moved incubating deployment assets into `deploy/incubating/`.
- Hardened the supported Ansible host path around Debian 12+/Ubuntu 24.04+, verified SSH host keys, explicit firewall defaults, and automatic security updates.

## Not Part Of The Supported Surface

Anything not listed as supported in the support matrix is not part of the primary operator UX or default validation contract.
