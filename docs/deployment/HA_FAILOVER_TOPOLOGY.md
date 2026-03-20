# HA And Failover Topology

The primary validated deployment contract in this repository remains a **single-node** host-first deployment. In addition, the repository now validates a **customer-operated manual active-passive failover drill evidence surface** for a two-host reference pattern. This document explains what is primary today, what HA proof is now validated, and what ACP still does **not** automate.

This is topology and claim-boundary guidance, not runtime automation. Do not describe ACP as automatic failover, cluster orchestration, or ACP-managed PostgreSQL replication.

## Current Supported Topology

The current supported production topology is:

- one Debian 12+ or Ubuntu 24.04+ host
- one Docker/Compose runtime
- one LiteLLM gateway
- one PostgreSQL instance
- optional overlays on the same host: Caddy TLS, Presidio DLP, LibreChat UI, offline mode
- one canonical secrets file: `/etc/ai-control-plane/secrets.env`
- automated local backups via `ai-control-plane-backup.timer` (default: daily, keep 7)
- recovery by restoring host access, restoring the database, re-applying the host deployment, and re-running health checks

The tracked Ansible playbook, `deploy/ansible/playbooks/gateway_host.yml`, still converges **one gateway host at a time**. The validated two-host reference pattern reuses that single-host convergence surface against each host before and after a manual promotion. ACP still does not ship a cluster playbook, replication controller, fencing controller, or automatic traffic-cutover controller.

```text
Clients
   |
   v
[ optional Caddy TLS ]
   |
   v
[ LiteLLM gateway ] <----> [ PostgreSQL ]
        \______________________________/
             same host, same storage,
             same operator recovery path
```

Any selected overlays share that same host-level failure domain.

## What "High Availability" And "Failover" Mean Here

For this repository:

- **High availability** means service remains available across a host or database failure without rebuilding the primary service from backup on that same node.
- **Failover** means client traffic and authoritative state move to a secondary host or secondary database role after a failure.
- **Recovery** means operators restore service after a failure by restarting services, restoring data, re-applying the deployment, or rebuilding a host.

Truthful boundary for the current supported topology:

- Service restarts on the same host are **recovery**, not failover.
- Backup/restore workflows are **disaster recovery**, not automatic HA.
- The current supported host-first contract does **not** provide automatic failover for host loss, local storage loss, or database loss.
- The current supported contract is **single-node operation plus documented recovery**.

## Failure Domains In The Current Supported Topology

| Failure domain | What fails together | Expected impact | Supported response today | Automatic failover? |
| --- | --- | --- | --- | --- |
| LiteLLM process/container failure | Gateway request path on the host | Gateway requests fail until service is restarted or re-converged | `make health`, `./scripts/acpctl.sh doctor`, `./scripts/acpctl.sh host apply ...` | No |
| PostgreSQL process/container failure | Gateway state and request handling | The gateway can become unavailable or unhealthy because its database is unavailable | DB restore/status workflows, then re-apply and re-verify | No |
| Docker daemon / Compose runtime failure | All ACP containers on the host | Full control-plane outage on that host | Restore Docker health, re-run host workflow, verify health/smoke | No |
| Host OS / VM / hardware failure | Gateway, database, overlays, systemd timers, local artifacts | Full service outage | Restore a host, restore secrets, restore the database, re-apply the deployment | No |
| Local storage failure or corruption | Database files, repo checkout, local backups, local cert/runtime state | Outage plus possible data loss | Restore from a known-good backup copy and re-converge | No |
| Customer DNS / load balancer / network failure | External reachability | Clients may lose access even if ACP services are healthy | Customer-operated DNS/LB/network recovery | No |

## Backup, RPO, And RTO Truth

The default repo-backed protection mechanisms are **scheduled backups and documented restore drills**.

- Default automated backup cadence: `daily`
- Default retention: keep newest `7` backups
- Default backup location in the tracked runtime: `demo/backups/`

Truthful implications:

- With the default daily backup timer, the **worst-case reference RPO** is up to **24 hours of data loss** since the last successful backup.
- That RPO is a **reference based on current timer defaults**, not a universal SLA or guarantee.
- If backups exist **only on the same host/storage that failed**, a disk or host-loss event can destroy both the live database and the local backup artifacts. In that case, effective data loss can be total.
- Off-host backup copies and retention remain customer-owned. The repo now validates a staged off-host copy through an explicit manifest plus scratch-restore drill, with truthful `staged-local` and `separate-host` evidence labeling, but it still does **not** automate replication transport into S3, NFS, rsync targets, or other customer storage, and neither drill mode is HA or failover automation.
- The repo does **not** currently publish a fixed minute-based RTO. Recovery time depends on spare-host availability, operator response, backup accessibility, network cutover, and environment-specific constraints.
- The validated recovery sequence is: restore host access -> restore `/etc/ai-control-plane/secrets.env` -> restore the database -> re-apply the deployment -> verify with `make health` and `make prod-smoke`.

