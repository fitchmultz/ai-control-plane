# Enterprise Buyer Objection Matrix

Direct answers to the highest-friction buyer objections.

Each answer is split into three parts:
- what the repo proves now
- what the customer must own
- what should not be claimed without additional validation

---

## 1. "Can users bypass this with their own keys or direct SaaS access?"

**Short answer:** Yes, unless customer-owned network and endpoint controls prevent it.

**Repo proof now:**
- Gateway-routed usage is enforceable.
- Direct/bypass usage can be treated as detective coverage with SIEM correlation and workspace evidence.
- Strategy and deployment docs explicitly separate enforce vs detect paths.

**Customer dependency:**
- SWG/CASB/firewall policy
- managed-device policy
- workspace admin controls

**Do not claim:**
- "We block all AI usage by default."
- "The repo alone prevents bypass."

Primary references:
- `docs/ENTERPRISE_STRATEGY.md`
- `docs/DEPLOYMENT.md`
- `docs/demo/q_and_a.md`

---

## 2. "Is this compliance-ready or certified?"

**Short answer:** It is control-oriented and evidence-oriented, not a compliance certification package.

**Repo proof now:**
- The repo maps controls, runbooks, detections, and evidence flows.
- The crosswalk is useful for scoping customer control ownership and evidence planning.

**Customer dependency:**
- environment-specific implementation
- customer control operation
- auditor interpretation and attestation

**Do not claim:**
- "FedRAMP-ready by default"
- "SOC 2 / ISO / CMMC certified by using this repo"

Primary references:
- `docs/GO_TO_MARKET_SCOPE.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`

---

## 3. "How do I know the detections and SIEM mappings are real instead of hand-written examples?"

**Short answer:** The repo now has typed validation for the detection pack and SIEM mappings.

**Repo proof now:**
- `./scripts/acpctl.sh validate detections`
- `./scripts/acpctl.sh validate siem-queries --validate-schema`
- These commands verify enabled-rule coverage, severity/category alignment, required platform queries, normalized schema mappings, and approved-model placeholder integrity.

**Customer dependency:**
- final SIEM implementation and alert routing in the customer environment

**Do not claim:**
- that repo validation alone proves customer ingestion, routing, or analyst workflow success

Primary references:
- `demo/config/detection_rules.yaml`
- `demo/config/siem_queries.yaml`
- `docs/security/DETECTION.md`
- `docs/security/SIEM_INTEGRATION.md`

---

## 4. "What happens when the gateway is down?"

**Short answer:** Routed workloads fail closed; direct vendor paths may continue outside inline enforcement depending on the customer topology.

**Repo proof now:**
- Health/status/doctor workflows exist.
- Runbooks define operator actions.
- Release bundle and validation workflow exist.

**Customer dependency:**
- HA topology
- operational ownership
- incident response routing
- egress design for fallback behavior

**Do not claim:**
- universal HA guarantees from the single-host reference baseline

Primary references:
- `docs/RUNBOOK.md`
- `docs/DEPLOYMENT.md`
- `docs/GO_TO_MARKET_SCOPE.md`

---

## 5. "Will this scale in our environment?"

**Short answer:** The repo now ships a runnable local benchmark baseline and named workload profiles, but those results are still reference-host evidence rather than customer-environment capacity proof.

**Repo proof now:**
- `make performance-baseline`
- `make performance-baseline PERFORMANCE_PROFILE=interactive`
- `make performance-baseline PERFORMANCE_PROFILE=burst`
- `make performance-baseline PERFORMANCE_PROFILE=sustained`
- Performance interpretation rules and thresholds are documented for the host-first reference baseline.

**Customer dependency:**
- load profile
- concurrency targets
- provider latency
- infrastructure sizing and database topology

**Do not claim:**
- fixed RPS or latency numbers as customer-grade proof from this repo alone
- that a healthy reference-host profile run is enough to skip target-environment validation

Primary references:
- `docs/PERFORMANCE_BASELINE.md`
- `docs/DEPLOYMENT.md`
- `docs/demo/q_and_a.md`

---

## 6. "Can finance actually use this for chargeback?"

**Short answer:** Yes for attribution patterns and reporting workflow design, but not every automation path is productized in this public snapshot.

**Repo proof now:**
- spend attribution model
- cost-center alias convention
- monthly reporting workflow and Helm packaging for chargeback artifacts

**Customer dependency:**
- ERP/journal-entry integration
- seat-roster reconciliation for subscription billing
- finance signoff on allocation rules

**Do not claim:**
- turnkey ERP integration from this repo alone

Primary references:
- `docs/policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md`
- `./scripts/acpctl.sh chargeback report --format all`

---

## 7. "What is the minimum credible pilot?"

**Short answer:** A pilot must prove routed enforcement, attribution, SIEM evidence, and customer-owned bypass controls in writing.

**Repo proof now:**
- A single pilot-package source of truth exists.
- The minimum validation command set is defined.
- Deliverables and exit criteria are explicit.
- Customer-owned validation and closeout decision patterns are documented.

Primary reference:
- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_EXECUTION_MODEL.md`
- `docs/PILOT_SPONSOR_ONE_PAGER.md`
- `docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md`
- `docs/PILOT_CLOSEOUT_EXAMPLES.md`
