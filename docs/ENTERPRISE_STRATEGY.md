# Enterprise AI Control Plane Strategy

## Governance, approved-only access, misuse detection, cost control

Date: March 5, 2026

---

## Executive summary

Enterprise AI adoption is splitting into two real usage channels:

1. API usage, meaning apps, agents, and automations calling model provider APIs with keys.
2. Subscription and SaaS usage, meaning developer tools and assistants authenticated with user accounts inside vendor workspaces.

If we only govern API keys, we miss a growing share of real usage. If we only govern SaaS, we cannot enforce policies for automated workloads where the highest risk lives.

This strategy defines an AI Control Plane built on three foundations:

* **Enforcement gateway built on LiteLLM** as the standard choke point for API and tool traffic that can be routed. This is where we do approved model allowlists, policy guardrails, budgets, rate limits, attribution, and audit logging.
* **Workspace governance layer** for subscription and SaaS usage using identity, vendor admin controls, and enterprise audit sources. Where enforcement is not technically possible, we move to detective controls plus network constraints.
* **Unified evidence pipeline into the customer SIEM** with normalized events, correlation, detections, and operator runbooks.

This is standard security engineering applied to AI: choke points, identity, allowlists, logging, detection, and response.

Validated scope in this repository: local host-first reference implementation, typed operator workflows, evidence-generation patterns, and an incubating design-only package for future multi-tenant isolation and billing. Cloud-specific enforcement claims and generalized managed-service claims still require additional environment-specific validation before external commitment.

## Non-Negotiable Control Truth

The strategy only works if we state the control truth plainly:

- routed gateway traffic can be enforced
- direct SaaS and browser usage often cannot be enforced by this repo alone
- customer-owned network, endpoint, workspace, and IdP controls determine whether bypass is prevented or merely detected
- evidence without ownership is not governance

Any sales, pilot, or service narrative that blurs those lines weakens the offering.

---

## The problem we are actually solving

### First principles

Any control system needs three things:

1. A place where traffic must pass, so you can enforce.
2. Identity on every request, so you can attribute.
3. Evidence, so you can prove what happened and respond fast.

AI breaks the old assumptions because it is not one system. It is many systems, many vendors, many tools, and two completely different access patterns.

### Why current enterprise governance fails

Most teams try one of these and get burned:

* They manage API keys and think they are done. Meanwhile developers use ChatGPT web, Claude web, IDE assistants, and vendor workspaces with OAuth. This is the majority of usage in many orgs.
* They rely on vendor workspaces only. That gives governance, but it does not give strong, real-time enforcement for app and agent workloads.
* They rely on policy docs and training without technical enforcement or verification controls.

The AI Control Plane strategy provides enforceable controls with auditable evidence.

---

## Strategy statement

This project focuses on a repeatable pattern for **approved AI usage paths** that are:

* Easy to adopt
* Hard to bypass
* Easy to audit

This pattern must cover both:

* API-key workloads and routed tool traffic, enforced through a gateway
* Subscription and SaaS usage, governed through workspaces, identity, and audit evidence

Everything else is either constrained by network controls or treated as unapproved with detection and response.

## Control Boundary In One Table

Use this table whenever the buyer asks what is actually controlled.

| Usage path | What we can do | Control type | Who must own the missing controls |
|---|---|---|---|
| Routed API and agent traffic through LiteLLM | allowlists, budgets, attribution, guardrails, logging, detections | Enforce + detect | Delivery team for app path; customer for surrounding infrastructure |
| Managed browser/chat path routed through sanctioned entrypoint | approved entrypoint, constrained model exposure, identity-aware evidence, detections | Mixed: enforce on routed path, detect on surrounding usage | Customer workspace, browser, and endpoint owners |
| Direct vendor SaaS/browser usage outside sanctioned path | detect, correlate, escalate | Detect only | Customer network, SWG/CASB, endpoint, and workspace owners |
| Bring-your-own-key or unmanaged tool usage | detect partially, investigate, constrain with policy and network controls where available | Mostly detect | Customer network, endpoint, IAM, and procurement owners |

The offer becomes stronger when this table is used early, not hidden late.

---

## The core design choice: LiteLLM as the gateway standard

LiteLLM is the core of the solution because it gives us one unified gateway surface across many model providers. That matters for a simple reason:

