# 45–60 Minute Workshop Outline

## Audience

Platform/security engineers evaluating AI governance infrastructure patterns.

## Agenda

1. **Context and architecture (10 min)**
   - Walk through `docs/technical-architecture.md` and runtime topology.

2. **Hands-on: deterministic quality gates (15 min)**
   - Run `make ci-pr`, inspect what is checked and why.
   - Run `make ci`, observe runtime stage behavior and cleanup.

3. **Hands-on: operational workflows (15 min)**
   - Run offline runtime checks and release bundle verify.
   - Inspect key operator docs (`docs/RUNBOOK.md`, `docs/DEPLOYMENT.md`).

4. **Safety/security deep dive (10 min)**
   - Review hygiene gate, supply-chain gate, and secret handling model.

5. **Debrief + Q&A (5–10 min)**
   - Discuss trade-offs (local-first CI, host-first baseline, optional K8s track).

## Success criteria

- Participants can run deterministic gates locally.
- Participants can identify where policy/security/release checks live.
- Participants can explain CI tier separation and why heavy checks are not in PR gate.

## Common failure modes to teach

- Docker daemon unavailable.
- Runtime startup lag during first migration.
- Missing optional heavy-tool prerequisites (e.g., Trivy) for manual-heavy tier.
