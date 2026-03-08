# Evidence Map

## Production workflow design

- **Claim:** The repo exposes clear operator entrypoints and runbooks.
- **Evidence:** `README.md`, `docs/README.md`, `docs/RUNBOOK.md`, `docs/technical-architecture.md`.

## Rollout and release confidence

- **Claim:** Release workflow is reproducible and verifiable.
- **Evidence:** `make release-bundle`, `make release-bundle-verify`, `internal/bundle/*`, `internal/readiness/*`, `internal/closeout/*`, `internal/artifactrun/*`, `docs/release/PRESENTATION_READINESS_TRACKER.md`, `docs/release/go_no_go_decision.md`.

## Reliability and determinism

- **Claim:** PR checks are fast/high-signal; runtime checks are isolated and deterministic.
- **Evidence:** `mk/ci.mk` (`ci-pr`, `ci`, `ci-nightly`, `ci-manual-heavy`), `mk/offline.mk` (`down-offline-clean`).
- **Run receipts (2026-03-05):**
  - `make ci-pr` passed in ~7.82s (warm cache)
  - `make ci` passed in ~56.48s (warm cache)

## Safety and security posture

- **Claim:** Public-release hygiene and supply-chain boundaries are enforced.
- **Evidence:** `mk/security.mk` (`public-hygiene-check`, `security-gate`, hardened image scan targets), `docs/ARTIFACTS.md`, `SECURITY.md`.

## Developer productivity

- **Claim:** Local-first, repeatable make targets cover install, CI, runtime checks, and release verification.
- **Evidence:** `Makefile` help categories and tiered CI targets, `docs/release/VALIDATION_CHECKLIST.md`.
