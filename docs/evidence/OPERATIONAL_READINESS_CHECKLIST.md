# Operational Readiness Checklist

Use this checklist before public release or major demo sessions.

## Configuration and secrets

- [ ] `demo/.env` exists locally and is not tracked by git.
- [ ] Placeholder keys have been replaced for local runtime use.
- [ ] `make public-hygiene-check` passes.

## Reliability gates

- [ ] `make ci-pr` passes.
- [ ] `make ci` passes.
- [ ] Runtime smoke path passes (`up-offline` + `ci-runtime-checks` + teardown).

## Security/supply chain

- [ ] `make security-gate` passes.
- [ ] Supply-chain allowlist windows are current.
- [ ] No secrets or internal-only assets are present in tracked files.

## Release artifacts

- [ ] `make release-bundle` passes.
- [ ] `make release-bundle-verify` passes.
- [ ] Release report/checklist docs are current.

## Rollout/rollback notes

- Rollout: `make up` (or `make up-offline` for provider-independent demos).
- Health validation: `make health` and `make doctor`.
- Rollback/cleanup: `make down` or CI-slot-specific `ACP_SLOT=ci-runtime make down-offline-clean`.
