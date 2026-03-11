# Production Handoff Runbook

This runbook covers the supported host-first handoff only.

## Handoff Checklist

- Confirm `/etc/ai-control-plane/secrets.env` exists with private permissions.
- Confirm `make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env` passes.
- Confirm `./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env` passes.
- Confirm `./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml` passes.
- Confirm `./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml` completes.
- Confirm `make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env` passes.
- Confirm readiness evidence and release artifacts are captured when required.

## Operator Notes

- The supported runtime is the host-first Docker baseline.
- Optional overlays must be called out explicitly in the handoff.
- Incubating deployment tracks are not part of the handoff contract.
