# Performance Benchmarks and Sizing Guidance

This document is the published performance and sizing artifact for the AI Control Plane host-first reference deployment.

> **Claim boundary:** these benchmarks measure the current host-first reference stack on a specific host and configuration. They are useful as reproducible capacity evidence and regression evidence. They do **not** prove customer-environment capacity, production SLAs, WAN/provider latency, browser-route behavior, Kubernetes parity, or universal scale guarantees.

## What is published here

This artifact publishes the roadmap-required benchmark inputs and interpretation rules:

- methodology
- workload profiles
- hardware and software baseline
- published result bands and interpretation guidance
- sizing caveats
- reproducibility steps

The machine-readable source of truth for named workload profiles and hardware tiers is:

- `demo/config/benchmark_thresholds.json`

The runnable harness is implemented in:

- `internal/performance/baseline.go`
- `internal/performance/profile.go`
- `cmd/acpctl/cmd_benchmark.go`

## Canonical commands

Recommended reproducible path:

```bash
make up-offline
make performance-baseline
make performance-baseline PERFORMANCE_PROFILE=interactive
make performance-baseline PERFORMANCE_PROFILE=burst
make performance-baseline PERFORMANCE_PROFILE=sustained
```

Direct typed CLI remains available when you need machine-readable output:

```bash
./scripts/acpctl.sh benchmark baseline --json
./scripts/acpctl.sh benchmark baseline --profile interactive --json
./scripts/acpctl.sh benchmark baseline --profile burst --json
./scripts/acpctl.sh benchmark baseline --profile sustained --json
```

## Benchmark methodology

The benchmark harness measures the gateway by issuing concurrent HTTP requests to:

- `POST /v1/chat/completions`

For each measured request, the harness records:

- per-request latency
- success or failure status

It then calculates:

- total requests
- success count
- failure count
- wall-clock duration
- throughput in requests per second
- average latency
- p50 latency
- p95 latency
- minimum latency
- maximum latency

### Harness behavior

| Aspect | Published behavior |
| --- | --- |
| Driver | `make performance-baseline` -> `acpctl benchmark baseline` |
| Endpoint | `/v1/chat/completions` |
| Default model | `mock-gpt` |
| Default prompt shape | fixed short prompt for repeatability |
| Default `max_tokens` | `32` |
| Default raw baseline | `20` measured requests at concurrency `2` |
| Named profiles | `interactive`, `burst`, `sustained` |
| Warmup support | yes; named profiles define warmup counts |
| Readiness guard | `make performance-baseline` waits for stack readiness before running |
| Recommended mode for repeatability | offline stack with mock models |

### Gateway URL resolution

The benchmark target resolves in this order:

1. `ACP_GATEWAY_URL`
2. `GATEWAY_URL`
3. `GATEWAY_HOST` + `LITELLM_PORT` + `ACP_GATEWAY_SCHEME` / `ACP_GATEWAY_TLS`
4. fallback `http://127.0.0.1:4000`

### Default lightweight baseline

The default lightweight baseline exists for quick operator checks:

- model: `mock-gpt`
- warmup requests: `0`
- measured requests: `20`
- concurrency: `2`
- `max_tokens`: `32`

This is intentionally smaller than the named profiles so it is cheap to rerun during normal workflows.

## Software baseline

The published methodology assumes the current repository revision of the host-first reference deployment and benchmark harness.

For the most repeatable local result, use:

- the host-first Docker reference stack
- the offline runtime path
- mock models such as `mock-gpt` or `mock-claude`
- the canonical Make entrypoint `make performance-baseline`

This artifact documents how the repository measures performance today. If the harness changes, this document must change with it.

## Workload profiles and published result bands

Named workload profiles are defined in `demo/config/benchmark_thresholds.json` and runnable via `make performance-baseline PERFORMANCE_PROFILE=<name>`.

| Profile | Description | Warmup Requests | Measured Requests | Concurrency | p95 Latency Max | Error Rate Max | Throughput Min | Estimated Cost Max |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `interactive` | Low-latency interactive chat workloads (1-2 concurrent users) | 5 | 30 | 1 | 1800 ms | 1.0% | 1.0 req/s | $0.10 |
| `burst` | Short-duration burst workloads (spikes in traffic) | 5 | 60 | 4 | 2500 ms | 2.0% | 3.0 req/s | $0.30 |
| `sustained` | Sustained throughput workloads (batch processing, agents) | 10 | 100 | 6 | 3000 ms | 2.5% | 4.0 req/s | $0.50 |

These published bands are **reference-host interpretation bands**, not customer contracts.

### How to use the published bands

- Use them to judge whether a local run looks healthy for the selected profile.
- Use them to compare one local environment to another using the same benchmark shape.
- Do **not** present them as guaranteed production numbers for a buyer's environment without rerunning there.

## Reference host hardware tiers

The benchmark catalog also publishes reference single-host hardware tiers.

