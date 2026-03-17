---
marp: true
theme: default
class: invert
paginate: true
size: 16:9
backgroundColor: '#111111'
color: '#ffffff'
style: |
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');
  
  :root {
    --acp-orange: #f26522;
    --acp-black: #000000;
    --acp-white: #ffffff;
    --acp-gray-900: #111111;
    --acp-gray-800: #222222;
    --acp-gray-700: #444444;
  }
  
  section {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
    background: #111111;
    padding: 35px 45px;
  }
  
  h1 {
    color: #f26522;
    font-size: 1.9em;
    font-weight: 700;
    margin-bottom: 20px;
    margin-top: 0;
    letter-spacing: -0.02em;
  }
  
  h2 {
    color: #f26522;
    font-size: 1.3em;
    font-weight: 600;
    margin-top: 20px;
    margin-bottom: 12px;
  }
  
  p {
    margin-bottom: 12px;
    line-height: 1.5;
  }
  
  strong {
    color: #f26522;
    font-weight: 600;
  }
  
  ul {
    list-style: none;
    padding-left: 0;
    margin: 12px 0;
  }
  
  ul li {
    position: relative;
    padding-left: 20px;
    margin-bottom: 8px;
    line-height: 1.4;
  }
  
  ul li::before {
    content: "▸";
    color: #f26522;
    position: absolute;
    left: 0;
    font-weight: bold;
  }
  
  table {
    width: 100%;
    border-collapse: collapse;
    margin: 15px 0;
    font-size: 0.85em;
  }
  
  th {
    background: #f26522;
    color: #ffffff;
    padding: 10px 12px;
    text-align: left;
    font-weight: 600;
  }
  
  td {
    padding: 8px 12px;
    border-bottom: 1px solid #444444;
    color: #eeeeee;
  }
  
  tr:nth-child(even) {
    background: rgba(255, 255, 255, 0.03);
  }
  
  blockquote {
    border-left: 3px solid #f26522;
    margin: 15px 0;
    padding: 10px 15px;
    background: rgba(242, 101, 34, 0.1);
    font-style: italic;
    color: #eeeeee;
  }
  
  pre {
    background: #1a1a2e;
    border-radius: 6px;
    padding: 12px 16px;
    margin: 12px 0;
    font-size: 0.78em;
    line-height: 1.4;
  }
  
  code {
    font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
    color: #e0e0e0;
  }
  
  footer {
    color: #666666;
    font-size: 0.7em;
  }
  
  /* Lead slide (title slides) */
  section.lead {
    text-align: center;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  
  section.lead h1 {
    font-size: 2.8em;
    margin-bottom: 20px;
  }
  
  section.lead h2 {
    border: none;
    font-size: 1.5em;
    color: #ffffff;
    margin: 10px 0;
  }
  
  /* Two column layout */
  .two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 40px;
  }
  
  /* Architecture diagram styles */
  .arch-container {
    text-align: center;
    margin-top: 10px;
  }
  
  .arch-box {
    border: 2px solid #f26522;
    padding: 12px 15px;
    border-radius: 6px;
    display: inline-block;
    text-align: left;
  }
  
  .arch-title {
    color: #f26522;
    font-weight: bold;
    text-align: center;
    margin-bottom: 10px;
    font-size: 0.9em;
  }
  
  .arch-row {
    display: flex;
    gap: 10px;
    margin-bottom: 10px;
  }
  
  .arch-col {
    border: 1px solid #666;
    padding: 8px 10px;
    border-radius: 3px;
    flex: 1;
    min-width: 120px;
  }
  
  .arch-col-title {
    color: #f26522;
    font-weight: bold;
    margin-bottom: 4px;
    font-size: 0.85em;
  }
  
  .arch-col-content {
    font-size: 0.8em;
    line-height: 1.3;
  }
  
  .arch-arrow {
    text-align: center;
    margin: 6px 0;
    font-size: 0.85em;
  }
  
  .arch-pipeline {
    border: 2px solid #0089cf;
    padding: 8px 12px;
    border-radius: 3px;
    text-align: center;
    margin: 0 auto;
    max-width: 320px;
  }
  
  .arch-pipeline-title {
    color: #0089cf;
    font-weight: bold;
    margin-bottom: 2px;
    font-size: 0.85em;
  }
  
  .arch-pipeline-content {
    font-size: 0.8em;
  }
  
  .arch-siem {
    border: 1px solid #666;
    padding: 5px 12px;
    border-radius: 3px;
    text-align: center;
    max-width: 160px;
    margin: 0 auto;
    font-size: 0.8em;
  }
  
  /* Highlight box */
  .highlight-box {
    margin-top: 20px;
    padding: 12px 15px;
    background: rgba(242, 101, 34, 0.1);
    border-left: 3px solid #f26522;
    border-radius: 0 6px 6px 0;
  }
  
  /* Status text */
  .status-pass {
    color: #22c55e;
  }
  
  .status-detect {
    color: #0089cf;
  }
  
  /* Compact list */
  .compact-list li {
    margin-bottom: 6px;
  }
