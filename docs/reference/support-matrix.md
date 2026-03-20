# Support Matrix

> Generated from `docs/support-matrix.yaml`. Do not edit manually.

## Supported Surfaces

| Surface | Summary | Validation |
| --- | --- | --- |
| Host-first Docker reference implementation | Minimal host-first Docker deployment with the typed operator core and truthful runtime gates. | make ci, make prod-smoke, acpctl host preflight |
| Typed operator core | acpctl typed workflows for runtime inspection, validation, DB, keys, host deployment, and evidence artifacts. | make ci-pr, make validate-acpctl-parity |
| Evidence workflows | Readiness evidence, release bundle, assessor packet, and pilot closeout artifacts generated locally from tracked inputs. | make readiness-evidence, make assessor-packet, make pilot-closeout-bundle |
| Active-passive HA failover drill evidence workflow | Customer-operated manual active-passive failover drill validation and private evidence archiving for replication readiness, fencing, promotion, customer-owned traffic cutover, and post-cutover checks. ACP does not automate PostgreSQL replication, promotion, fencing, or customer-owned DNS/load-balancer/VIP cutover. | make ha-failover-drill, acpctl host failover-drill |
| Managed browser UI overlay | Optional LibreChat overlay for governed browser access on top of the host-first runtime. | make up-ui, make librechat-health |
| DLP overlay | Optional Presidio overlay for deterministic guardrails on top of the host-first runtime. | make up-dlp, make dlp-health |
| Offline deterministic overlay | Optional mock-upstream overlay for deterministic offline demos, CI, and schema/runtime verification. | make up-offline, make ci |
| TLS ingress overlay | Optional Caddy TLS ingress overlay for host-first external client access and production OTEL ingress. | make up-tls, make cert-status, make up-production |

## Incubating Surfaces

| Surface | Summary | Validation |
| --- | --- | --- |
| Helm deployment assets | Retained in-repo under deploy/incubating for explicit internal exploration only. | explicit internal checks only |
| Terraform deployment assets | AWS-first Terraform assets are validated through explicit internal fmt, validate, and validation-only plan workflows plus AWS hardening guidance and a basic cost-estimation model. They remain under deploy/incubating and outside the supported host-first/default-CI operator surface. Azure and GCP remain incubating and unvalidated for external claims. | make tf-fmt-check, make tf-validate, make tf-plan-aws, make tf-security-check (optional) |
| Multi-tenant isolation design package | Design-only contract for future organization/workspace isolation, tenant-safe reporting, and service-provider billing boundaries. | make validate-tenant, acpctl tenant validate |

