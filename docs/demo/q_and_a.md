# AI Control Plane -- Hard-Question Q&A Packet

## Purpose

This document provides evidence-backed answers to challenging questions that may arise during or after the demo presentation. Use these responses to address concerns confidently and transparently.

---

## Q1: Bypass Risk -- "Can't developers just use their own API keys?"

### Short Answer

Yes, but only if customer-owned network controls are in place. We detect bypass through egress monitoring, compliance exports, and detection rules; strong prevention requires deny-by-default egress plus SWG/CASB/MDM enforcement.

### Detailed Answer

This is the right question to ask. The AI Control Plane operates at the choke point -- the network egress and the authentication layer. Here's the layered defense:

**Layer 1: Network Controls (Enforcement)**
- Deploy deny-by-default egress rules via SWG, CASB, or firewall
- Allow only gateway IP for AI provider endpoints (OpenAI, Anthropic, etc.)
- Direct API key usage is blocked at the network level

**Layer 2: Detection (Governance)**
- Compliance exports from vendor workspaces show which users accessed AI
- Cross-reference with gateway logs: if a user is active in workspace exports but not in gateway logs, they're using personal keys
- Detection rule DR-001 flags requests to non-approved models

**Layer 3: Policy (People)**
- Acceptable use policy requiring gateway usage
- Key distribution via secrets management (not hardcoded in repos)
- Regular audits of workspace membership

**What the Demo Proves vs. Customer Infrastructure:**

| Capability | Demo Proves | Requires Customer Infra |
|------------|-------------|------------------------|
| Gateway enforcement | Yes | -- |
| Detection rules | Yes | -- |
| SIEM integration patterns | Yes | -- |
| Egress deny-by-default | -- | SWG/CASB/firewall |
| MDM config enforcement | -- | Endpoint management |

**Boundary statement:** This project provides architecture, controls, and validation guidance; customer network/security teams must implement and operate production egress/SWG/CASB controls.

### Evidence References

- `docs/ENTERPRISE_STRATEGY.md` lines 76-83
- `docs/demos/NETWORK_ENDPOINT_ENFORCEMENT_DEMO.md` (if available)
- `docs/DEPLOYMENT.md` network configuration section

---

## Q2: Outage Behavior -- "What happens when the gateway is down?"

### Short Answer

API-key mode requests fail fast with clear errors. Subscription mode continues unaffected but without governance logging during the outage.

### Detailed Answer

**API-Key Mode (Enforcement Path):**

When the gateway is unavailable:
- Applications receive connection errors (HTTP 502/503)
- No requests reach providers (fail-closed behavior)
- Budget tracking pauses (no spend accumulation)
- Recovery: automatic when gateway restarts

This is intentional: we prefer denied requests over ungoverned requests.

**Subscription Mode (Governance Path):**

Subscription tools (Claude Code, Codex, Cursor) connect directly to providers via OAuth:
- They continue working independently of gateway status
- No governance logging during gateway outage
- Compliance exports from vendors still capture activity (delayed)
- OTEL telemetry may be lost if gateway is the collection point

**Availability Boundary:**

For production deployments, the supported repository baseline is a **single-node** host-first deployment. Truthful guidance:

| Topic | Truthful answer |
|-------|-----------------|
| Container health checks | They can restart services on the same host, but that is recovery, not failover. |
| Automatic failover | Not part of the current supported repo contract. |
| Current validated topology | One host running LiteLLM, PostgreSQL, and any selected overlays. |
| Next credible HA pattern | Active-passive with PostgreSQL replication and customer-owned traffic cutover, documented as reference guidance only. |

See `docs/deployment/HA_FAILOVER_TOPOLOGY.md` for the explicit failure-domain and recovery-vs-failover model.

**Recovery Validation:**

After any outage, run:
```bash
make health
make detection
```

### Evidence References

- `docs/RUNBOOK.md` -- incident response procedures
- `docs/deployment/HA_FAILOVER_TOPOLOGY.md` -- single-node availability boundary and next-step HA reference
- `demo/logs/evidence/12_failure_injection.log` -- failure testing evidence

---

## Q3: Scaling Limits -- "How many requests per second can this handle?"

### Short Answer

The architecture supports vertical and horizontal scaling, but this public repository should not be used as customer-grade capacity proof on its own. Capacity claims need a customer-like load test in the target environment.

### Detailed Answer

