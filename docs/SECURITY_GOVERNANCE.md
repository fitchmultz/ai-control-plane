# Security And Governance

The supported host-first surface keeps security and governance checks close to the typed core.

## Enforced Areas

- Config validation through `acpctl validate config`
- Secrets and repo hygiene checks through `internal/security`
- Approved-model governance from [demo/config/model_catalog.yaml](../demo/config/model_catalog.yaml)
- Detection and SIEM contract validation through `acpctl validate detections` and `acpctl validate siem-queries`
- Truthful runtime health and smoke gates through `status`, `health`, `smoke`, and `doctor`

## Canonical Commands

```bash
make validate-detections
make validate-siem-queries
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make security-gate
```

## References

- [Approved Models](reference/approved-models.md)
- [Detection Rules Reference](reference/detections.md)
- [Support Matrix](reference/support-matrix.md)
