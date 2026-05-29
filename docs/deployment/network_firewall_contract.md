# Network Firewall Contract

**Generated:** 2026-05-29T21:45:12Z
**Schema Version:** 1

## Contract Information

| Property | Value |
|----------|-------|
| contract | network-firewall |
| owners | platform |
| description | Canonical network and firewall contract for AI Control Plane deployments. This document defines all required network flows with source, destination, protocol, port, exposure semantics, and TLS requirements. |

## Shared Responsibility Requirements

**Customer must implement:**
- Default-deny egress so only approved gateway paths can reach provider APIs.
- SWG/CASB enforcement for browser AI usage (block personal tenants, allow approved enterprise tenants).
- Endpoint/MDM enforcement to prevent local override of managed tool configuration.

**AI Control Plane provides:**
- Gateway policy patterns, deployment guidance, and validation criteria.
- Reference firewall contract artifacts and evidence pipeline integration patterns.

## Network Flows

| Flow ID | Source | Destination | Direction | Protocol | Port | Exposure | TLS Required | Justification |
|---------|--------|-------------|-----------|----------|------|----------|--------------|---------------|
| flow-caddy-http-local | client_host | caddy_proxy | ingress | tcp | 80 | localhost | No | HTTP port for ACME challenge and redirect (localhost by default, public in production) |
| flow-caddy-http-public | client_host | caddy_proxy | ingress | tcp | 80 | public | No | HTTP port for ACME challenge (public in production with automatic redirect to HTTPS) |
| flow-caddy-https-local | client_host | caddy_proxy | ingress | tcp | 443 | localhost | Yes | HTTPS port for secure gateway access (localhost by default) |
| flow-caddy-https-public | client_host | caddy_proxy | ingress | tcp | 443 | public | Yes | HTTPS port for secure gateway access (public in production deployments) |
| flow-caddy-litellm-backend | caddy_proxy | litellm_gateway | egress | tcp | 4000 | internal_only | No | Caddy reverse proxy to LiteLLM backend (internal Docker network) |
| flow-caddy-otel-backend | caddy_proxy | otel_collector | egress | tcp | 4318 | internal_only | No | Caddy authenticated /otel/* reverse proxy to the OTLP/HTTP collector endpoint for supported remote telemetry ingress |
| flow-helm-ingress-http | external_client | litellm_gateway | ingress | tcp | 80 | public | No | Kubernetes ingress HTTP endpoint (optional, redirects to HTTPS when TLS enabled) |
| flow-helm-ingress-https | external_client | litellm_gateway | ingress | tcp | 443 | public | Yes | Kubernetes ingress HTTPS endpoint (optional, for production TLS termination) |
| flow-librechat-litellm | librechat | litellm_gateway | egress | tcp | 4000 | internal_only | No | LibreChat connects to LiteLLM gateway for LLM API access (internal Docker network) |
| flow-librechat-local | client_host | librechat | ingress | tcp | 3080 | localhost | Yes | LibreChat web UI (optional managed UI overlay, localhost by default for security) |
| flow-librechat-meilisearch | librechat | librechat_meilisearch | egress | tcp | 7700 | internal_only | No | LibreChat internal Meilisearch connectivity for chat search functionality |
| flow-librechat-mongodb | librechat | librechat_mongodb | egress | tcp | 27017 | internal_only | No | LibreChat internal MongoDB connectivity for session/chat storage |
| flow-litellm-http-local | client_host | litellm_gateway | ingress | tcp | 4000 | localhost | Yes | Gateway API access for clients and health endpoints (localhost binding default for security) |
| flow-litellm-internal-helm | ingress_controller | litellm_gateway | ingress | tcp | 4000 | internal_only | Yes | Kubernetes ClusterIP service for internal cluster access (ingress handles external traffic) |
| flow-litellm-mock-upstream | litellm_gateway | mock_upstream | egress | tcp | 8080 | internal_only | No | Offline mode - LLM requests routed to mock upstream instead of external providers |
| flow-litellm-postgres-egress | litellm_gateway | postgres | egress | tcp | 5432 | internal_only | No | Database connectivity for LiteLLM schema and token storage |
| flow-litellm-presidio-analyzer | litellm_gateway | presidio_analyzer | egress | tcp | 3000 | internal_only | No | PII detection for request/response content filtering |
| flow-litellm-presidio-anonymizer | litellm_gateway | presidio_anonymizer | egress | tcp | 3000 | internal_only | No | PII anonymization/redaction for request/response content filtering |
| flow-mock-upstream-helm-clusterip | litellm_gateway | mock_upstream | ingress | tcp | 8080 | internal_only | No | Kubernetes ClusterIP service for mock upstream (internal cluster access only) |
| flow-mock-upstream-internal | litellm_gateway | mock_upstream | ingress | tcp | 8080 | internal_only | No | Mock LLM service for offline demos - returns deterministic responses (internal only) |
| flow-otel-grpc | client_host | otel_collector | ingress | tcp | 4317 | localhost | No | OpenTelemetry gRPC endpoint remains localhost-only; remote ingest must traverse an authenticated HTTPS proxy path |
| flow-otel-health | client_host | otel_collector | ingress | tcp | 13133 | localhost | No | OTel collector health check endpoint remains localhost-only for local operator verification |
| flow-otel-http | client_host | otel_collector | ingress | tcp | 4318 | localhost | No | OpenTelemetry HTTP endpoint remains localhost-only; remote ingest must traverse an authenticated HTTPS proxy path |
| flow-postgres-helm-clusterip | litellm_gateway | postgres | ingress | tcp | 5432 | internal_only | No | Kubernetes ClusterIP service for PostgreSQL (internal cluster access only) |
| flow-postgres-internal-docker | litellm_gateway | postgres | ingress | tcp | 5432 | internal_only | No | Database for LiteLLM token storage and audit logs (internal Docker network only) |
| flow-presidio-analyzer-internal | litellm_gateway | presidio_analyzer | ingress | tcp | 3000 | internal_only | No | PII analysis service - detects sensitive entities in text (internal Docker network only) |
| flow-presidio-anonymizer-internal | litellm_gateway | presidio_anonymizer | ingress | tcp | 3000 | internal_only | No | PII anonymization service - masks/redacts sensitive entities (internal Docker network only) |

---

*Total flows: 27*

## Manifest References

- `demo/docker-compose.dlp.yml:services.presidio-analyzer.expose`
- `demo/docker-compose.dlp.yml:services.presidio-anonymizer.expose`
- `demo/docker-compose.offline.yml:services.mock-upstream.expose`
- `demo/docker-compose.tls.yml:services.caddy.depends_on`
- `demo/docker-compose.tls.yml:services.caddy.environment`
- `demo/docker-compose.tls.yml:services.caddy.ports`
- `demo/docker-compose.ui.yml:services.librechat.depends_on`
- `demo/docker-compose.ui.yml:services.librechat.environment`
- `demo/docker-compose.ui.yml:services.librechat.ports`
- `demo/docker-compose.yml:services.litellm.environment`
- `demo/docker-compose.yml:services.litellm.ports`
- `demo/docker-compose.yml:services.otel-collector`
- `demo/docker-compose.yml:services.otel-collector.ports`
- `demo/docker-compose.yml:services.postgres`
- `deploy/incubating/helm/ai-control-plane/values.yaml:ingress`
- `deploy/incubating/helm/ai-control-plane/values.yaml:ingress.tls`
- `deploy/incubating/helm/ai-control-plane/values.yaml:litellm.service`
- `deploy/incubating/helm/ai-control-plane/values.yaml:mockUpstream.service`
- `deploy/incubating/helm/ai-control-plane/values.yaml:postgres.service`
