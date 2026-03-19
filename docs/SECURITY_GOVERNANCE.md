# Security And Governance

The supported host-first surface keeps security and governance checks close to the typed core.

## Enforced Areas

- Config validation through `acpctl validate config`
- Secrets and repo hygiene checks through `internal/security`
- Approved-model governance from [demo/config/model_catalog.yaml](../demo/config/model_catalog.yaml)
- Detection and SIEM contract validation through `acpctl validate detections` and `acpctl validate siem-queries`
- Truthful runtime health and smoke gates through `status`, `health`, `smoke`, and `doctor`
- Supply-chain governance through the live register, machine-readable allowlist, expiry checks, and quarterly review records

## CVE and vulnerability governance

Use these artifacts together for the canonical vulnerability-governance surface:

- [Known Limitations](KNOWN_LIMITATIONS.md) — human-readable status register
- [CVE Remediation and Risk Acceptance Policy](security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md) — canonical process and buyer-safe communication rules
- [CVE Review Log](security/CVE_REVIEW_LOG.md) — dated quarterly and off-cycle review output
- [`demo/config/supply_chain_vulnerability_policy.json`](../demo/config/supply_chain_vulnerability_policy.json) — machine-readable accepted-risk records with expiry dates

## Canonical Commands

```bash
make validate-detections
make validate-siem-queries
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make supply-chain-gate
make supply-chain-allowlist-expiry-check
make security-gate
```

## References

- [Compliance Crosswalk](COMPLIANCE_CROSSWALK.md)
- [Security Whitepaper and Threat Model](security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md)
- [CVE Remediation and Risk Acceptance Policy](security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md)
- [CVE Review Log](security/CVE_REVIEW_LOG.md)
- [Approved Models](reference/approved-models.md)
- [Detection Rules Reference](reference/detections.md)
- [Support Matrix](reference/support-matrix.md)