* Enterprises do not want to bet their control plane on a single model vendor.
* They want policy once and coverage everywhere.

At a high level, LiteLLM gives us:

* A single endpoint for routed traffic
* Provider abstraction and routing across OpenAI, Anthropic, Google, and others
* Centralized logging and attribution
* A place to enforce allowlists, budgets, and guardrails consistently

Think of LiteLLM like an API firewall plus a toll booth. If traffic goes through it, we can enforce and meter it. If it does not, we need different controls.

---

## Target capabilities

### 1. Gateway enforcement and observability

This is the strong control surface. When traffic is routed through the LiteLLM gateway, we can deliver:

* Approved model and provider allowlists
* Request policy guardrails and real-time blocking
* Per-user and per-service identity using virtual keys
* Budgets, rate limits, and usage throttles
* Centralized audit logs and spend attribution
* Consistent telemetry into the evidence pipeline

This is the control surface where approved-only access is enforceable.

### 2. Workspace governance for subscription and SaaS

This is governance plus evidence, not always enforcement. It includes:

* Vendor workspace configuration and identity controls
* RBAC and least-privilege access patterns
* Audit trail exports where the vendor supports them
* Endpoint posture and egress constraints to reduce bypass
* A managed internal chat experience as the default browser path, if needed

Operational constraint: unmanaged vendor web UIs cannot be forced through LiteLLM. Constrain these paths with network controls, or treat them as unapproved and detect usage.

### 3. Evidence pipeline and detections

The evidence pipeline is the glue. It turns AI activity into security facts:

* Normalized event schema across gateway logs, tool telemetry, and vendor exports
* Correlation by user, service, project, and cost center
* SIEM detections with runbooks
* Executive reporting and audit readiness packages

This is how we move from “we think” to “we can prove.”

### 4. Financial governance

AI spend is a security problem and a finance problem. The control plane must support:

* Showback reporting by team and cost center
* Chargeback, if the customer wants it
* Clear separation of usage cost vs billable internal allocation
* Budget policy tied to identities and projects
* Monthly governance workflow with exceptions and approvals

If we do not control spend, someone will shut the program down.

---

## Architecture overview

### The conceptual system

All AI activity routes into one of two planes, then feeds one evidence pipeline.

```
People and workloads
  |
  +-> Routed tools and apps -> LiteLLM gateway -> Model providers
  |                          |               |
  |                          +-> Policy       +-> Usage data
  |                          +-> Logs
  |
  +-> Vendor workspaces and web tools -> Workspace governance -> Audit evidence
                                 |
                                 +-> Identity controls
                                 +-> Exports where available
                                 +-> Network constraints where required

Both feed
  -> Unified evidence pipeline -> Customer SIEM -> Detections, reporting, runbooks
```

### Control surfaces: enforce vs detect

This is the truth table we should be explicit about.

| Surface                 | Routed through gateway | Direct subscription and bypass               |
| ----------------------- | ---------------------- | -------------------------------------------- |
| Real-time blocking      | Yes                    | No                                           |
| Model allowlists        | Yes                    | Not reliably                                 |
| Budgets and rate limits | Yes                    | Not reliably                                 |
| Spend attribution       | Strong                 | Partial, vendor-dependent                    |
| Audit evidence          | Strong                 | Vendor-dependent                             |
| Best control type       | Enforce and observe    | Detect and respond, plus network constraints |

Analogy: the gateway path is a staffed checkpoint. The bypass path is a camera on the road. Cameras are useful, but they are not a checkpoint.

### Customer-owned controls are not implementation details

The hardest enterprise objections are rarely about the gateway itself. They are about:

- egress ownership
- browser and managed-device ownership
- workspace admin discipline
- SIEM response ownership
- business approval for cost and risk decisions

If those owners are missing, the technical baseline can still be valid, but the enterprise claim cannot.

---

## Where this repository proves the strategy

Use these source-of-truth artifacts when mapping strategy claims to demonstrable repo assets:

