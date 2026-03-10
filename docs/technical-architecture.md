# Technical Architecture Overview

This document describes the production-oriented architecture of the AI Control Plane repository and the engineering decisions behind it.

## Reader Guide

If you are reviewing the repository structure itself, focus on these layers:

1. `cmd/acpctl`: public operator command surface, help generation, parsing, dispatch, and command adapters.
2. `internal/*`: typed domain logic for validation, security, runtime inspection, status, doctor, readiness, release, and closeout flows.
3. `demo/`: runnable Docker-first reference environment and local config.
4. `docs/`: scope boundaries, deployment guidance, policy docs, and evidence workflows.

## 1) System Scope

The repository is a **Docker-first reference implementation** for enterprise AI governance. It is designed to demonstrate practical deployment engineering with repeatable operations, typed tooling, policy controls, and evidence-oriented validation.

The baseline deployment target is a **single Linux host**. Kubernetes and Terraform tracks are supported as optional extensions.

## 2) Core Components

### Operator and automation surfaces

- **Makefile + `mk/*.mk`**: canonical task orchestration for install, run, test, validation, security, and release workflows.
- **`acpctl` (`cmd/acpctl`, `internal/*`)**: typed operator CLI for status, health, doctor checks, security gates, key operations, and release bundles.
- **`internal/logging` + `cmd/acpctl` workflow helpers**: shared structured workflow-event logging for typed operator flows; native command handlers emit `workflow.start|complete|failed|warn` events while final human renderers stay with commands and report packages.

### Command-platform architecture

The CLI is intentionally split so command growth does not turn into parser drift:

- `command_registry.go`: composes the root command tree
- `command_types.go`: shared command model
- `command_parse.go`: option/argument parsing
- `command_help.go`: generated help rendering
- `command_dispatch.go`: backend dispatch
- `cmd_*.go`: focused domain command definitions and adapters

This keeps the public command surface declarative and makes it easier to refactor multiple command groups consistently.

### Runtime control plane

- **LiteLLM Gateway**: central policy enforcement and model routing layer.
- **PostgreSQL**: persistent state and governance data.
- **Presidio (optional integration)**: PII/DLP enrichment path.
- **Managed UI option (LibreChat)**: browser-based governed access path.
- **Caddy / OTEL integrations**: ingress/TLS and observability support.

### Policy and contract artifacts

- **Policy configuration** in `demo/config/` (approved models, budgets/rate limits, detection rules, webhook and SIEM mappings).
- **Validation contracts** in CLI and Make targets (`lint-*`, `validate-*`, security gates, health checks).
- **`internal/validation`**: deployment-surface and profile-aware configuration validation.
- **`internal/contracts`**: governance-data contract validation for detections, SIEM mappings, and approved-model alignment.

### Deployment assets

- **Host-first Docker Compose** in `demo/` (default path).
- **Optional production tracks** under `deploy/helm`, `deploy/terraform`, and `deploy/ansible`.

## 3) Data and Control Flow

### Routed (enforceable) AI traffic

1. Client tooling (CLI/IDE/UI) sends requests to LiteLLM.
2. LiteLLM applies key auth, approved-model policy, and budget/rate controls.
3. Request is routed to upstream providers.
4. Usage metadata and operational signals are captured for audit and reporting.

### Bypass (detect/respond) traffic

Direct-to-provider usage outside the gateway is treated as a detection-and-response problem. The repository documents this boundary explicitly and provides detection-rule and evidence workflows to close the operational loop.

## 4) Operational Lifecycle

### Provision and bootstrap

- `make install` prepares local dependencies, environment template, and local CLI binary.
- `make up` / `make up-offline` starts runtime services.

### Runtime verification

- `make health` validates service endpoints.
- `make doctor` performs structured environment diagnostics and actionable remediation hints.

### Governance validation

- `make license-check` and `make supply-chain-gate` enforce policy boundaries.
- `make lint`, `make type-check`, and `make test-go` validate quality contracts.

### Release artifact process

- `make release-bundle` produces deployment bundle outputs.
- `make release-bundle-verify` verifies structure, checksums, and extraction safety.
- `make readiness-evidence` generates decision-grade readiness artifacts.
- `make pilot-closeout-bundle` assembles local pilot closeout deliverables.

## 5) Repository Design Goals

- **Typed over ad hoc**: complex operational logic belongs in Go packages, not scattered shell.
- **Truthful validation**: health, smoke, readiness, and security gates should describe what they actually verify.
- **Evidence before claims**: docs and commands make validation boundaries explicit.
- **Host-first baseline**: the mainline path is the Linux Docker deployment that this repo validates directly.

## 6) CI and Test Strategy (Resource-Aware)

The repository uses **tiered checks** to keep default validation fast while preserving deep verification paths:

- **`make ci-pr`**: PR-required deterministic checks (lint/static/policy/unit).
- **`make ci`**: full local validation gate including runtime-aware test path.
- **`make ci-nightly`**: scheduled runtime + release verification sweep.
- **`make ci-manual-heavy`**: on-demand heavyweight checks (e.g., hardened image scan).

This split keeps default contribution loops responsive while preserving high-confidence release checks.

## 7) Key Engineering Decisions and Trade-offs

### Host-first default

**Decision**: prioritize Docker-on-host deployment path.  
**Why**: fastest path to reproducible operations and easier operator onboarding.  
**Trade-off**: fewer cluster-native guarantees than Kubernetes-first designs.

### Typed operator core + shell orchestration

**Decision**: keep shell scripts thin and move complex behavior into Go packages.  
**Why**: stronger testability, clearer error handling, and maintainability.  
**Trade-off**: dual surface area (Make/shell + Go) requires strict contract discipline.

### Evidence-driven governance posture

**Decision**: document enforce-vs-detect boundaries explicitly.  
**Why**: avoids overclaiming capabilities and improves operator confidence.  
**Trade-off**: requires more explicit runbook and limitation documentation.

### Local CI as canonical development gate

**Decision**: preserve deterministic local `make` gates as source of truth.  
**Why**: reduces environment drift and makes validation straightforward.  
**Trade-off**: hosted CI wiring is optional and can vary by deployment context.

## 8) Non-Goals

- Not a turnkey managed SaaS offering.
- Not a guarantee that all AI usage is preventively enforceable.
- Not a compliance certification substitute; it is a deployment and control reference implementation.