See [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md) for the supported restore workflow.

## Customer-Owned Availability Responsibilities

The current host-first support boundary does **not** own these HA building blocks:

- off-host backup copy/replication and longer-term retention remain customer-owned; ACP validates a staged recovery copy but does not perform the transport
- replacement-host provisioning or standby-host readiness
- DNS failover, load balancer health checks, VIP ownership, or traffic-manager cutover
- network routing, firewall policy, and external reachability
- PostgreSQL replication, fencing, and split-brain prevention in any multi-host design
- monitoring/on-call procedures for customer infrastructure
- customer-specific failover drills and acceptance criteria

## Validated Active-Passive Reference Pattern: Manual Failover With PostgreSQL Replication

The next credible HA pattern is now **validated as a customer-operated manual drill evidence surface**.

Validated components now present in the repository:

- typed failover-drill contract validation in `internal/ha`
- operator command surface: `acpctl host failover-drill`
- make entrypoint: `make ha-failover-drill`
- repeatable private evidence bundling under `demo/logs/evidence/ha-failover-drill/`
- operator runbook: [HA_FAILOVER_RUNBOOK.md](HA_FAILOVER_RUNBOOK.md)
- example two-host inventory: `deploy/ansible/inventory/hosts.ha.example.yml`
- production-only readiness-gate wiring in `demo/config/readiness_evidence.yaml`

> Manual customer-operated active-passive failover proof only. ACP validates the drill contract and archives evidence for replication readiness, fencing, promotion, traffic cutover, and post-cutover checks. ACP does not automate PostgreSQL replication, promotion, or customer-owned DNS/load-balancer/VIP cutover.

Why this is the validated next step:

- it preserves one authoritative PostgreSQL writer
- it reduces the single-host outage gap without pretending ACP ships automatic HA
- it fits the existing host-first operating model by converging one host at a time
- it keeps customer-owned fencing and traffic cutover explicit

Reference pattern:

```text
                    +-------------------------------+
Clients ----------->| Customer-owned DNS / LB / VIP |
                    +-------------------------------+
                               |
                               v
                     +-------------------+
                     | Active ACP host   |
                     | LiteLLM + Caddy   |
                     | PostgreSQL primary|
                     +-------------------+
                               |
                     streaming replication
                               |
                               v
                     +-------------------+
                     | Passive ACP host  |
                     | warm standby ACP  |
                     | PostgreSQL replica|
                     +-------------------+
```

Recommended characteristics:

- **Active host:** serves client traffic and runs the PostgreSQL primary.
- **Passive host:** remains provisioned and ready, with PostgreSQL receiving streaming replication and ACP deployment assets staged.
- **Customer-owned cutover:** DNS, load balancer, or virtual IP determines which host receives traffic.
- **Promotion discipline:** on primary loss, fence the failed primary first, promote the standby database, start and verify the standby gateway path, then swing traffic.
- **Secrets discipline:** the canonical secrets material required for promotion must exist on both hosts through a secure customer-owned distribution process.
- **Backup discipline:** keep backups off-host even when replication exists; replication protects availability, not accidental deletion or corruption.

Important boundary:

- This pattern is a **validated customer-operated manual failover-drill evidence surface** in the current repository.
- ACP still does **not** automate multi-host failover, PostgreSQL promotion, fencing, split-brain prevention, or customer-owned cutover.
- Do **not** present this pattern as automatic failover or a managed cutover controller.

## Decision Guide

| Requirement | Truthful answer today |
| --- | --- |
| "Can the supported host-first deployment survive a single-host outage without downtime?" | The primary deployment topology is still single-node, so not by itself. |
| "Do scheduled backups equal HA?" | No. They improve recovery, not failover. |
| "Is disaster recovery supported?" | Yes. Backup, restore, scratch-restore drills, and re-apply workflows are part of the current contract. |
| "Is there a validated HA pattern?" | Yes. A customer-operated active-passive reference pattern is validated through the typed failover-drill contract, runbook, inventory example, and repeatable evidence workflow. |
| "Is automatic failover supported?" | No. ACP does not automate PostgreSQL replication, promotion, fencing, split-brain prevention, or customer-owned DNS/load-balancer/VIP cutover. |

## Related Documents

- [Operations And Deployment](../DEPLOYMENT.md)
- [Active-Passive HA Failover Runbook](HA_FAILOVER_RUNBOOK.md)
- [Disaster Recovery](DISASTER_RECOVERY.md)
- [Single-Tenant Production Contract](SINGLE_TENANT_PRODUCTION_CONTRACT.md)
- [Support](../SUPPORT.md)
- [Go-To-Market Scope](../GO_TO_MARKET_SCOPE.md)
