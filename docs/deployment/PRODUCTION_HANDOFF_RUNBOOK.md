# Production Handoff Runbook

This runbook covers the supported host-first handoff only.

## Handoff Checklist

- Confirm `/etc/ai-control-plane/secrets.env` exists with private permissions.
- Confirm `make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env` passes.
- Confirm `./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env` passes.
- Confirm `./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml` passes.
- Confirm `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml` completes.
- Confirm `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env` passes.
- Confirm `./scripts/acpctl.sh cert check --threshold-days 30` passes.
- Confirm `systemctl status ai-control-plane-cert-renewal.timer` is active when TLS is enabled.
- Capture `./scripts/acpctl.sh cert list` output in the handoff notes for the deployed hostname.
- Confirm a customer-owned off-host backup copy exists for the deployment and is not only retained under `demo/backups/` on the primary host.
- Confirm `demo/logs/recovery-inputs/off_host_recovery.yaml` (or the customer-approved equivalent manifest path) documents the staged off-host recovery inputs, including truthful `drill_mode` and `drill_host` values.
- Run `make db-off-host-drill OFF_HOST_RECOVERY_MANIFEST=demo/logs/recovery-inputs/off_host_recovery.yaml` or the customer-approved separate-host manifest path.
- Capture the latest successful run under `demo/logs/evidence/replacement-host-recovery/` in the handoff packet.
- In the handoff notes, label the evidence exactly as recorded by the manifest and evidence bundle: either `staged-local` or `separate-host`. Do not present a staged local drill as proof of real customer transport or separate-hardware replacement-host recovery.
- Confirm operators understand the replacement-host recovery sequence: initial `host apply --skip-smoke-tests` -> `db restore <off-host backup>` -> normal `host apply` -> health/smoke verification.
- Confirm the handoff notes explicitly state that the supported deployment is **single-node** unless a separate customer-owned HA design exists.
- Confirm operators understand that scheduled backups and restore drills are **disaster recovery**, not automatic failover.
- Review [HA_FAILOVER_TOPOLOGY.md](HA_FAILOVER_TOPOLOGY.md) whenever availability expectations are part of the handoff.
- Confirm readiness evidence and release artifacts are captured when required.

## Operator Notes

- The supported runtime is the host-first Docker baseline.
- Optional overlays must be called out explicitly in the handoff.
- If the customer needs HA beyond single-node recovery, document that design as customer-owned or separately validated. Do not imply repo-native automatic failover.
- Incubating deployment tracks are not part of the handoff contract.
