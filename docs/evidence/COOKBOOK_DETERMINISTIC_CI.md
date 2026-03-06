# Cookbook: Deterministic CI Tiering for Infrastructure Repos

## Problem

A single monolithic CI command either becomes too slow for PRs or too shallow to trust.

## Pattern used in this repo

1. Define **PR-required** checks as deterministic/high-signal only (`make ci-pr`).
2. Put runtime startup smoke in a **full local/nightly** tier (`make ci`, `make ci-nightly`).
3. Keep heavyweight security/image scans in a **manual-heavy** tier (`make ci-manual-heavy`).
4. Make runtime checks stateless using a dedicated slot + explicit teardown (`ACP_SLOT=ci-runtime`, `down-offline-clean`).

## Why it works

- Fast feedback on PRs.
- High confidence available on demand and nightly.
- Controlled resource usage and less flake pressure.

## Safe defaults

- Retry readiness checks with bounded attempts/timeouts.
- Avoid cross-test shared runtime state (use dedicated volumes/slot cleanup).
- Fail fast on hygiene/security/supply-chain violations.

## Adaptation checklist

- [ ] Separate checks by signal/cost.
- [ ] Keep PR gate runtime-free when possible.
- [ ] Add one explicit runtime smoke path with cleanup.
- [ ] Provide a single validation checklist with exact commands.
- [ ] Measure and publish baseline runtimes.
