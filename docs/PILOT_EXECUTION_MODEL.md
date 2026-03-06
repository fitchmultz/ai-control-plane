# Pilot Execution Model

This is the strict execution model for customer pilots.

Its job is simple: prevent the team from calling something a credible enterprise pilot before the required owners, controls, evidence, and decisions exist.

## Core Rule

A pilot is only credible when these four things are true at the same time:

1. The routed control path is working and evidenced.
2. Customer-owned controls are named and actively validated.
3. Enforce-vs-detect boundaries are written down in plain language.
4. The next decision is explicit: `EXPAND`, `REMEDIATE_AND_CONTINUE`, or `NO_GO`.

If any one of those is missing, the engagement is a reference validation exercise, not an enterprise rollout-readiness pilot.

## Phase Model

### Phase 0: Qualify

Goal:
- Decide whether the opportunity should be sold as a pilot at all.

Entry conditions:
- Customer has a real sponsor.
- Customer wants to validate governance, not just watch a demo.

Required owners:
- Sponsor
- Network owner
- IAM owner
- SIEM owner
- Platform owner
- FinOps owner if spend governance is in scope

Required outputs:
- Initial scope statement
- Named owners list
- Decision date target

Hard stop conditions:
- No named network owner
- No named SIEM owner
- Sponsor wants universal control claims the repo does not support

If a hard stop is hit:
- Reframe as assessment or demo, not pilot

### Phase 1: Charter

Goal:
- Lock the pilot boundary before implementation work starts.

Required outputs:
- Pilot charter
- Customer validation checklist with named owners
- Initial enforce-vs-detect statement
- Initial success criteria

Must be true before moving on:
- Customer acknowledges bypass prevention is partly customer-owned
- Customer agrees the host-first routed path is the validated baseline
- Target environment and change window are known

### Phase 2: Implement

Goal:
- Stand up the routed baseline and operator path cleanly.

Required outputs:
- Healthy deployment
- Approved-model and budget policy baseline
- Detection pack and SIEM mapping validation
- Operator runbook and checkpoint command set

Minimum commands:

```bash
make ci
make health
./scripts/acpctl.sh validate detections
./scripts/acpctl.sh validate siem-queries --validate-schema
```

Must be true before moving on:
- Routed enforcement works for the scoped path
- Evidence generation is current
- Operators can run the agreed command set

### Phase 3: Validate Customer Controls

Goal:
- Prove what only the customer can prove in their own environment.

Customer validation areas:
- Network / egress
- IAM / identity mapping
- SIEM ingestion and alert routing
- Workspace / browser governance
- FinOps review workflow

Required outputs:
- Updated customer validation checklist
- Evidence references for every critical item
- Explicit accepted-risk statements where validation is incomplete

Hard stop conditions:
- Customer wants bypass-blocking claims without network validation
- SIEM path is not working and no owner is assigned
- Workspace/browser ownership is undefined while browser governance is in scope

### Phase 4: Decide

Goal:
- Convert evidence into a decision without ambiguity.

Required inputs:
- Current readiness evidence
- Current validation checklist
- Acceptance memo
- Closeout bundle

Required commands:

```bash
make readiness-evidence
make readiness-evidence-verify
make pilot-closeout-bundle
make pilot-closeout-bundle-verify
```

Allowed decisions:
- `EXPAND`
- `REMEDIATE_AND_CONTINUE`
- `NO_GO`

Forbidden outcomes:
- “Successful pilot” with no decision
- “Enterprise-ready” with unresolved control ownership
- “Proceed carefully” without named remediation owners

### Phase 5: Transition

Goal:
- Move cleanly into implementation expansion, managed operations, or stop.

Required outputs:
- Signed acceptance memo
- Shared-responsibility agreement
- Named next-phase owners
- Explicit list of open risks that remain customer-owned

## Decision Rules

### Expand

Use `EXPAND` only when:
- routed enforcement is proven
- customer-owned critical controls are validated or accepted in writing
- sponsor and control owners agree on the next deployment scope

### Remediate And Continue

Use `REMEDIATE_AND_CONTINUE` when:
- the product baseline is credible
- one or more customer-owned validations are incomplete
- the missing work is bounded and assigned

### No-Go

Use `NO_GO` when:
- the required owners never materialized
- the desired claims exceed what the repo and customer environment can currently prove
- the control boundary is materially weaker than the buyer requires

## Non-Negotiable Writing Rules

Every pilot readout must state:
- what was enforced
- what was only detected
- what the customer owned
- what decision was made

Never say:
- “enterprise-ready” without naming the environment
- “blocked all AI usage”
- “fully governed” when bypass remains detective-only
- “successful” without a boundary statement
