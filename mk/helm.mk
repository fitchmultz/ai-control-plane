# AI Control Plane - Helm/Kubernetes Targets
#
# Purpose: Helm chart validation and Kubernetes operations
# Responsibilities:
#   - Helm chart validation
#   - Helm smoke validation gates
#   - Kubernetes resource management
#
# Non-scope:
#   - Does not deploy to Kubernetes clusters
#   - Does not manage Helm releases

.PHONY: helm-validate
helm-validate: ## Validate Helm chart
	@echo '$(COLOR_BOLD)Validating Helm chart...$(COLOR_RESET)'
	@$(ACPCTL_BIN) helm validate \
		&& echo '$(COLOR_GREEN)✓ Helm chart validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Helm chart validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: helm-smoke
helm-smoke: ## Run truthful Helm smoke validation
	@echo '$(COLOR_BOLD)Running Helm smoke checks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) helm smoke \
		&& echo '$(COLOR_GREEN)✓ Helm smoke checks passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Helm smoke checks failed$(COLOR_RESET)'; exit 1; }

.PHONY: helm-template
helm-template: ## Generate Helm templates
	@echo '$(COLOR_BOLD)Generating Helm templates...$(COLOR_RESET)'
	@helm template ai-control-plane deploy/helm/ai-control-plane \
		&& echo '$(COLOR_GREEN)✓ Helm template generation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Helm template generation failed$(COLOR_RESET)'; exit 1; }

.PHONY: helm-lint
helm-lint: ## Run Helm lint
	@echo '$(COLOR_BOLD)Running Helm lint...$(COLOR_RESET)'
	@helm lint deploy/helm/ai-control-plane \
		&& echo '$(COLOR_GREEN)✓ Helm lint passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Helm lint failed$(COLOR_RESET)'; exit 1; }

.PHONY: helm-deps
helm-deps: ## Update Helm dependencies
	@echo '$(COLOR_BOLD)Updating Helm dependencies...$(COLOR_RESET)'
	@helm dependency update deploy/helm/ai-control-plane \
		&& echo '$(COLOR_GREEN)✓ Helm dependencies updated$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Helm dependency update failed$(COLOR_RESET)'; exit 1; }
