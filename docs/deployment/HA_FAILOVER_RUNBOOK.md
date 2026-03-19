# Active-Passive HA Failover Runbook

This runbook turns the host-first active-passive reference pattern into a concrete **customer-operated** drill workflow.

It is intentionally explicit about what ACP validates and what ACP does **not** automate.

## Boundary

This runbook covers a manual active-passive failover drill with:

- one active ACP host
- one passive ACP host
- one PostgreSQL primary on the active host
- one PostgreSQL replica on the passive host
- customer-owned fencing and traffic cutover

This runbook does **not** provide:

- automatic failover
- split-brain prevention automation
- ACP-managed PostgreSQL replication setup
- ACP-managed DNS, load-balancer, or VIP cutover

For topology truth and failure-domain guidance, see [HA_FAILOVER_TOPOLOGY.md](HA_FAILOVER_TOPOLOGY.md).

## Required Inputs

Prepare all of the following before the drill:

- host inventory based on [../../deploy/ansible/inventory/hosts.ha.example.yml](../../deploy/ansible/inventory/hosts.ha.example.yml)
- canonical secrets file on both hosts: `/etc/ai-control-plane/secrets.env`
- customer-owned PostgreSQL replication already configured and caught up
- documented fencing path for the active host
- documented traffic-cutover path (`dns`, `load-balancer`, `vip`, or truthful `manual`)
- writable evidence directory on the drill host, for example `/var/tmp/ai-control-plane-ha/`
- a copied failover manifest based on [../../demo/config/ha_failover_drill.example.yaml](../../demo/config/ha_failover_drill.example.yaml)

## PostgreSQL Replication Guidance

ACP does not configure replication for you, but the active-passive reference pattern expects these truths:

- exactly one authoritative writer before failover
- the passive node is in streaming-replication or equivalent warm-standby state before promotion
- operators can prove replica health before fencing the primary
- backups remain off-host even when replication exists

Recommended pre-failover checks to capture into `replication_evidence_path`:

```bash
ssh acp-active-1 "sudo -u postgres psql -x -c 'select application_name, client_addr, state, sync_state, write_lag, flush_lag, replay_lag from pg_stat_replication;'"
ssh acp-passive-1 "sudo -u postgres psql -x -c 'select status, receive_start_lsn, received_tli, latest_end_lsn, latest_end_time from pg_stat_wal_receiver;'"
```

If these checks do not show a healthy replica, stop the drill.

## Drill Sequence

### 1. Validate the passive host is convergable

Run a typed preflight against the passive inventory target:

```bash
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.ha.yml --limit acp-passive-1
```

### 2. Capture replication readiness evidence

Run the replication checks above and store the output in the file referenced by `replication_evidence_path`.

### 3. Fence the active primary first

Use the customer-owned mechanism for the failed or soon-to-be-failed primary host.
Examples include IPMI power-off, hypervisor shutdown, cloud-instance stop, or network fencing.

Record the exact action and result in the file referenced by `fencing.evidence_path`.

Do **not** promote the standby until fencing is complete.

### 4. Promote the passive PostgreSQL node

Example promotion command:

```bash
ssh acp-passive-1 "sudo -u postgres psql -c 'select pg_promote(wait_seconds => 60);'"
```

Record promotion output and any role-verification queries in the file referenced by `promotion.evidence_path`.

### 5. Re-converge the promoted host before traffic

Run host convergence on the promoted target without smoke gating first:

```bash
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.ha.yml --limit acp-passive-1 --skip-smoke-tests
```

### 6. Perform customer-owned traffic cutover

Swing traffic using the method declared in `traffic_cutover.method`:

- `dns` — update authoritative records or failover aliases
- `load-balancer` — shift backend target health or pool membership
- `vip` — move the virtual IP under customer control
- `manual` — use the customer-specific documented handoff method

Record the cutover action and observed result in `traffic_cutover.evidence_path`.

### 7. Re-run full convergence and postchecks

After traffic points at the promoted host:

```bash
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.ha.yml --limit acp-passive-1
```

Capture the resulting health and smoke proof in `postcheck_evidence_path`.
Recommended evidence includes:

```bash
make health COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
make prod-smoke COMPOSE_ENV_FILE=/etc/ai-control-plane/secrets.env
```

### 8. Archive the drill evidence bundle

Once all evidence files exist and the manifest is truthful, archive the drill bundle:

```bash
make ha-failover-drill HA_FAILOVER_MANIFEST=demo/logs/recovery-inputs/ha_failover_drill.yaml
```

Equivalent direct CLI form:

```bash
./scripts/acpctl.sh host failover-drill --manifest demo/logs/recovery-inputs/ha_failover_drill.yaml
```

Successful runs write a private local bundle under:

- `demo/logs/evidence/ha-failover-drill/`

The generated summary records the claim boundary so the evidence cannot be mistaken for automatic failover support.

## Manifest Contract

The typed drill command requires evidence for all of these stages:

- replication readiness
- fencing
- promotion
- traffic cutover
- post-cutover verification

Start from [../../demo/config/ha_failover_drill.example.yaml](../../demo/config/ha_failover_drill.example.yaml).

## Readiness Evidence Integration

For customer-like production readiness runs, include the failover manifest at:

- `demo/logs/recovery-inputs/ha_failover_drill.yaml`

Then run:

```bash
make readiness-evidence READINESS_INCLUDE_PRODUCTION=1 \
  SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

The production-only readiness workflow will invoke `make ha-failover-drill` and archive the corresponding command log.

## Related Documents

- [HA_FAILOVER_TOPOLOGY.md](HA_FAILOVER_TOPOLOGY.md)
- [DISASTER_RECOVERY.md](DISASTER_RECOVERY.md)
- [../DEPLOYMENT.md](../DEPLOYMENT.md)
- [../release/READINESS_EVIDENCE_WORKFLOW.md](../release/READINESS_EVIDENCE_WORKFLOW.md)
