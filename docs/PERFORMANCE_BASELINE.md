# Performance Baseline

This document defines the local performance harness for the validated host-first reference baseline.

It is intentionally narrow. The goal is to answer, "what does this reference stack do on this host right now?" It is not meant to claim universal scale, customer-environment capacity, or production SLA proof.

## What This Harness Is For

Use the performance baseline to:

- measure gateway-added latency on the current host
- compare one local run to another after a configuration change
- produce a reference-host data point for buyer conversations
- separate architecture claims from measured proof

Use it with the offline stack when you want deterministic, provider-independent results.

## Canonical Commands

Start the offline stack if you want the most repeatable local result:

```bash
make up-offline
```

Run the benchmark:

```bash
make performance-baseline
```

Adjust the run shape when needed:

```bash
make performance-baseline PERFORMANCE_REQUESTS=40 PERFORMANCE_CONCURRENCY=4 PERFORMANCE_MODEL=mock-gpt
```

Run a named workload profile when you want the repository-defined shape:

```bash
make performance-baseline PERFORMANCE_PROFILE=interactive
make performance-baseline PERFORMANCE_PROFILE=burst
make performance-baseline PERFORMANCE_PROFILE=sustained
```

Direct CLI form:

```bash
./scripts/acpctl.sh benchmark baseline --requests 20 --concurrency 2 --json
./scripts/acpctl.sh benchmark baseline --profile interactive --json
```

## Default Baseline Shape

The default harness uses:

- gateway URL: resolved from `ACP_GATEWAY_URL` or `GATEWAY_URL`, then `GATEWAY_HOST` + `LITELLM_PORT` + `ACP_GATEWAY_SCHEME`/`ACP_GATEWAY_TLS`, then `http://127.0.0.1:4000`
- model: `mock-gpt`
- requests: `20`
- concurrency: `2`
- max tokens: `32`

These defaults are intentionally conservative so the command is lightweight and safe to rerun during operator workflows.

Reference thresholds and hardware guidance are defined in `demo/config/benchmark_thresholds.json`.

## What The Output Means

The summary reports:

- total requests
- concurrency level
- success and failure counts
- wall-clock duration
- throughput in requests per second
- latency distribution: average, p50, p95, min, and max

Interpret the result as a local reference-host baseline only.

## Default Threshold Guidance

For the default offline baseline shape (`mock-gpt`, `20` requests, concurrency `2`), use this guidance:

| Signal | Healthy reference-host guidance | What it means |
|---|---|---|
| Failures | `0` | The baseline should complete cleanly in the validated reference environment |
| Throughput | `>= 3 req/s` | Lower values usually mean the host is saturated, cold, or unhealthy |
| p95 latency | `<= 2500 ms` | Above this, the local stack is too sluggish to present as a healthy baseline |
| Average latency | `<= 1200 ms` | Sustained average latency above this suggests local contention or startup drag |
| Max latency | Review if `> 5000 ms` | One slow request is acceptable; repeated spikes need explanation |

These are operator guidance thresholds, not contractual service levels.

## Interpreting Threshold Breaches

If the baseline breaches one threshold:

- rerun once after the stack is warm
- confirm the host is not resource-constrained
- check `make health` and recent restart activity

If the baseline breaches multiple thresholds:

- do not present the result as a healthy reference-host baseline
- investigate local host saturation, Docker slowdown, or gateway configuration drift
- regenerate the baseline before using it in a customer conversation

## Workload Profiles

The repository also carries broader reference profiles in `demo/config/benchmark_thresholds.json`:

- `interactive`
- `burst`
- `sustained`

Those profiles are now runnable through the benchmark command and Make target. Treat them as reference-host workload shapes, not universal proof. If you use a profile in a buyer conversation, rerun it in the target environment before making rollout claims.

## What This Does Not Prove

This harness does not prove:

- customer-environment throughput
- WAN or provider latency
- SaaS/browser route behavior
- Kubernetes/Helm parity
- procurement-grade scale commitments

For enterprise conversations, present these numbers as:

- local baseline evidence
- useful for regression detection
- rerun required in the target customer environment

## Recommended Use In Sales And Delivery

Use the benchmark as supporting evidence, not as the lead proof artifact.

It is strongest when paired with:

- `make readiness-evidence`
- `docs/ENTERPRISE_PILOT_PACKAGE.md`
- `docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md`
- `docs/BROWSER_WORKSPACE_PROOF_TRACK.md`

That combination lets you say:

- this is what the baseline does locally
- this is what the repo can currently prove
- this is what still must be validated in the customer environment

## Recommended Talk Track

When a buyer asks about scale, the clean answer is:

> We keep a repeatable local reference-host baseline and treat it as regression evidence, not as universal capacity proof. We can show current throughput and latency on the validated host-first stack, then rerun the same measurement in your environment before making rollout claims.
