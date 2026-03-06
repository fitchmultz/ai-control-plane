# Validation Checklist

Use this exact sequence from a fresh clone to validate correctness, safety, and operational readiness quickly.

Recommended setup:

```bash
git clone <repo-url>
cd ai-control-plane
```

## 1) Deterministic PR gate

```bash
make install-ci
make ci-pr
```

Expected:
- exits 0
- shell/Go checks pass
- `acpctl` contract tests pass
- no tracked local-only artifacts/secrets (`public-hygiene-check`)

## 2) Full local CI gate (runtime included)

```bash
make ci
```

Expected:
- exits 0
- offline runtime boots in CI slot
- gateway readiness checks pass
- detection validation passes
- CI runtime volumes are torn down automatically

If Docker runtime is unavailable on the review machine, run the deterministic subset instead:

```bash
make ci-fast
go test ./...
```

## 3) Runtime smoke (manual proof)

```bash
ACP_SLOT=ci-runtime make up-offline
ACP_SLOT=ci-runtime make ci-runtime-checks
ACP_SLOT=ci-runtime make down-offline-clean
```

Expected:
- health and model endpoints become ready
- detection rules validate
- teardown removes runtime volumes cleanly

## 4) Release integrity checks

```bash
make release-bundle
make release-bundle-verify
```

Expected:
- release bundle generated with checksums
- verification passes

## 5) Security/supply-chain checks

```bash
make security-gate
```

Expected:
- hygiene/secrets/license/supply-chain checks pass

## 6) Optional manual-heavy security tier

```bash
make ci-manual-heavy
```

Expected:
- exits 0 when Trivy is installed
- hardened images build and vulnerability scans pass

## Quick fail triage

- If runtime checks fail at startup, inspect: `docker compose -f demo/docker-compose.offline.yml logs litellm`.
- If hygiene check fails, untrack local-only files listed in `make public-hygiene-check` output.
- If supply-chain gate fails, update allowlist policy windows and digests per `demo/config/supply_chain_vulnerability_policy.json`.
- If `ci-manual-heavy` fails with missing Trivy, install Trivy first (`https://trivy.dev/latest/getting-started/installation/`).

## Scope Reminder

This checklist proves the repository's validated baseline. It does not, by itself, prove customer-environment network enforcement, IdP ownership, workspace governance maturity, or compliance attestation.
