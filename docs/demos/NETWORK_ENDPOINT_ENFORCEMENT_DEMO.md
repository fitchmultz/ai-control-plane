# Network and Endpoint Enforcement Demonstration

**Executive Summary:** This document demonstrates the third pillar of AI governance: **Network and Endpoint Enforcement**. It explains why egress controls are essential for definitive AI traffic coverage, documents the bypass threat model, and provides a repeatable walkthrough for demonstrating bypass prevention patterns.

---

## What This Doc Is Responsible For

- Demonstrating the **bypass threat model**: what shadow AI usage looks like in practice.
- Documenting **network controls** (default-deny egress) that prevent direct API access.
- Documenting **SWG/CASB controls** for browser-based AI usage governance.
- Documenting **endpoint controls** (MDM, tool configuration enforcement) that prevent bypass.
- Providing a clear explanation of what the local demo can/cannot prove vs. the AWS lab validation.

## What This Doc Does NOT Cover

- Production-grade SWG/CASB configuration (vendor-specific; requires customer infrastructure).
- MDM deployment procedures (platform-specific; requires customer endpoint management).
- Real-time egress blocking in the local demo environment (requires enterprise network infrastructure).

## Invariants / Assumptions

- **Local demo**: Demonstrates the bypass shape and governance requirements; cannot prove blocking without enterprise egress controls.
- **AWS lab**: Can validate default-deny egress and SWG/CASB enforcement.
- Network controls require customer-managed infrastructure (firewall, SWG, CASB, MDM).
- Project can design, configure, and operate these controls as part of service offerings.

## Shared Responsibility (Non-Negotiable Boundary)

| Domain | Project Role | Customer Role |
|---|---|---|
| Gateway policy design | Define and implement routed-path controls | Approve and operate within enterprise policy model |
| Egress enforcement | Provide deny-list/allow-list design patterns and validation checks | Implement firewall/SWG/CASB rules in production network |
| Endpoint posture | Provide managed tool configuration guidance | Enforce via MDM/endpoint controls across managed devices |
| Monitoring and response | Provide normalized evidence + detection patterns | Operate SIEM workflows and incident response authority |

---

## Overview

The Network and Endpoint Enforcement demonstration showcases the **third pillar** of the AI Control Plane architecture. While the API Control Plane and SaaS Control Plane provide governance for approved paths, **Network and Endpoint Enforcement prevents (or materially reduces) bypass** through shadow IT AI usage.

| Governance Pillar | Enforcement Capability | Bypass Prevention |
|-------------------|------------------------|-------------------|
| **API Control Plane** | Gateway blocks non-approved models | Direct API calls with personal keys |
| **SaaS Control Plane** | Workspace policies, compliance exports | Personal subscriptions, unapproved tools |
| **Network & Endpoint** | Egress blocking, SWG policies, MDM controls | Direct endpoint access, unauthorized tools |

### Why Egress Control Is Non-Negotiable

> *"There is no 'catch all AI traffic' without egress control."*

If endpoints can reach the open internet freely, it is **impossible to guarantee** you see or block all AI usage. The control objective becomes probabilistic rather than deterministic.

**Definitive coverage requires:**
- Default-deny posture on AI API egress
- Only the approved gateway may reach model provider endpoints
- SWG/CASB governance for browser-based AI usage
- Endpoint management to enforce tool configurations

---

## Prerequisites

1. Docker and Docker Compose installed
2. AI Control Plane repository cloned
3. Basic environment configured (`make install`)
4. For egress validation: AWS lab or enterprise network environment

---

## Threat Model: What "Bypass" Looks Like

### Bypass Vector 1: API-Key Bypass (Direct Provider Access)

**Scenario**: Developer uses personal OpenAI API key directly instead of the corporate gateway.

