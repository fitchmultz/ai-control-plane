# Support Matrix

> Generated from `docs/support-matrix.yaml`. Do not edit manually.

## Supported Surfaces

| Surface | Summary | Validation |
| --- | --- | --- |
| Host-first Docker reference implementation | Minimal host-first Docker deployment with the typed operator core and truthful runtime gates. | make ci, make prod-smoke, acpctl host preflight |
| Typed operator core | acpctl typed workflows for runtime inspection, validation, DB, keys, host deployment, and evidence artifacts. | make ci-pr, make validate-acpctl-parity |
| Evidence workflows | Readiness evidence, release bundle, and pilot closeout artifacts generated locally from tracked inputs. | make readiness-evidence, make pilot-closeout-bundle |
| Managed browser UI overlay | Optional LibreChat overlay for governed browser access on top of the host-first runtime. | make up-ui, make librechat-health |
| DLP overlay | Optional Presidio overlay for deterministic guardrails on top of the host-first runtime. | make up-dlp, make dlp-health |
| Offline deterministic overlay | Optional mock-upstream overlay for deterministic offline demos, CI, and schema/runtime verification. | make up-offline, make ci |
| TLS ingress overlay | Optional Caddy TLS ingress overlay for host-first external client access and production OTEL ingress. | make up-tls, make up-production |

## Incubating Surfaces

| Surface | Summary | Validation |
| --- | --- | --- |
| Helm deployment assets | Retained in-repo under deploy/incubating for explicit internal exploration only. | explicit internal checks only |
| Terraform deployment assets | Retained in-repo under deploy/incubating for explicit internal exploration only. | explicit internal checks only |

