# Go/No-Go Decision

This file defines how to read the current go/no-go decision for the repository baseline.

The canonical current decision is generated locally by the readiness workflow, not maintained as a stale committed snapshot. Generate it with:

```bash
make readiness-evidence
make readiness-evidence-verify
```

The generated decision memo for the latest run lives at:

- `demo/logs/evidence/readiness-<TIMESTAMP>/go-no-go-decision.md`
- `demo/logs/evidence/latest-run.txt`
- `demo/logs/evidence/latest-success.txt`

## Decision Rule

- `GO`: all required gates in the generated run pass
- `NO_GO`: any required gate fails
- Production-gate results are included only when a real secrets file is provided and the run explicitly includes that gate

## Scope Rule

A `GO` decision from this workflow is a statement about the repository's validated baseline. It is not a substitute for customer-environment validation of network controls, IdP policy, browser/workspace governance, or customer SIEM operations.

## Related Documents

- [READINESS_EVIDENCE_WORKFLOW.md](READINESS_EVIDENCE_WORKFLOW.md)
- [PRESENTATION_READINESS_TRACKER.md](PRESENTATION_READINESS_TRACKER.md)
- [../PILOT_CONTROL_OWNERSHIP_MATRIX.md](../PILOT_CONTROL_OWNERSHIP_MATRIX.md)
