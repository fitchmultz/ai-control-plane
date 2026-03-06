# Pilot Control Ownership Matrix

This matrix makes the pilot honest and executable. It separates what the repository validates locally from what the customer must validate in their own environment.

## Control Matrix

| Control Area | Repository Validates | Customer Owns in Pilot | Delivery Team Owns in Pilot | Proof Artifact |
| --- | --- | --- | --- | --- |
| Gateway enforcement | Routed traffic can be approved, blocked, budgeted, and attributed through the AI Control Plane baseline | Provide target use cases and confirm which traffic classes must be forced through the gateway | Configure the gateway baseline, key model aliases, and policy controls | `make ci`; `make readiness-evidence`; demo scenarios |
| Network egress / bypass prevention | Detection-and-response patterns, network contract artifacts, and architecture boundaries | Enforce SWG, firewall, DNS, CASB, or proxy policy to stop unapproved AI endpoints | Map required AI endpoints and provide rollout guidance | `docs/deployment/network_firewall_contract.md`; customer firewall change records |
| Identity and access | Trusted attribution model for LibreChat + LiteLLM and role examples | Integrate IdP, group mapping, MFA, device posture, and joiner/mover/leaver process | Configure claim mapping and validate attribution behavior | `docs/security/ENTERPRISE_AUTH_ARCHITECTURE.md`; customer IdP test evidence |
| SIEM ingestion | Normalized evidence schema, detections, and query validation | Onboard the feed into the customer SIEM, retention, alert routing, and SOC workflow | Tune mappings, detections, and investigation runbooks | `./scripts/acpctl.sh validate detections`; `./scripts/acpctl.sh validate siem-queries --validate-schema`; customer SIEM screenshots or test cases |
| FinOps / chargeback | Gateway spend attribution, chargeback query workflow, and monthly report path | Supply cost centers, cost model, and downstream finance workflow | Map aliases/tags and deliver the report workflow | `make chargeback-report`; chargeback SQL export; finance sign-off |
| Browser / workspace governance | Managed browser chat routed through LibreChat and evidence-aware architecture | Configure vendor workspace admin settings, browser management, sanctioned extensions, and data-retention policy | Validate the governed workspace pattern and handoff runbooks | [BROWSER_WORKSPACE_PROOF_TRACK.md](BROWSER_WORKSPACE_PROOF_TRACK.md) |
| Platform operations | Local host-first operating baseline, health checks, release bundle, and rollback path | Provide target host/cloud ownership, backup retention, and change window approvals | Run deployment, release, and remediation workflows for the delivered platform | `make release-bundle`; `make readiness-evidence`; runbook walkthrough |

## Minimum Customer Stakeholders

- Network/security owner
- Identity owner
- SIEM or SOC owner
- Finance/FinOps owner
- Workspace or collaboration-suite admin
- Application/platform operations owner
- Executive sponsor

If these owners are unavailable, the pilot can still demonstrate local product behavior, but it cannot credibly prove enterprise rollout readiness.