**What the repo proves now:**

- The gateway and database topology are explicit.
- The validated host-first deployment track is documented, and incubating Kubernetes material is clearly separated from the supported surface.
- Operational checks, health gates, and release workflows are reproducible.

**What still requires environment-specific validation:**

- target requests/second
- latency under customer traffic patterns
- provider-side bottlenecks
- database sizing and pooling strategy
- network and SIEM overhead

**Scaling Strategies:**

1. **Vertical Scaling**: Increase CPU/RAM on gateway host
   - Linear improvement up to ~8 cores
   - Database becomes bottleneck at high throughput

2. **Horizontal Scaling**: Multiple gateway replicas behind load balancer
   - Shared PostgreSQL database (connection pooling recommended)
   - Stateless gateway design enables easy scaling

3. **Database Scaling**:
   - Connection pooling (PgBouncer) for high concurrency
   - Read replicas for reporting queries
   - Partitioning for high-volume audit logs

**Production Sizing Guidance:**

| Scale | Gateway Instances | Database |
|-------|-------------------|----------|
| Small (<1000 req/day) | 1 | Docker PostgreSQL |
| Medium (<100K req/day) | 2-4 | Managed PostgreSQL |
| Large (>100K req/day) | 4+ | Managed PostgreSQL + read replica |

**Run Your Own Load Test:**

Use external load-testing tooling in the customer-like environment after the gateway baseline is stable.

### Evidence References

- `docs/DEPLOYMENT.md` -- hardware requirements section
- `docs/deployment/KUBERNETES_HELM.md` -- scale-out deployment track

---

## Q4: Key Rotation -- "How do you rotate keys without service disruption?"

### Short Answer

Generate a replacement key, distribute it to clients, then revoke the old key. There's no service disruption because both keys work simultaneously during the transition.

### Detailed Answer

**Key Rotation Workflow:**

1. **Generate Replacement Key**
   ```bash
   make key-gen ALIAS=app-v2__team-platform BUDGET=100.00
   ```

2. **Distribute New Key**
   - Update secrets manager (HashiCorp Vault, AWS Secrets Manager, etc.)
   - Trigger application redeploy / config reload
   - Clients automatically pick up new key

3. **Monitor Dual-Key Period**
   - Both keys are valid simultaneously
   - Audit logs show which key version is being used
   ```bash
   make db-status | grep "app-"
   ```

4. **Revoke Old Key**
   ```bash
   make key-revoke ALIAS=<alias>
   ```

5. **Verify**
   ```bash
   # Confirm only new key appears in active requests
   make detection  # Check for auth/authz findings
   ```

**Best Practices:**

| Practice | Description |
|----------|-------------|
| Key naming convention | Include version: `app-v1`, `app-v2` |
| Overlap period | 24-48 hours for safe transition |
| Automated rotation | Integrate with secrets manager rotation |
| Audit trail | Key lifecycle events are logged |

**Automated Rotation (CI/CD):**

```yaml
# Example GitLab CI rotation job
rotate_keys:
  script:
    - make key-gen ALIAS=app-v${CI_PIPELINE_ID} BUDGET=100.00
    - ./scripts/update_secrets_manager.sh
    - sleep 86400  # 24-hour overlap
    - make key-revoke ALIAS=<alias>
  rules:
    - if: $SCHEDULED_ROTATION
```

### Evidence References

- `docs/demos/API_KEY_GOVERNANCE_DEMO.md` lines 489-525
- `demo/logs/evidence/04_key_lifecycle.log`
- `docs/RUNBOOK.md` -- key rotation procedures

---

## Q5: Compliance Fit -- "Does this meet SOC2/HIPAA/GDPR requirements?"

### Short Answer

The AI Control Plane provides audit logging, access controls, and data governance capabilities that support compliance. Specific compliance requirements depend on your organization's interpretation and should be reviewed with your compliance team.

### Detailed Answer

**Compliance Control Mapping:**

| Control | SOC2 | HIPAA | GDPR | Implementation |
|---------|------|-------|------|----------------|
| Audit logging | CC6.1 | 164.312(b) | Art. 30 | All requests logged with attribution |
| Access control | CC6.1 | 164.312(d) | Art. 32 | Per-key RBAC, budget limits |
| Data minimization | CC6.5 | 164.502(b) | Art. 5 | DLP blocks PII before providers |
| Encryption in transit | CC6.7 | 164.312(e)(1) | Art. 32 | TLS mode available |
| Incident response | CC7.4 | 164.308(a)(6) | Art. 33 | Detection rules, runbooks |