---

<!-- _class: lead -->
<!-- _paginate: false -->
<!-- _backgroundImage: url('./assets/01-hero-cover.png') -->
<!-- _backgroundSize: cover -->
<!-- _backgroundPosition: center -->

# Enterprise AI Control Plane

## Governance · Approved-Only Access · Misuse Detection

**AI Control Plane Project**

February 2026

---

<!-- _backgroundImage: url('./assets/08-problem-statement.png') -->
<!-- _backgroundSize: cover -->
<!-- _backgroundPosition: center -->

# The Problem in 60 Seconds

Organizations adopt AI through **two channels** requiring different controls:

**API Usage:** Apps/agents calling provider APIs with keys
- Risk: Uncontrolled spend, shadow API keys, no audit trail

**Subscription/SaaS:** Developer tools via OAuth & workspaces
- Risk: Unmanaged seats, data exfiltration, compliance gaps

> **The Coverage Gap:** A strategy covering only API keys is incomplete. A strategy covering only SaaS is incomplete. Real-world programs must address **both channels simultaneously**.

---

# Unified AI Control Plane Solution

**This project delivers a unified governance platform:**

| Control Plane | Function | Coverage |
|--------------|----------|----------|
| **API Gateway** | Enforcement chokepoint | All API-key traffic |
| **SaaS Governance** | Workspace + compliance exports | All subscription usage |
| **Evidence Pipeline** | Normalized SIEM feed | Both channels |

> "This is standard security engineering applied to AI surfaces, with enforceable controls and auditable evidence."

---

<!-- _backgroundImage: url('./assets/03-two-track-governance.png') -->
<!-- _backgroundSize: contain -->
<!-- _backgroundPosition: center -->

# Route-Based Governance Model

<div class="two-col">

<div>

**Gateway-Routed Mode (API-Key + Subscription-Backed CLI)**
- Model allowlists, budget enforcement, rate limiting
- Real-time blocking at gateway
- Tools: Codex, Claude, OpenCode, Cursor

</div>

<div>

**Direct Subscription / Bypass (Detect + Respond)**
- OTEL telemetry, compliance exports
- Cross-source correlation, governance reporting
- Tools: direct OAuth flows and unmanaged web clients

</div>

</div>

**Unified Pipeline:** All evidence feeds into your SIEM regardless of source

---

<!-- _backgroundImage: url('./assets/02-architecture-diagram.png') -->
<!-- _backgroundSize: contain -->
<!-- _backgroundPosition: center -->

# Target Architecture

<div class="arch-container">

<div class="arch-box">

<div class="arch-title">UNIFIED AI CONTROL PLANE</div>

<div class="arch-row">
<div class="arch-col">
<div class="arch-col-title">API Control</div>
<div class="arch-col-content">• AI Gateway<br>• Allowlists<br>• Budgets</div>
</div>
<div class="arch-col">
<div class="arch-col-title">SaaS Control</div>
<div class="arch-col-content">• Workspaces<br>• Audit logs<br>• Compliance</div>
</div>
<div class="arch-col">
<div class="arch-col-title">Network</div>
<div class="arch-col-content">• Egress control<br>• SWG/CASB<br>• MDM</div>
</div>
</div>

<div class="arch-arrow">▼</div>

<div class="arch-pipeline">
<div class="arch-pipeline-title">CENTRAL EVIDENCE PIPELINE</div>
<div class="arch-pipeline-content">Principal · Model · Cost · Policy</div>
</div>

<div class="arch-arrow">▼</div>

<div class="arch-siem">[ YOUR SIEM ]</div>

</div>

</div>

---

# Demonstrable Evidence

**Commands:**

```bash
make onboard-claude
make demo-scenario SCENARIO=1
make validate-detections && make release-bundle && make release-bundle-verify
```

**Artifacts:**

| Artifact | File |
|----------|------|
| Gateway Events | `gateway_events.jsonl` |
| Telemetry | `telemetry.jsonl` |
| Compliance | `compliance_events.jsonl` |
| Unified Evidence | `evidence.jsonl` |

---

# Honest Capabilities: Enforce vs. Detect

