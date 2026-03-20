<!--
Purpose: Record the decision to validate AWS first without changing the host-first primary surface.
Responsibilities:
  - Define what AWS-first validation means in this repository.
  - Explain why Azure/GCP are not promoted yet.
  - Preserve relationship to ADR 0001.
Scope:
  - Cloud deployment validation boundary.
Usage:
  - Reference when discussing cloud support posture.
Invariants/Assumptions:
  - Host-first remains primary.
  - Terraform remains incubating even after AWS-first validation.
-->

# 0003: AWS-first cloud validation before broader cloud promotion

- Status: Accepted
- Date: 2026-03-20

## Context

The repository contains incubating Terraform and Helm assets for AWS, Azure, and GCP, but the validated primary deployment surface remains the host-first Docker path.

Cloud claims must be evidence-backed. At the same time, buyers need more than architecture intent when AWS deployment comes up.

## Decision

Validate the cloud path deliberately with AWS first by shipping:

- explicit internal Terraform validation targets
- AWS hardening guidance for the provided IaC
- a basic AWS cost-estimation model
- a deterministic validation-only dry-run plan mode for the AWS example

Keep Terraform under `deploy/incubating/` and keep the support-matrix status incubating.

## What validation means in this repository

AWS-first validation means:

- `terraform fmt` / `terraform validate` workflows exist and are maintained
- an AWS dry-run plan path exists for `examples/aws-complete`
- validation-only plan mode works without requiring live AWS lookups
- AWS hardening guidance exists for the provided IaC
- a basic AWS cost-estimation model exists

AWS-first validation does **not** mean:

- full end-to-end apply evidence in every AWS account
- runtime smoke-test coverage in cloud CI
- Azure or GCP validation
- a change to the host-first primary support contract

## Why AWS first

AWS has the strongest in-repo Terraform path today:

- VPC
- EKS
- RDS
- IRSA
- backup bucket/lifecycle
- Helm wiring
- explicit guardrail preconditions

Validating AWS first creates a truthful evidence package without overextending into clouds that do not yet have the same documentation and validation depth.

## Why not Azure/GCP yet

Azure and GCP assets remain incubating because they do not yet have the same validated package:

- no AWS-equivalent hardening guide for those clouds
- no cloud-specific cost model
- no equivalent validation-only dry-run plan contract
- no claim-boundary update supporting them

## Consequences

- Host-first remains the primary product surface.
- Terraform stays incubating and out of default CI/help.
- External cloud claims must remain AWS-specific and narrowly defined.
- Azure/GCP remain explicitly out until they receive equivalent validation work.

## Relationship to ADR 0001

This ADR extends the evidence boundary without replacing ADR 0001.

ADR 0001 still governs the primary supported surface: host-first Docker with typed operator workflows.