**What We Provide:**

1. **Audit Trail**: Every request logged with principal, model, tokens, cost
2. **Access Control**: Per-key model allowlists and budgets
3. **DLP**: PII blocking before data reaches providers
4. **Detection**: Anomaly detection for security incidents
5. **Runbooks**: Documented incident response procedures

**What Requires Customer Action:**

- Data processing agreements (DPAs) with AI providers
- BAA for HIPAA (provider-specific)
- Data residency configuration (provider regions)
- Retention policy implementation
- Regular access reviews

**Transparency Note:**

> "The AI Control Plane is a governance layer. Compliance certification (SOC2, HIPAA, etc.) for the underlying AI services remains with the providers (OpenAI, Anthropic). Our role is to provide the control plane for audit, access management, and data protection."

### Evidence References

- `docs/DEPLOYMENT.md` lines 1693-1702 -- compliance configuration
- `docs/security/DETECTION.md` -- audit logging details
- `docs/RUNBOOK.md` -- incident response procedures

---

## Q6: Data Residency -- "Where does our data go?"

### Short Answer

You control data residency through provider configuration. The gateway can be deployed in your region, and providers offer regional endpoints. Request content is not stored by the gateway (metadata only).

### Detailed Answer

**Data Flow:**

```
Client Request -> Gateway (your infra) -> Provider API (your region config) -> Response
                      |
                      v
              PostgreSQL (metadata only: tokens, cost, timestamps)
```

**What the Gateway Stores:**

| Field | Stored? | Purpose |
|-------|---------|---------|
| Model ID | Yes | Audit, cost attribution |
| Token counts | Yes | Usage tracking |
| Cost (USD) | Yes | Budget enforcement |
| Timestamp | Yes | Audit timeline |
| Key alias | Yes | Attribution |
| Request content | No | Privacy |
| Response content | No | Privacy |

**Provider Regional Endpoints:**

| Provider | Regional Option |
|----------|-----------------|
| OpenAI | US (default), EU (Azure) |
| Anthropic | US (default) |
| Azure OpenAI | Region selection |
| AWS Bedrock | Region selection |

**Deployment Options:**

1. **On-premises**: Gateway runs in your data center
2. **Cloud VPC**: Gateway in AWS/Azure/GCP VPC
3. **Hybrid**: Gateway on-prem, database in cloud

**GDPR Considerations:**

- Configure provider to use EU data residency
- Gateway can be deployed in EU region
- Data processing agreement required with providers
- Right to erasure: database records can be purged per retention policy

### Evidence References

- `docs/DEPLOYMENT.md` -- deployment location options
- `docs/DATABASE.md` -- schema details (metadata only)

---

## Q7: Multi-Tenancy -- "Can we isolate different business units?"

### Short Answer

Yes. Use key namespacing for logical isolation, or deploy separate gateway instances for physical isolation. Budget tracking is per-key by default.

### Detailed Answer

**Option 1: Key Namespacing (Logical Isolation)**

```bash
# Business Unit A
make key-gen ALIAS=bu-finance__team-accounting BUDGET=500.00
make key-gen ALIAS=bu-finance__team-payables BUDGET=250.00

# Business Unit B
make key-gen ALIAS=bu-engineering__team-platform BUDGET=1000.00
make key-gen ALIAS=bu-engineering__team-ml BUDGET=2000.00
```

**Chargeback Query:**

```sql
SELECT
  substring(key_alias from '^bu-([^_]+)') AS business_unit,
  ROUND(SUM(spend)::numeric, 2) AS total_spend,
  COUNT(*) AS request_count
FROM "LiteLLM_SpendLogs" s
JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
GROUP BY business_unit
ORDER BY total_spend DESC;
```

**Option 2: Separate Deployments (Physical Isolation)**

For strict isolation (different compliance domains, acquisitions):

| Business Unit | Gateway | Database |
|---------------|---------|----------|
| Finance | gateway-finance.internal | db-finance.internal |
| Engineering | gateway-eng.internal | db-eng.internal |
| Healthcare | gateway-hc.internal | db-hc.internal (BAA required) |

**Comparison:**