| Strategic claim | Primary repo evidence |
| ---------------- | --------------------- |
| Gateway enforcement and operator workflow | `docs/DEPLOYMENT.md`, `demo/README.md`, `docs/tooling/ACPCTL.md` |
| Approved-model, budget, and spend controls | `docs/policy/APPROVED_MODELS.md`, `docs/policy/BUDGETS_AND_RATE_LIMITS.md`, `docs/policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md` |
| Workspace governance and browser-based managed path | `docs/security/ENTERPRISE_AUTH_ARCHITECTURE.md`, `docs/tooling/LIBRECHAT.md` |
| Evidence pipeline, detections, and SIEM workflow | `docs/security/SIEM_INTEGRATION.md`, `docs/security/DETECTION.md`, `demo/config/normalized_schema.yaml`, `demo/config/siem_queries.yaml` |
| Readiness certification and decision criteria | `docs/release/GO_NO_GO.md`, `docs/release/PRESENTATION_READINESS_TRACKER.md`, `docs/release/go_no_go_decision.md` |

This mapping is intentional: strategy claims should trace to inspectable implementation, operational, or validation artifacts.

---

## Policy model: approved paths are explicit

We need a simple customer standard:

### Approved paths

* Routed API and tool traffic through LiteLLM, fully enforced
* Managed browser chat path, governed and logged

### Conditional paths

* Vendor SaaS and workspace usage, allowed only when workspace controls are configured, and evidence is available

### Unapproved paths

* Anything else, including unmanaged web usage outside governance controls

Unapproved means detect, investigate, and block where feasible with network controls.

---

## Operating model

### What this project provides

* Reference architecture and policy model
* Deployment patterns and enforcement configuration for the gateway
* Workspace governance configuration patterns
* Evidence pipeline integration into the customer SIEM
* Detection rules and runbooks
* Executive reporting templates and governance cadence

### What the customer owns

* Their infrastructure and networks
* Their SIEM and retention policies
* Their identity provider and access model
* Their incident response authority and decision rights
* Their vendor contracts and workspace tiers

Core principle: the project provides implementation guidance and repeatability while customer data stays in customer-controlled environments.

### What must never be implied

Do not imply that this project:

- replaces customer network controls
- replaces the customer SIEM or SOC
- removes the need for workspace administration
- can unilaterally block all browser-based AI usage
- turns a partially validated pilot into a production guarantee

---

## Delivery progression

The canonical execution-ordered backlog for this repository lives in [ROADMAP.md](ROADMAP.md).

Use this strategy document for control truth, enterprise positioning, and ownership boundaries. Use the roadmap for outstanding implementation, productization, and procurement-readiness work.

The maturity progression remains:

1. Reference implementation and demo proof
2. Pilot with real teams
3. Production rollout
4. Managed operations and continuous improvement

---

## Risks and constraints with mitigations

### Risk: bypass remains possible without network constraints

* Fact: you cannot force all web traffic through a gateway.
* Mitigation: pair governance with egress controls where the customer needs hard guarantees, otherwise be honest and treat it as detect and respond.

### Risk: vendor audit exports vary by tier and product

* Fact: evidence quality depends on vendor capabilities.
* Mitigation: design connectors to be pluggable, normalize into a common schema, and keep the gateway as the primary enforcement surface.

### Risk: secrets and token safety

* Fact: AI tooling involves API keys and OAuth tokens.
* Mitigation: never log raw authorization secrets, enforce redaction, and treat evidence storage as a security system with strict access and retention.

### Risk: cost blowouts in early adoption

* Fact: AI spend grows faster than people expect.
* Mitigation: budgets, rate limits, default cheaper models for routine tasks, and monthly governance.

---

## Success metrics

If this is working, we can measure it.

Coverage and adoption

* Percent of AI traffic routed through the gateway
* Percent of users using approved paths
* Count of onboarded tools and teams

Control effectiveness

* Number of blocked policy violations
* Time to detect and time to respond for misuse events
* Reduction in unknown or ungoverned AI usage over time

Financial control

* Spend by cost center with explainable attribution
* Budget compliance rate
* Exceptions granted vs exceptions requested

Audit readiness

* Ability to produce an evidence package for a given time window within hours, not weeks

---

## Decisions needed from the CTO

1. Confirm scope: the control plane must cover both API usage and subscription and SaaS usage.
2. Approve the standard stack: LiteLLM gateway as the enforcement choke point, workspace governance for SaaS, unified evidence pipeline into SIEM.
3. Set the data handling stance: metadata-first evidence by default, content ingestion only by explicit opt-in with restricted storage and retention.
4. Approve productization: package this as a repeatable offering with clear deliverables, acceptance checks, and a managed operations path.

---
