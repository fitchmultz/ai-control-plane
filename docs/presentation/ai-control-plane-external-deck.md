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
---

# Enterprise AI Control Plane

**Unified governance for AI API traffic and browser-based AI SaaS use**

- Visibility across routed and non-routed usage
- Enforceable controls where traffic is governed
- Detective monitoring and operational response for bypass paths

---

# The AI Governance Gap

Organizations are adopting AI through two channels:

1. **Direct API use**
2. **Browser-based AI SaaS use**

Without a control plane, teams lose:

- Central visibility
- Policy consistency
- Cost accountability
- Reliable audit evidence

---

# The Solution

- Central policy enforcement for routed traffic
- Monitoring and response workflows for non-routed and bypass activity
- Shared operational evidence for security and leadership
- A practical rollout path without stopping adoption

---

# How It Works

## Architecture overview

- Users and applications access approved AI services through the control plane
- Gateway controls apply to routed API traffic
- SaaS usage is monitored through detective controls and response procedures
- Logs, policy events, and readiness evidence support operations and review

---

# Route-Based Governance

## Control model

**Enforce**
- Approved routes
- Access policy
- Budget and rate controls
- Logging and evidence capture

**Detect and respond**
- Non-routed AI SaaS usage
- Bypass attempts
- Unapproved destinations
- Drift from approved operating patterns

---

# Service Offerings

| Offering | Primary Outcome |
|---|---|
| AI Usage Exposure Assessment | Establish visibility and baseline risk |
| Vendor Workspace Governance | Control SaaS AI workspace usage |
| AI Control Plane Implementation | Deploy enforceable routed controls |
| Managed AI Security Operations | Sustain detections, reviews, and reporting |

---

# Financial Governance

The platform supports:

- Usage attribution by tenant, team, and workload
- Budget guardrails
- Showback and chargeback reporting
- Better decisions on model and vendor usage

**Result:** governance improves without losing cost transparency.

---

# Why This Project?

## Differentiation

- Built around real operational controls, not policy slides
- Clear distinction between what can be enforced and what must be detected
- Customer-ready services with delivery structure
- Evidence-backed operating model instead of aspirational claims

---

# Implementation Approach

## Four-phase rollout

1. **Assess** - identify AI usage and governance gaps
2. **Control** - deploy routed enforcement and policy foundations
3. **Operationalize** - add detections, reviews, and reporting
4. **Scale** - expand coverage, service ownership, and accountability

---

# Get Started

Choose the path that matches your current maturity:

1. **Assessment-first** - understand current exposure
2. **Pilot implementation** - stand up core routed governance
3. **Managed operations** - continue with ongoing governance support

---

# Closing

## A practical governance model for enterprise AI

- Start with real visibility
- Enforce where architecture allows
- Detect and respond where direct enforcement is not possible
- Build measurable operating evidence over time

---

# Contact

**AI Control Plane Project**

Recommended next steps:

- Review current AI usage exposure
- Select an initial pilot scope
- Align stakeholders on governance and operating model

See also:

- `docs/SERVICE_OFFERINGS.md`
- `docs/ENTERPRISE_STRATEGY.md`
- `docs/presentation/EXECUTIVE_ONE_PAGER.md`
