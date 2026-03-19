# Vendor Evidence Example

This example shows the supported `compliance-api` ingest shape for `acpctl evidence ingest`.

## Run it

```bash
./scripts/acpctl.sh evidence ingest \
  --format compliance-api \
  --file examples/vendor-evidence/compliance_export.sample.json
```

## What it demonstrates

- file-based ingest on the supported host-first surface
- normalization into the tracked ACP evidence schema
- local artifact output under `demo/logs/evidence/vendor-ingest/`

For the full workflow, see `docs/evidence/VENDOR_EVIDENCE_INGEST.md`.
