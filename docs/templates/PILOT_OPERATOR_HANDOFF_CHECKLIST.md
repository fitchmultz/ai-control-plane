# Pilot Operator Handoff Checklist

Use this checklist when the pilot is moving from build activity into day-to-day operation.

---

## Named Owners

- Customer operations lead: [CUSTOMER_OPS_LEAD]
- Customer security lead: [CUSTOMER_SECURITY_LEAD]
- Delivery lead: [DELIVERY_LEAD]
- Escalation contact: [ESCALATION_CONTACT]

## Minimum Command Set Confirmed

- [ ] `make health`
- [ ] `./scripts/acpctl.sh status`
- [ ] `make readiness-evidence`
- [ ] `make readiness-evidence-verify`
- [ ] `./scripts/acpctl.sh validate detections`
- [ ] `./scripts/acpctl.sh validate siem-queries --validate-schema`

## Operational Readiness

- [ ] Runbook location has been shared
- [ ] Evidence output location has been shared
- [ ] Incident routing destination is documented
- [ ] Change approval contact is documented
- [ ] Rollback decision owner is documented
- [ ] Customer-owned dependencies are documented in writing

## Customer Dependencies Confirmed

- [ ] Network egress owner engaged
- [ ] IAM owner engaged
- [ ] SIEM owner engaged
- [ ] FinOps/report consumer engaged
- [ ] Workspace admin engaged where applicable

## Handoff Decision

- [ ] Pilot can enter day-to-day operation
- [ ] Pilot remains in delivery-led stabilization
- [ ] Pilot is blocked pending customer dependency
