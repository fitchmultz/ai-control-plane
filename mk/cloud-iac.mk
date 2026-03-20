# AI Control Plane - Incubating Terraform Targets
#
# Purpose:
#   Provide explicit internal-only Terraform validation helpers for the
#   incubating cloud deployment assets.
#
# Responsibilities:
#   - Expose hidden make targets for Terraform fmt, validate, plan, and
#     optional tfsec checks.
#   - Keep the incubating Terraform surface out of public help and default CI.
#
# Non-scope:
#   - Does not promote Terraform into the supported operator UX.
#   - Does not add Terraform checks to make ci, make ci-pr, or public help.
#
# Invariants/Assumptions:
#   - Targets intentionally omit help annotations.
#   - Validation runs through scripts/libexec/terraform-incubating.sh.

TF_INCUBATING_RUNNER := ./scripts/libexec/terraform-incubating.sh

.PHONY: tf-fmt-check tf-validate tf-plan-aws tf-security-check

tf-fmt-check:
	@$(TF_INCUBATING_RUNNER) fmt-check

tf-validate:
	@$(TF_INCUBATING_RUNNER) validate

tf-plan-aws:
	@$(TF_INCUBATING_RUNNER) plan-aws

tf-security-check:
	@$(TF_INCUBATING_RUNNER) security-check