```bash
# Bypass: Direct provider access (unapproved)
export OPENAI_API_KEY="sk-personal-key-from-developer"
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

**What the enterprise sees**: Nothing. No logs, no attribution, no policy enforcement.

**Controls needed**:
- Network egress: Block direct access to `api.openai.com` except from gateway
- Endpoint DLP: Detect API keys in clipboard/code
- Gateway incentive: Make the approved path easier than the bypass

### Bypass Vector 2: Subscription/SaaS Bypass (OAuth Direct)

**Scenario**: Developer uses Claude Code or Codex with personal subscription instead of enterprise workspace.

```bash
# Bypass: Subscription mode without gateway routing
claude --login  # Personal account, not enterprise
claude "Help me analyze this sensitive data"
```

**What the enterprise sees**: Nothing at the gateway (traffic goes direct to vendor).

**Controls needed**:
- SWG/CASB: Block personal AI tool categories
- Endpoint management: Enforce tool configuration
- Vendor workspace policies: Force enterprise workspace enrollment

### Bypass Vector 3: Unauthorized Tooling (Shadow IT)

**Scenario**: Developer downloads a new AI CLI tool not approved by IT.

```bash
# Bypass: New/unapproved AI tool
curl -sSL https://new-ai-tool.io/install.sh | bash
new-ai-tool --api-key personal-key
```

**Controls needed**:
- Application control/allowlisting
- Network egress monitoring
- SWG blocking of unapproved AI SaaS categories

---

## Control Objectives

### Approved vs. Unapproved Paths

| Path | Approved | Unapproved |
|------|----------|------------|
| **API Access** | Via corporate gateway only | Direct to provider endpoints |
| **Browser Usage** | Enterprise workspace tenants | Personal/free tier usage |
| **Tool Configuration** | MDM-enforced, gateway-pointed | User-configured, direct access |
| **Authentication** | Corporate SSO/identity | Personal accounts |

### Preventive vs. Detective Controls

| Control Type | Implementation | Where It Works |
|--------------|----------------|----------------|
| **Preventive** | Egress blocking, SWG deny policies | AWS lab, enterprise network |
| **Detective** | Egress logging, DLP alerts, anomaly detection | Local + AWS lab |
| **Corrective** | Key rotation, access revocation, config enforcement | All environments |

---

## Network Controls: Default-Deny Egress

### VPC Security Group / Route Table Patterns (AWS Lab)

In the AWS lab environment, network controls are implemented as follows:

```
┌─────────────────────────────────────────────────────────────────┐
│                         AWS VPC                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Private Subnets (Workloads)                            │   │
│  │  • Security Group: sg-workloads                         │   │
│  │    - Outbound: ONLY to sg-gateway (port 4000)          │   │
│  │    - NO direct internet access                         │   │
│  └──────────────────────┬──────────────────────────────────┘   │
│                         │                                       │
│                         ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Private Subnet (AI Gateway)                            │   │
│  │  • Security Group: sg-gateway                           │   │
│  │    - Outbound: HTTPS to approved providers ONLY        │   │
│  │    - OpenAI API, Anthropic API (allowlist)             │   │
│  └──────────────────────┬──────────────────────────────────┘   │
│                         │                                       │
│                         ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Network Firewall / NAT Gateway                         │   │
│  │  • Explicit allowlist: only sg-gateway can reach       │   │
│  │    api.openai.com, api.anthropic.com                   │   │
│  │  • Default DENY all other AI provider endpoints        │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### The "Allow Only Gateway" Model

**Key principle**: Only the AI Gateway security group can reach model provider APIs.

| Source | Destination | Port | Action |
|--------|-------------|------|--------|
| sg-gateway | api.openai.com | 443 | ALLOW |
| sg-gateway | api.anthropic.com | 443 | ALLOW |
| sg-workloads | api.openai.com | 443 | DENY |
| sg-workloads | api.anthropic.com | 443 | DENY |
| sg-workloads | sg-gateway | 4000 | ALLOW |
| * | * | * | DENY |

### Avoiding Brittle IP Allowlists

**Do not rely solely on IP allowlists** for model provider endpoints:

- Provider IPs change frequently
- CDN-based endpoints resolve to many IPs
- IPv6 introduces additional complexity

**Preferred approaches**:
- Security group references (AWS-native)
- DNS-based firewall rules (with wildcard support)
- Category-based SWG policies for SaaS/web traffic

---

## SWG/CASB Controls (Browser)

### Enterprise vs. Personal Tenant Enforcement

| Policy | Enterprise Tenant | Personal Tenant |
|--------|-------------------|-----------------|
| ChatGPT | `chat.openai.com/?workspace=*` | `chat.openai.com` (no workspace) |
| Claude | `claude.ai/new?workspace=*` | `claude.ai/new` |
| Action | ALLOW | BLOCK |

### Unapproved AI SaaS Categories

SWG/CASB policies should block or monitor:

- Generative AI (unapproved providers)
- AI code assistants (unapproved tools)
- AI image generation
- AI chatbots (consumer)

### Evidence Sources

SWG logs provide:
- URL/category visited
- User identity (if authenticated)
- Timestamp and duration
- Data volume (upload/download)
- Policy action (allowed/blocked)