| Tier | CPU Cores | Memory | Storage | Network | Recommended For |
| --- | ---: | ---: | ---: | ---: | --- |
| `single_host_minimum` | 4 | 8 GB | 50 GB | 100 Mbps | Development, testing, and light interactive workloads |
| `single_host_recommended` | 8 | 16 GB | 100 GB | 1000 Mbps | Production interactive workloads up to 10 concurrent users |
| `single_host_high_throughput` | 16 | 32 GB | 200 GB | 1000 Mbps | Burst and sustained throughput workloads, 20+ concurrent users |

These are **starting-point sizing tiers**, not guaranteed capacity commitments.

## Practical sizing guidance

Use the tiers as a conservative starting point:

- **Development, demos, and light validation:** start at `single_host_minimum`
- **Production-oriented interactive workloads:** start at `single_host_recommended`
- **Burst traffic, sustained throughput, or higher concurrency:** start at `single_host_high_throughput`

Then rerun the named benchmark profiles using:

- the customer's prompt sizes
- the customer's model/provider path
- the customer's real concurrency targets
- the customer's database and observability topology

If the customer wants a scale claim, measure their shape.

## What the output means

A benchmark summary reports:

- total requests
- concurrency
- successes
- failures
- duration
- throughput in requests per second
- average latency
- p50 latency
- p95 latency
- minimum latency
- maximum latency

Interpret the metrics as follows:

| Signal | Why it matters |
| --- | --- |
| Failures | Any failure means the local stack did not complete cleanly for that run |
| Throughput | Useful for comparing the same profile across hosts or configuration changes |
| p50 latency | Midpoint experience for typical requests |
| p95 latency | Tail latency; the most useful single buyer-facing latency signal in this harness |
| Max latency | Helps identify isolated spikes or instability |
| Avg latency | Good for trend comparison, but less important than p95 for user experience |

## Result interpretation guidance

### Healthy reference-host result

A run is a healthy local reference-host data point when:

- failures are `0`
- throughput is at or above the selected profile's floor
- p95 latency is within the selected profile's band
- the host is otherwise healthy and not obviously cold-starting or resource-starved

### If one threshold is missed

- rerun once after the stack is warm
- check `make health`
- confirm the host is not under unrelated load
- confirm the same model and profile were used

### If multiple thresholds are missed or failures occur

- do **not** present the result as a healthy reference-host baseline
- investigate host saturation, Docker slowdown, gateway drift, or local dependency issues
- rerun only after the environment is stable

## Reproducibility guide

### 1. Start the repeatable offline stack

```bash
make up-offline
```

### 2. Run the lightweight default baseline

```bash
make performance-baseline
```

### 3. Run the published workload profiles

```bash
make performance-baseline PERFORMANCE_PROFILE=interactive
make performance-baseline PERFORMANCE_PROFILE=burst
make performance-baseline PERFORMANCE_PROFILE=sustained
```

### 4. Capture machine-readable output when needed

```bash
./scripts/acpctl.sh benchmark baseline --profile interactive --json
./scripts/acpctl.sh benchmark baseline --profile burst --json
./scripts/acpctl.sh benchmark baseline --profile sustained --json
```

### 5. Compare like-for-like

Only compare runs that keep these constant:

- profile or raw request shape
- model
- prompt shape
- `max_tokens`
- host tier
- gateway configuration
- database mode/topology
- observability/security sidecars that add overhead

## Honest sizing caveats

This artifact intentionally does **not** claim more than the repository currently proves.

It does **not** prove:

- customer-environment throughput without rerunning there
- WAN or provider latency
- browser or SaaS-route behavior
- Kubernetes/Helm parity
- multi-host HA behavior
- procurement-grade SLA commitments

It must also be read alongside current limitations:

- the supported host-first deployment remains constrained by the documented single-node / no-automatic-failover limitation
- provider-backed runs can differ materially from offline mock-model runs
- prompt size and token volume can dominate latency more than raw request count
- database topology, TLS posture, logging, DLP/guardrails, and observability settings can change the result
- offline token and cost estimates are approximations, not billing truth

Primary limitation reference:

- `docs/KNOWN_LIMITATIONS.md`

## Local-only benchmark outputs

Per-run benchmark outputs are intentionally local-only runtime evidence. The committed artifact in this repository is the benchmark methodology, workload-profile catalog, hardware tiers, interpretation guidance, and sizing caveats documented here.

When current numbers are needed, regenerate them locally from the current stack instead of treating old run captures as stable proof.

## How to talk about scale without overclaiming

Use this talk track:

> We publish a repeatable benchmark methodology, named workload profiles, and reference host sizing tiers for the host-first stack. Those results are useful as local capacity evidence and regression evidence, but we still rerun the same benchmark in the customer environment before making rollout or spend commitments.

## Related documents

- [ENTERPRISE_BUYER_OBJECTIONS.md](ENTERPRISE_BUYER_OBJECTIONS.md)
- [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)
- [ARTIFACTS.md](ARTIFACTS.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
