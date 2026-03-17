# Falcon Insurance Group Customer Validation Checklist

Use this checklist to keep the pilot honest in the customer environment.

## Network / Egress Controls

Owner: `NETWORK_OWNER`

| Validation item | Status | Evidence |
| --- | --- | --- |
| Approved AI endpoints for routed traffic are documented | Validated | Change request CR-1042 |
| Firewall, SWG, CASB, proxy, or DNS controls for unapproved AI endpoints are identified | In Progress | Network design review |
| Bypass-prevention stance is explicitly documented | Validated | Pilot boundary memo |
| Direct SaaS AI traffic test was performed and outcome recorded | Validated | Test record BR-22 |
| Escalation path for bypass findings is agreed | Validated | SOC escalation matrix |

## Identity / IAM

Owner: `IAM_OWNER`

| Validation item | Status | Evidence |
| --- | --- | --- |
| Pilot user cohort is defined | Validated | IAM cohort export |
| IdP or workspace identity mapping for routed usage is tested | Validated | Identity validation notes |
| Required MFA and device posture policy is confirmed | Validated | Endpoint policy reference |
| Joiner/mover/leaver ownership is documented | Validated | IAM runbook |
| Privileged admin access path is documented | Validated | Break-glass procedure |

## SIEM / SOC

Owner: `SIEM_OWNER`

| Validation item | Status | Evidence |
| --- | --- | --- |
| Gateway evidence reaches the customer SIEM | Validated | Dashboard screenshot |
| Detection mappings are validated against pilot config | Validated | Detection test run |
| Alert routing destination is configured | Validated | Pager destination review |
| Investigation owner for pilot alerts is named | Validated | SOC owner list |
| Retention and case-management expectations are documented | Validated | Retention policy excerpt |

## Platform Operations

Owner: `PLATFORM_OWNER`

| Validation item | Status | Evidence |
| --- | --- | --- |
| Target host or environment is defined | Validated | Host build sheet |
| Change window is approved | Validated | CAB approval |
| Backup and restore expectations are documented | In Progress | DR work item |
| Named operator is available for checkpoint reviews | Validated | Ops rota |
| Release evidence review cadence is agreed | Validated | Weekly pilot review agenda |
