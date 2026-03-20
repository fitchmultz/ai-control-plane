<!--
Purpose: Define the truthful Terraform deployment boundary for AI Control Plane.
Responsibilities:
  - Explain what AWS-first validation means in this repository.
  - Point operators to the internal validation workflow, hardening guidance, and cost model.
  - Preserve the host-first primary support boundary.
Scope:
  - Terraform assets under deploy/incubating/.
Usage:
  - Read before using deploy/incubating/terraform/examples/aws-complete.
Invariants/Assumptions:
  - Host-first Docker remains the supported primary production surface.
  - Terraform remains an incubating surface even after AWS-first validation.
-->

# Terraform Track

This Terraform track remains an **incubating deployment surface** retained under `deploy/incubating/` for explicit internal use.

## Status

- **Host-first remains the primary supported production contract.** See [SINGLE_TENANT_PRODUCTION_CONTRACT.md](SINGLE_TENANT_PRODUCTION_CONTRACT.md) and [ADR 0001](../adr/0001-host-first-docker-as-supported-surface.md).
- **AWS is the only cloud path with an explicit validation package in this repository today.**
- **Azure and GCP remain incubating and explicitly out of validated cloud claims.**
- Terraform is **not** part of the default operator UX, public `make help`, `acpctl`, or default CI.

## What “AWS-validated” means here

In this repository, AWS-first validation means all of the following exist and are maintained:

1. explicit internal Terraform validation targets:
   - `make tf-fmt-check`
   - `make tf-validate`
   - `make tf-plan-aws`
   - `make tf-security-check` (optional)
2. an AWS hardening guide for the provided IaC:
   - [../security/AWS_CLOUD_HARDENING.md](../security/AWS_CLOUD_HARDENING.md)
3. a basic AWS cost-estimation model:
   - [AWS_COST_ESTIMATION.md](AWS_COST_ESTIMATION.md)

This does **not** mean:

- Terraform apply has been validated in every AWS account or region
- cloud runtime smoke tests are part of default CI
- Azure or GCP are validated
- the Terraform surface has replaced the host-first primary deployment contract

## Prerequisites

- Docker for the pinned Terraform container fallback, or host Terraform `>= 1.5.0`
- For `TF_AWS_PLAN_MODE=live`, valid AWS credentials for the target account/region
- Helm and kubectl only when moving beyond validation into manual incubating exploration

## Internal validation workflow

```bash
make tf-fmt-check
make tf-validate
make tf-security-check
make tf-plan-aws
```

### Notes

- `make tf-validate` runs formatting plus `terraform init -backend=false` and `terraform validate` across the incubating examples from a staged temporary tree.
- `make tf-plan-aws` defaults to **validation-only** mode so the dry-run plan can be exercised without live AWS account lookups.
- Validation-only mode scopes the plan to AWS infrastructure targets and excludes Kubernetes/Helm apply-time surfaces that depend on a live EKS control plane.
- To run the same plan flow against a named AWS account, use `TF_AWS_PLAN_MODE=live make tf-plan-aws`.
- To disable the default infra target list and attempt a broader plan, use `TF_AWS_PLAN_TARGETS=none`.
- These commands are **explicit internal workflows only**. They are intentionally out of default CI and public help.

## Quick start

```bash
cp deploy/incubating/terraform/examples/aws-complete/terraform.tfvars.example \
   deploy/incubating/terraform/examples/aws-complete/terraform.tfvars

make tf-validate
make tf-plan-aws
```

## Guidance

- Hardening checklist: [../security/AWS_CLOUD_HARDENING.md](../security/AWS_CLOUD_HARDENING.md)
- Cost model: [AWS_COST_ESTIMATION.md](AWS_COST_ESTIMATION.md)
- Incubating Terraform assets: [../../deploy/incubating/terraform/README.md](../../deploy/incubating/terraform/README.md)

## Boundary statement

Use this Terraform track only when you intentionally need the incubating AWS deployment path.

For the primary supported production path, use the host-first Docker workflow documented in:

- [../DEPLOYMENT.md](../DEPLOYMENT.md)
- [SINGLE_TENANT_PRODUCTION_CONTRACT.md](SINGLE_TENANT_PRODUCTION_CONTRACT.md)
