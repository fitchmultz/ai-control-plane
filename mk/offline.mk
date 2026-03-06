# AI Control Plane - Offline Mode Targets
#
# Purpose: Offline mode service management
# Responsibilities:
#   - Start/stop offline mode services
#   - Offline health checks
#   - Offline demo scenarios
#
# Non-scope:
#   - Does not manage online/production services

.PHONY: up-offline
up-offline: hardened-images-build ## Start offline mode services
	@echo '$(COLOR_BOLD)Starting offline mode services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_PULL_POLICY=never LITELLM_IMAGE=ai-control-plane/litellm-hardened:local $(DOCKER_COMPOSE_PROJECT) -f docker-compose.offline.yml up -d
	@echo '$(COLOR_GREEN)✓ Offline services started$(COLOR_RESET)'

.PHONY: down-offline
down-offline: ## Stop offline mode services
	@echo '$(COLOR_BOLD)Stopping offline mode services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) -f docker-compose.offline.yml down
	@echo '$(COLOR_GREEN)✓ Offline services stopped$(COLOR_RESET)'

.PHONY: down-offline-clean
down-offline-clean: ## Stop offline services and remove volumes/orphans (CI-slot safe teardown)
	@echo '$(COLOR_BOLD)Stopping offline mode services and removing volumes...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) -f docker-compose.offline.yml down -v --remove-orphans
	@echo '$(COLOR_GREEN)✓ Offline services + volumes removed$(COLOR_RESET)'

.PHONY: restart-offline
restart-offline: down-offline up-offline ## Restart offline mode services

.PHONY: logs-offline
logs-offline: ## Tail offline mode logs
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) -f docker-compose.offline.yml logs -f

.PHONY: health-offline
health-offline: ## Run offline mode health checks
	@echo '$(COLOR_BOLD)Running offline health checks...$(COLOR_RESET)'
	@GATEWAY_HOST=localhost LITELLM_PORT=$(LITELLM_PORT) $(ACPCTL_BIN) health \
		&& echo '$(COLOR_GREEN)✓ Offline health checks passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Offline health checks failed$(COLOR_RESET)'; exit 1; }

.PHONY: demo-offline
demo-offline: ## Run offline demo scenarios (delegated)
	@$(ACPCTL_BIN) demo offline

.PHONY: demo-offline-test
demo-offline-test: ## Run quick offline demo checks (delegated)
	@$(ACPCTL_BIN) demo offline-test
