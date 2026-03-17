# Troubleshooting Index

Use this page when the supported workflow does not behave as expected.

## Runtime bring-up problems

- `make up` fails or services never become healthy:
  - check [Operations And Deployment](../DEPLOYMENT.md)
  - run `./scripts/acpctl.sh doctor`
  - run `make health`

## Config validation failures

- Run `make validate-config`
- For production secrets and host-contract failures, run:
  - `make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env`

## Host deployment problems

- [Operations And Deployment](../DEPLOYMENT.md)
- [Production handoff runbook](../deployment/PRODUCTION_HANDOFF_RUNBOOK.md)

## Security and governance validation failures

- [Security And Governance](../SECURITY_GOVERNANCE.md)
- `make validate-detections`
- `make validate-siem-queries`
- `make security-gate`

## Docs and reference drift failures

- Run `make generate`
- Run `make validate-generated-docs`
- Confirm generated files under `docs/reference/` and `scripts/completions/` were committed
