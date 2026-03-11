# Disaster Recovery

The supported disaster-recovery story is the host-first Docker reference implementation.

## Supported Recovery Inputs

- Canonical secrets file: `/etc/ai-control-plane/secrets.env`
- Database backups created through the typed DB workflow
- Release bundle, readiness evidence, and pilot closeout artifacts when applicable

## Supported Recovery Flow

1. Restore host access and the canonical secrets file.
2. Restore the database using the typed DB workflow if required.
3. Re-apply the host deployment with `./scripts/acpctl.sh host apply --inventory ...`.
4. Verify runtime readiness with `make health` and `make prod-smoke`.

Incubating deployment tracks are intentionally excluded from the supported recovery contract.
