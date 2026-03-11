# Technical Architecture

The repository is intentionally narrow: one supported runtime surface, one primary operator UX, and typed packages for the workflows that need stable behavior.

## Layers

1. Operator surfaces
   - `make` is the main human interface.
   - `acpctl` is the typed implementation core for status, smoke, doctor, validate, db, key, host, onboarding, chargeback, and evidence artifacts.

2. Core services and state
   - LiteLLM gateway
   - PostgreSQL
   - Optional overlays for DLP, managed browser UI, deterministic offline mode, and TLS ingress

3. Policy, validation, and security
   - `internal/config` owns env loading.
   - `internal/policy` owns supported deployment/config scan scope.
   - `internal/validation` owns structural validation.
   - `internal/security` owns policy enforcement and repo hygiene checks.

4. Workflow and artifact packages
   - `internal/runtimeinspect`, `internal/status`, and `internal/doctor` own truthful runtime inspection.
   - `internal/readiness`, `internal/bundle`, `internal/closeout`, and `internal/artifactrun` own evidence and artifact workflows.
   - `internal/chargeback` owns typed reporting.

5. Deployment assets
   - `demo/docker-compose.yml` is the supported base runtime.
   - `demo/docker-compose.dlp.yml`, `demo/docker-compose.ui.yml`, `demo/docker-compose.offline.yml`, and `demo/docker-compose.tls.yml` are explicit overlays.
   - `deploy/ansible` and `deploy/systemd` support the host-first production path.

## Design Rules

- Keep the default runtime honest and minimal.
- Add optional behavior through overlays, not by broadening the base stack.
- Keep typed logic in Go packages and thin orchestration in Make or shell.
- Treat generated references as derived outputs, not hand-maintained truth.
