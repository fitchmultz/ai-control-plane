# Support

The supported product surface is the **single-node** host-first Docker reference implementation. That means:

- `make up` starts the supported base runtime only: LiteLLM plus PostgreSQL.
- `make up-dlp`, `make up-ui`, and `make up-full` are explicit supported overlays.
- `make up-offline` is the supported deterministic offline path.
- `make up-tls` is the supported TLS ingress overlay.
- `make ci`, `make ci-pr`, `make prod-smoke`, and the typed `acpctl` host/runtime workflows are the supported validation surface.
- `make ha-failover-drill` and `./scripts/acpctl.sh host failover-drill --manifest ...` are the supported customer-operated active-passive HA failover-drill evidence surface.

Support levels are defined in [support-matrix.yaml](support-matrix.yaml) and rendered in [reference/support-matrix.md](reference/support-matrix.md).

Start with [README.md](../README.md) for the public repo overview, [troubleshooting/README.md](troubleshooting/README.md) for failure-mode triage, and [../examples/README.md](../examples/README.md) for curated operator examples.

## Operator Contract

- Use `make` for day-to-day operations.
- Use `./scripts/acpctl.sh` for typed workflows and machine-oriented tasks.
- Use `demo/.env` for local-only runs.
- Use `/etc/ai-control-plane/secrets.env` for host-production workflows.
- Select supported host overlays through `acp_runtime_overlays` in the Ansible inventory.
- Keep `acp_public_url` loopback-only unless the `tls` overlay is enabled.
- Expect the supported host path to verify SSH host keys, enforce baseline host hardening (UFW defaults, unattended security updates, SSH hardening, private secrets-file permissions), install the automated backup timer contract, install the certificate renewal timer whenever the `tls` overlay is enabled, and use the typed upgrade framework for any future in-place release edge.
- Treat outbound allow-listing, SWG/CASB policy, and broader perimeter controls as customer-owned responsibilities outside the host playbook.

## Active-Passive Failover Drill Evidence

The supported HA surface in this repository is the **manual evidence workflow**, not automatic cluster control.

Supported entrypoints:

- `make ha-failover-drill`
- `./scripts/acpctl.sh host failover-drill --manifest <path>`

Claim boundary:

> Manual customer-operated active-passive failover proof only. ACP validates the drill contract and archives evidence for replication readiness, fencing, promotion, traffic cutover, and post-cutover checks. ACP does not automate PostgreSQL replication, promotion, or customer-owned DNS/load-balancer/VIP cutover.

Use [deployment/HA_FAILOVER_RUNBOOK.md](deployment/HA_FAILOVER_RUNBOOK.md) for the operator sequence and [deployment/HA_FAILOVER_TOPOLOGY.md](deployment/HA_FAILOVER_TOPOLOGY.md) for topology truth, failure domains, and decision guidance.

## Availability Boundary

- The primary supported deployment topology is **single-node** today.
- Scheduled backups, restore drills, and typed recovery workflows are part of the supported contract.
- A **validated customer-operated manual active-passive failover drill evidence workflow** is also supported.
- Automatic failover to a secondary host is **not** part of the supported surface.
- Customer-owned DNS, load balancers, off-host backups, fencing, PostgreSQL replication, and network infrastructure determine any broader availability posture beyond the supported ACP surfaces.

## Migration Notes

- Removed public `acpctl` groups for demo, incubating deployment tracks, and bridge delegation.
- Removed `host secrets-refresh`; production reads `/etc/ai-control-plane/secrets.env` directly.
- Moved incubating deployment assets into `deploy/incubating/`.
- Hardened the supported Ansible host path around Debian 12+/Ubuntu 24.04+, verified SSH host keys, explicit firewall defaults, and automatic security updates.
- Added typed certificate lifecycle workflows for Caddy-managed TLS on the supported host-first path.
- Added the typed host-first upgrade framework; only explicit release edges may claim in-place support.

## Not Part Of The Supported Surface

Anything not listed as supported in the support matrix is not part of the primary operator UX or default validation contract.
