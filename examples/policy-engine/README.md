# Policy Engine Example

This directory contains a sample request/response payload for the local ACP custom policy engine.

## Quick start

Validate the tracked policy rules:

```bash
make validate-policy-rules
```

Run the sample evaluation:

```bash
make policy-eval
```

Or call the typed workflow directly:

```bash
./scripts/acpctl.sh policy eval \
  --file examples/policy-engine/request_response_eval.sample.json
```

The sample payload intentionally triggers:

- RBAC model restriction blocking
- prompt-injection detection blocking
- response SSN redaction

Generated artifacts are written under `demo/logs/evidence/policy-eval/`.
