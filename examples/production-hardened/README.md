# Production-Hardened Example

Use this example when preparing a host-first production-like deployment.

## Commands

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.example.yml
```

## References

- `deploy/ansible/`
- `deploy/systemd/`
- `docs/DEPLOYMENT.md`
- `docs/deployment/PRODUCTION_HANDOFF_RUNBOOK.md`
