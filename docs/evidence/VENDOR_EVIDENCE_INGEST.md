# Vendor Evidence Ingest

Use `acpctl evidence ingest` to normalize supported vendor audit exports into the tracked ACP evidence schema.

## Support boundary

This is a **local host-first ingest workflow**.

- Supported input surfaces: JSON file or stdin
- Supported output: private local artifacts under `demo/logs/evidence/vendor-ingest/`
- Not claimed: an always-on webhook service, hosted control plane, or cloud ingestion backend

If you run your own webhook receiver, have it write the raw JSON payload to disk or pipe it to `acpctl evidence ingest`.

## Supported format

Today ACP supports one concrete typed format:

- `compliance-api` — compliance export payloads aligned to the `compliance_api_to_schema` mapping in `demo/config/normalized_schema.yaml`

Accepted JSON shapes:

1. Single wrapped record
2. Single raw record
3. Array of wrapped/raw records
4. Object with a top-level `records` array

## Quick start

### File input

```bash
./scripts/acpctl.sh evidence ingest \
  --format compliance-api \
  --file examples/vendor-evidence/compliance_export.sample.json
```

### Stdin input

```bash
cat examples/vendor-evidence/compliance_export.sample.json \
  | ./scripts/acpctl.sh evidence ingest --format compliance-api
```

## Generated artifacts

Each run creates a dated directory under:

```text
demo/logs/evidence/vendor-ingest/vendor-ingest-<TIMESTAMP>/
```

Artifacts:

- `raw-input.json` — captured source payload
- `normalized-records.json` — normalized ACP evidence records
- `validation-issues.txt` — schema issues, if any
- `summary.json` — machine-readable run summary
- `vendor-evidence-summary.md` — human-readable run summary
- `ingest-inventory.txt` — artifact inventory
- `latest-run.txt` — latest ingest pointer under `demo/logs/evidence/vendor-ingest/`

## Normalized fields

The workflow validates required normalized fields from `demo/config/normalized_schema.yaml`, including:

- `principal.id`
- `principal.type`
- `ai.model.id`
- `ai.provider`
- `ai.request.id`
- `ai.request.timestamp`
- `policy.action`
- `source.type`

For the current `compliance-api` format, ACP emits `source.type=compliance_api`.

## Webhook handoff pattern

If you already receive vendor webhook deliveries locally, keep the receiver simple:

1. verify the vendor signature in your receiver
2. write the raw JSON body to a private file
3. call `acpctl evidence ingest --format compliance-api --file <path>`
4. archive or forward the normalized artifacts as needed

Example:

```bash
curl -fsS https://vendor.example.test/export/latest \
  -o /tmp/vendor-export.json

./scripts/acpctl.sh evidence ingest \
  --format compliance-api \
  --file /tmp/vendor-export.json
```

## Validation behavior

- Exit `0` when normalization and schema validation pass
- Exit `1` when normalization ran but schema validation found issues
- Exit `64` for usage problems such as missing input
- Exit `3` for runtime failures such as unreadable files or malformed JSON

## Related documents

- [Webhook Events Reference](../security/WEBHOOK_EVENTS.md)
- [Normalized Evidence Schema](../../demo/config/normalized_schema.yaml)
- [SIEM Integration Guide](../security/SIEM_INTEGRATION.md)
- [Evidence Map](EVIDENCE_MAP.md)
