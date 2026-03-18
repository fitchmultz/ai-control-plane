# Troubleshooting Index

Use this page when the supported workflow does not behave as expected.

## Runtime bring-up problems

- `make up` fails or services never become healthy:
  - check [Operations And Deployment](../DEPLOYMENT.md)
  - run `./scripts/acpctl.sh doctor`
  - run `make health`

- local hardened LiteLLM fails with Prisma or filesystem permission errors:
  - common symptoms:
    - `prisma.engine.errors.NotConnectedError: Not connected to the query engine`
    - `Permission denied` writing under `litellm/proxy/_experimental/out`
    - noisy `P1012` warnings for duplicate model `LiteLLM_DeletedTeamTable` during the post-migration sanity check
  - `make up` builds and uses the local hardened image: `ai-control-plane/litellm-hardened:local`
  - rebuild the local hardened image: `make hardened-images-build`
  - the hardened image now hotfixes the current upstream `litellm_proxy_extras/schema.prisma` duplicate-model packaging defect during image build and clears generated `*_baseline_diff` migrations before each startup; if you still see the warning, confirm the local image was rebuilt before startup
  - inspect gateway logs with Docker Compose or `docker logs`
  - CI intentionally uses the pinned fallback image for offline runtime checks by clearing `ACP_RUNTIME_LITELLM_IMAGE`; do not remove that override when debugging local runtime issues

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