These logs should feed into the same SIEM pipeline as gateway logs.

---

## Endpoint Controls

### MDM-Managed Configuration for Tools

**Codex Configuration Enforcement** (via MDM payload):

```json
{
  "forced_login_method": "chatgpt_sso",
  "forced_chatgpt_workspace_id": "org-enterprise123",
  "allowed_models": ["gpt-4", "gpt-5.2"]
}
```

**Claude Code Configuration Enforcement**:

```json
{
  "anthropic_base_url": "https://gateway.company.com",
  "require_gateway": true,
  "allowed_workspaces": ["company-enterprise"]
}
```

### "Force Gateway" Posture

Endpoint controls should:

1. **Set environment variables** via MDM/shell profiles:
   - `OPENAI_BASE_URL=https://gateway.company.com`
   - `ANTHROPIC_BASE_URL=https://gateway.company.com`

2. **Prevent local overrides** where possible:
   - Read-only config files
   - Periodic validation checks
   - Detection rules for non-gateway traffic

3. **Detect unauthorized tooling**:
   - Application inventory/allowlisting
   - Process monitoring for known AI tools
   - Behavioral detection (anomalous API calls)

---

## Local Demo vs. AWS Lab Validation

### Local Demo (What We Can Show)

**Capabilities**:
- Demonstrate the bypass threat model
- Show configuration differences between approved/unapproved paths
- Document what evidence is/isn't produced without egress controls
- Explain the governance requirements

**Limitations**:
- Cannot prove egress blocking (no corporate firewall)
- Cannot enforce MDM policies (personal device)
- Cannot validate SWG/CASB (no enterprise web gateway)

### AWS Lab (What We Can Prove)

**Validation scenarios**:

1. **Default-Deny Egress Test**:

   ```bash
   # From non-gateway host: Should FAIL (blocked)
   curl https://api.openai.com/v1/models
   
   # From gateway host: Should SUCCEED (allowed)
   curl https://api.openai.com/v1/models
   ```

2. **Gateway-Only Access Test**:

   ```bash
   # Non-gateway host can reach gateway
   curl http://gateway-host:4000/health
   
   # Non-gateway host cannot reach OpenAI directly
   curl https://api.openai.com/v1/models  # BLOCKED
   ```

3. **SWG Policy Validation**:
   - Personal ChatGPT tenant: BLOCKED
   - Enterprise ChatGPT workspace: ALLOWED

---

## Demo Script

### Run the Network/Endpoint Enforcement Demo

```bash
# Local mode (demonstrates threat model and controls)
make demo-scenario SCENARIO=7

# Use environment-specific configuration in demo/.env when validating AWS lab behavior.
```

### Expected Output (Local Mode)

```
=== Scenario 7: Network and Endpoint Enforcement ===

Environment
  Gateway: http://127.0.0.1:4000
  Database: postgres:5432 (internal; not published to host by default)
  Deployment: Local Mode

Step 1: Threat Model - Bypass Vectors

  Bypass Vector 1: API-Key Bypass (Direct Provider Access)
    Risk: Developer uses personal API key, bypasses gateway
    Impact: No logs, no attribution, no policy enforcement
    Control: Default-deny egress to AI provider endpoints

  Bypass Vector 2: Subscription/SaaS Bypass (OAuth Direct)
    Risk: Developer uses personal subscription
    Impact: Traffic goes direct to vendor, no gateway visibility
    Control: SWG/CASB block personal tenants

  Bypass Vector 3: Unauthorized Tooling (Shadow IT)
    Risk: Developer uses unapproved AI tool
    Impact: Unknown tool, unknown data handling
    Control: Application allowlisting, egress monitoring

Step 2: Network Controls (Default-Deny Egress)

  AWS Lab Pattern:
    • Only sg-gateway may reach api.openai.com
    • Only sg-gateway may reach api.anthropic.com
    • All other hosts: DENY

  Local Demo Limitation:
    ⚠ This environment cannot prove blocking without enterprise egress controls

Step 3: SWG/CASB Controls

  Enterprise Tenant: ALLOW
    • chat.openai.com/?workspace=org-enterprise
    • claude.ai/new?workspace=company-enterprise

  Personal Tenant: BLOCK
    • chat.openai.com (no workspace)
    • claude.ai/new (personal account)

Step 4: Endpoint Controls

  MDM-Enforced Configuration:
    • OPENAI_BASE_URL=https://gateway.company.com
    • ANTHROPIC_BASE_URL=https://gateway.company.com
    • Codex: forced_login_method=chatgpt_sso

Step 5: Evidence Checklist

  What to screenshot in AWS lab:
    • Security group rules showing default-deny
    • Failed connection attempts from non-gateway host
    • Successful connections from gateway host
    • SWG logs showing blocked personal tenant access

  What this local demo demonstrates:
    ✓ Bypass threat model documented
    ✓ Control objectives defined
    ✓ AWS lab validation steps outlined
    ✓ Evidence sources identified

=== Scenario 7: PASSED ===
```