| Control Surface | Gateway-Routed Mode | Direct Subscription / Bypass |
|----------------|---------------------|------------------------------|
| **Gateway** | Virtual keys + routed OAuth headers | Not in path |
| **SaaS/Workspace** | Optional augmentation | Vendor-managed primary |
| **OTEL Telemetry** | Optional dual telemetry | Primary telemetry path |
| **Compliance Exports** | Optional augmentation | Enterprise tier evidence |
| **Network Egress** | Prevents bypass + enforces routing | Required to constrain bypass |

**Gateway-Routed:** <span class="status-pass">Full enforcement</span> (real-time blocking)

**Direct Bypass:** <span class="status-detect">Detect + respond</span> (no inline blocking)

---

<!-- _backgroundImage: url('./assets/04-service-offerings.png') -->
<!-- _backgroundSize: contain -->
<!-- _backgroundPosition: center -->

# Service Offerings

| Offering | Duration | Primary Outcome |
|----------|----------|-----------------|
| **AI Usage & Exposure Assessment** | 2–4 weeks | Inventory, risk analysis, target architecture |
| **AI Control Plane Implementation** | 4–10 weeks | Operational gateway, SIEM integration, pilot |
| **Managed AI Security Operations** | Ongoing | Continuous monitoring, IR, governance reporting |
| **Vendor Workspace Governance** | 2–6 weeks | Workspace config, RBAC, compliance exports |

**Engagement Models:** Fixed Scope · Time & Materials · Retainer · Outcome-Based

---

# Delivery Model: Customer-Controlled Infrastructure

| Aspect | Customer Ownership | Project Responsibility |
|--------|-------------------|-------------------|
| **Infrastructure** | Customer VPC / On-Premises | Project deploys and configures |
| **Telemetry Storage** | Customer SIEM | Project integrates and tunes |
| **Data Residency** | Customer-defined | Project respects constraints |
| **Key Ownership** | Customer holds exclusively | Project manages lifecycle |
| **Incident Response** | Customer-led | Project playbooks and support |

**Core principle:** Project delivers expertise, not hosting. All data stays in customer-controlled environments.

---

<!-- _backgroundImage: url('./assets/06-financial-governance.png') -->
<!-- _backgroundSize: contain -->
<!-- _backgroundPosition: center -->

# Financial Governance

<div class="two-col">

<div>

**API-Key Billing (Usage-Based)**
- Billed by providers on token consumption
- Chargeback via user/team/service keys

</div>

<div>

**Subscription Billing (Seat-Based)**
- Billed per seat (ChatGPT, Claude, Copilot)
- Chargeback via seat assignment + usage logs

</div>

</div>

<div class="highlight-box">

**Attribution Model:** `key_alias` convention enables cost-center allocation and monthly chargeback

</div>

---

<!-- _backgroundImage: url('./assets/05-implementation-roadmap.png') -->
<!-- _backgroundSize: contain -->
<!-- _backgroundPosition: center -->

# Implementation Roadmap

| Phase | Timeline | Focus |
|-------|----------|-------|
| **0: Standards** | Week 1-2 | Define approved paths, retention rules |
| **1: Lab Validation** | Week 3-6 | Prove routing, logging, governance |
| **2: Internal Rollout** | Week 7-10 | Internal validation deployment |
| **3: Customer Pilots** | Week 11+ | Production pilots |

---

# Readiness Status: GO (Evidence Snapshot)

| Metric | Status |
|--------|--------|
| **Gates Passing** | 8/8 |
| **Open Blockers** | 0 |
| **Evidence Complete** | 100% |
| **Decision** | **GO** |

**Latest published full certification snapshot (readiness-20260219T155143Z, 2026-02-19 UTC):** Local CI · Production CI · Security Validation · Release Bundle · Clean-Host Deployability · Evidence Completeness · Independent Review · Backlog Triage. Refresh before customer reuse.

---

# Strategic Decisions Executing

The AI Control Plane strategy is built on four decisive strategic commitments:

1. **Route-Based Governance** — Full enforcement on gateway-routed traffic; detect and respond on bypass paths. No compromise on coverage.

2. **LiteLLM Foundation** — Vendor-neutral gateway architecture prevents lock-in and enables multi-provider agility.

3. **Productized Service Line** — Four clear offerings from assessment to managed operations. Not custom consulting—repeatable delivery.

4. **Internal Validation First** — The reference implementation is validated on operator-owned infrastructure before customer delivery.

---

<!-- _class: lead -->
<!-- _paginate: false -->
<!-- _backgroundImage: url('./assets/07-closing-visual.png') -->
<!-- _backgroundSize: cover -->
<!-- _backgroundPosition: center -->

# Ready to Build the Future

## of AI Security

**Standard security engineering applied to AI surfaces**

---

<!-- _class: lead -->
<!-- _paginate: false -->

AI Control Plane Project

**Documentation:** Strategy · Service Catalog · Readiness Tracker
