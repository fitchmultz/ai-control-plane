# Falcon Insurance Group Operator Handoff Checklist

## Named Owners

- Customer operations lead: claims platform operations manager
- Customer security lead: security operations manager
- Delivery lead: AI Control Plane delivery lead
- Escalation contact: pilot program manager

## Minimum Command Set Confirmed

- [x] `make health`
- [x] `./scripts/acpctl.sh status`
- [x] `make readiness-evidence`
- [x] `make readiness-evidence-verify`
- [x] `make validate-detections`
- [x] `make validate-siem-schema`

## Operational Readiness

- [x] Runbook location has been shared
- [x] Evidence output location has been shared
- [x] Incident routing destination is documented
- [x] Change approval contact is documented
- [x] Rollback decision owner is documented
- [x] Customer-owned dependencies are documented in writing

## Customer Dependencies Confirmed

- [x] Network egress owner engaged
- [x] IAM owner engaged
- [x] SIEM owner engaged
- [x] FinOps/report consumer engaged
- [x] Workspace admin engaged where applicable

## Handoff Decision

- [x] Pilot can enter day-to-day operation
- [ ] Pilot remains in delivery-led stabilization
- [ ] Pilot is blocked pending customer dependency