---

## Evidence Checklist (What to Screenshot/Export)

### AWS Lab Evidence

| Artifact | How to Capture | Evidence Of |
|----------|----------------|-------------|
| Security group rules | AWS Console screenshot | Default-deny egress configuration |
| Failed egress test | Terminal output | Non-gateway host cannot reach provider |
| Successful gateway test | Terminal output | Gateway can reach provider |
| SWG block logs | SIEM query output | Personal tenant access blocked |
| MDM config profile | MDM console screenshot | Tool configuration enforced |

### Local Demo Evidence

| Artifact | How to Capture | Evidence Of |
|----------|----------------|-------------|
| Threat model diagram | Terminal output | Comprehensive bypass analysis |
| Control objectives | Terminal output | Preventive + detective controls defined |
| AWS lab validation steps | Terminal output | Path to proving enforcement |

---

## Management Presentation Guide

### Key Talking Points

**1. The Bypass Problem**
> "Most AI governance strategies have a critical gap: they only cover traffic that voluntarily goes through the gateway. If a developer uses a personal API key or personal ChatGPT account, the enterprise has zero visibility."

**2. The Egress Control Solution**
> "Network and endpoint enforcement makes bypass materially harder. By implementing default-deny egress—where only the approved gateway can reach AI provider endpoints—we ensure all AI traffic is observable and governable."

**3. Defense in Depth**
> "This isn't just network firewalls. It's SWG policies blocking personal AI tenants, MDM enforcing tool configurations, and endpoint controls preventing unauthorized tooling. Multiple layers, each adding friction to bypass."

**4. Local vs. Production**
> "In this local demo, we're showing the threat model and control design. Production enforcement still has to be validated in the customer environment with real network controls."

---

## Related Documentation

| Document | Purpose |
|----------|---------|
| [Enterprise AI Control Plane Strategy](../ENTERPRISE_STRATEGY.md) | Strategic overview |
| [GO_TO_MARKET_SCOPE.md](../GO_TO_MARKET_SCOPE.md) | Validated baseline and customer-environment proof boundary |
| [SaaS Subscription Governance Demo](SaaS_SUBSCRIPTION_GOVERNANCE_DEMO.md) | Two-track governance model |
| [OpenCode Tooling Guide](../tooling/OPENCODE.md) | Bypass vector documentation (Codex auth plugin) |
| [DEPLOYMENT.md](../DEPLOYMENT.md) | Network setup and deployment modes |
| [demo/README.md](../../demo/README.md) | Demo environment quick start |

---

## Appendix: Control Verification Commands

### AWS Lab Validation

```bash
# Test 1: Verify gateway can reach provider
ssh gateway-host "curl -s https://api.openai.com/v1/models | head -5"

# Test 2: Verify non-gateway host cannot reach provider
ssh app-host "curl -s https://api.openai.com/v1/models"  # Should fail/block

# Test 3: Verify non-gateway host can reach gateway
ssh app-host "curl -s http://gateway-host:4000/health"  # Should succeed

# Test 4: Check Network Firewall logs
aws logs filter-log-events \
  --log-group-name /aws/network-firewall/alerts \
  --filter-pattern "DROP"
```

### Local Demo (Configuration Validation)

```bash
# Show current tool configuration
echo "OPENAI_BASE_URL: ${OPENAI_BASE_URL:-not set}"
echo "ANTHROPIC_BASE_URL: ${ANTHROPIC_BASE_URL:-not set}"

# Demonstrate what bypass looks like
echo "Bypass attempt would use: https://api.openai.com/v1/chat/completions"
echo "Approved path uses: ${OPENAI_BASE_URL:-http://127.0.0.1:4000}/v1/chat/completions"
```

---

*This demonstration is part of the AI Control Plane service offerings. For detailed service descriptions, deliverables, and SOW templates, see [Service Offerings](../SERVICE_OFFERINGS.md) and [SOW Templates](../templates/). For implementation assistance, contact Project Maintainer.*
