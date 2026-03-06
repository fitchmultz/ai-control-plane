# AI Control Plane - Terraform Targets
#
# Purpose: Terraform provisioning workflow helpers
# Responsibilities:
#   - Terraform initialization, planning, and application
#   - Terraform validation and formatting
#
# Non-scope:
#   - Does not manage remote state
#   - Does not handle cloud provider authentication

.PHONY: tf-init
tf-init: ## Initialize Terraform
	@cd deploy/terraform && terraform init

.PHONY: tf-plan
tf-plan: ## Run Terraform plan
	@cd deploy/terraform && terraform plan

.PHONY: tf-apply
tf-apply: ## Run Terraform apply
	@cd deploy/terraform && terraform apply

.PHONY: tf-destroy
tf-destroy: ## Run Terraform destroy
	@cd deploy/terraform && terraform destroy

.PHONY: tf-fmt
tf-fmt: ## Format Terraform files
	@cd deploy/terraform && terraform fmt -recursive

.PHONY: tf-validate
tf-validate: ## Validate Terraform configuration
	@cd deploy/terraform && terraform validate

.PHONY: tf-validate-modules
tf-validate-modules: ## Validate all Terraform modules
	@echo '$(COLOR_BOLD)Validating Terraform modules...$(COLOR_RESET)'
	@find deploy/terraform/modules -type d -mindepth 1 -maxdepth 1 | while read -r dir; do \
		echo "Validating $$dir..."; \
		(cd "$$dir" && terraform init -backend=false && terraform validate) || exit 1; \
	done
	@echo '$(COLOR_GREEN)✓ All Terraform modules validated$(COLOR_RESET)'

.PHONY: tf-clean
tf-clean: ## Clean Terraform artifacts
	@find deploy/terraform -type d -name ".terraform" -exec rm -rf {} + 2>/dev/null || true
	@find deploy/terraform -name ".terraform.lock.hcl" -delete 2>/dev/null || true
	@echo '$(COLOR_GREEN)✓ Terraform artifacts cleaned$(COLOR_RESET)'

.PHONY: tf-output
tf-output: ## Show Terraform outputs
	@cd deploy/terraform && terraform output

.PHONY: tf-aws
tf-aws: ## Select AWS Terraform stack
	@cd deploy/terraform/stacks/aws && terraform init

.PHONY: tf-azure
tf-azure: ## Select Azure Terraform stack
	@cd deploy/terraform/stacks/azure && terraform init

.PHONY: tf-gcp
tf-gcp: ## Select GCP Terraform stack
	@cd deploy/terraform/stacks/gcp && terraform init

.PHONY: tf-docs
tf-docs: ## Generate Terraform docs
	@if command -v terraform-docs >/dev/null 2>&1; then \
		terraform-docs markdown deploy/terraform --output-file README.md; \
	else \
		echo '$(COLOR_YELLOW)⚠ terraform-docs not installed$(COLOR_RESET)'; \
	fi
