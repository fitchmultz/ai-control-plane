# Examples

This directory contains operator-facing example patterns.

## Directory Contract

- `demo/` is for the runnable local reference runtime.
- `examples/` is for curated starting points, explanation, and reusable operator patterns.
- `deploy/` contains supported host deployment assets and incubating infrastructure tracks.

## Examples

| Example | Purpose | Start here |
| --- | --- | --- |
| `basic-deployment/` | Minimal host-first deployment path | `basic-deployment/README.md` |
| `production-hardened/` | Production-oriented host workflow and validation path | `production-hardened/README.md` |
| `offline-airgap/` | Deterministic offline demonstration/runtime pattern | `offline-airgap/README.md` |
| `tls-ingress/` | TLS ingress overlay pattern | `tls-ingress/README.md` |
| `vendor-evidence/` | Sample compliance export payload for `acpctl evidence ingest` | `vendor-evidence/README.md` |
| `policy-engine/` | Sample request/response payload for `acpctl policy eval` | `policy-engine/README.md` |
| `multi-tenant-design/` | Design-only example for organization/workspace isolation and service-provider billing boundaries | `multi-tenant-design/README.md` |
| `falcon-insurance-group/` | Sanitized pilot closeout artifact set, including a case study and measurable outcomes scorecard | `falcon-insurance-group/README.md` |

## Naming Rules

- Keep example names lowercase and hyphenated.
- Keep examples free of secrets.
- Prefer README-led examples that point to canonical tracked assets instead of duplicating runtime truth.