| Aspect | Namespacing | Separate Deployments |
|--------|-------------|---------------------|
| Cost | Lower (shared infra) | Higher (dedicated) |
| Isolation | Logical | Physical |
| Cross-BU visibility | Yes (central query) | No (separate DBs) |
| Compliance | Shared responsibility | Per-domain |
| Operational overhead | Lower | Higher |

**Per-Key Model Restrictions:**

Model access is controlled via roles, not a direct MODELS parameter.

```bash
# Finance can only use cheaper models (developer role)
make key-gen-dev ALIAS=bu-finance BUDGET=100.00

# Engineering can use more models (team-lead role)
make key-gen-lead ALIAS=bu-engineering BUDGET=500.00
```

> **Note:** Available roles are `admin`, `team-lead`, `developer`, `auditor`. Each role has a predefined model allowlist configured in `demo/config/roles.yaml`.

### Evidence References

- `docs/ENTERPRISE_STRATEGY.md` lines 293-304
- `docs/demos/API_KEY_GOVERNANCE_DEMO.md` -- chargeback walkthrough
- `docs/policy/FINANCIAL_GOVERNANCE_AND_CHARGEBACK.md`

---

## Q8: Vendor Lock-In -- "What if we want to switch from LiteLLM?"

### Short Answer

LiteLLM is open-source (MIT license) using the standard OpenAI API format. Your configuration is portable YAML. Switching costs are low -- primarily re-implementing the key management layer.

### Detailed Answer

**Portability Factors:**

| Component | Portability | Notes |
|-----------|-------------|-------|
| API format | High | Standard OpenAI `/v1/chat/completions` |
| Configuration | High | YAML files, easily translated |
| Key management | Medium | LiteLLM-specific schema |
| Detection rules | High | SQL queries, standard PostgreSQL |
| SIEM integration | High | Standard JSON output |

**Migration Path (if needed):**

1. **Export configuration**: `cat demo/config/litellm.yaml`
2. **Export keys**: Query `LiteLLM_VerificationToken` table
3. **Export detection rules**: `cat demo/config/detection_rules.yaml`
4. **Re-implement key management**: Most alternatives have similar concepts

**Alternative Gateway Options:**

| Gateway | Key Management | OpenAI Compatible |
|---------|---------------|-------------------|
| LiteLLM | Built-in | Yes |
| OpenAI Proxy | API keys only | Native |
| AWS Bedrock | IAM roles | Via adapter |
| Azure OpenAI | Azure AD | Via adapter |
| Kong + plugin | Custom | Yes |

**Why LiteLLM for This Solution:**

1. **Multi-provider**: OpenAI, Anthropic, Azure, AWS, local models
2. **Virtual keys**: Per-user/per-service keys with budgets
3. **Open source**: MIT license, active community
4. **DLP integration**: Presidio support built-in

**Lock-In Mitigation:**

- Document custom configurations
- Use standard detection rule formats
- Maintain export scripts for key data
- Regular configuration reviews

### Evidence References

- `demo/config/litellm.yaml` -- portable configuration
- `docs/DEPLOYMENT.md` -- alternative deployment options
- LiteLLM documentation: https://docs.litellm.ai/

For presenter delivery flow, pair this packet with `../presentation/PRESENTATION_GUIDE.md`.

---

## Quick Reference: Objection Handling

| Objection | Response Key |
|-----------|--------------|
| "Developers will bypass this" | Q1: Detection + egress controls |
| "What if it goes down?" | Q2: Fail-closed, rapid recovery |
| "Will it scale?" | Q3: Capacity requires environment-specific validation; single-node support boundary and scaling limits are explicit |
| "Key rotation is hard" | Q4: Generate-new-then-revoke workflow |
| "Is it compliant?" | Q5: Control mapping, requires customer review |
| "Where's our data?" | Q6: You control, metadata only |
| "We need isolation" | Q7: Namespacing or separate deployments |
| "What about lock-in?" | Q8: Open source, standard APIs, portable config |

---

## Related Documentation

- [10-Minute Demo Script](10_minute_script.md)
- [Presentation Guide](../presentation/PRESENTATION_GUIDE.md)
- [Executive One-Pager](../presentation/EXECUTIVE_ONE_PAGER.md)
- [Known Limitations](../KNOWN_LIMITATIONS.md)
- [Deployment Guide](../DEPLOYMENT.md)
- [Runbook](../RUNBOOK.md)
- [Detection Rules](../security/DETECTION.md)
