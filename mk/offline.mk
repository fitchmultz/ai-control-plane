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
up-offline: ## Start offline mode services with locally built hardened images
	@echo '$(COLOR_BOLD)Starting offline mode services...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=offline
	@echo '$(COLOR_GREEN)✓ Offline services started$(COLOR_RESET)'

.PHONY: up-offline-ci
up-offline-ci: ## Start offline mode services for CI using pinned fallback images
	@echo '$(COLOR_BOLD)Starting offline mode services for CI...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=offline ACP_RUNTIME_PULL_POLICY=missing ACP_RUNTIME_LITELLM_IMAGE=
	@echo '$(COLOR_GREEN)✓ Offline CI services started$(COLOR_RESET)'

.PHONY: down-offline
down-offline: ## Stop offline mode services
	@echo '$(COLOR_BOLD)Stopping offline mode services...$(COLOR_RESET)'
	@$(MAKE) --silent down-runtime ACP_RUNTIME_OVERLAYS=offline
	@echo '$(COLOR_GREEN)✓ Offline services stopped$(COLOR_RESET)'

.PHONY: down-offline-clean
down-offline-clean: ## Stop offline services and remove volumes/orphans (CI-slot safe teardown)
	@echo '$(COLOR_BOLD)Stopping offline mode services and removing volumes...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.offline.yml $(COMPOSE_DB_PROFILE) down -v --remove-orphans
	@echo '$(COLOR_GREEN)✓ Offline services + volumes removed$(COLOR_RESET)'

.PHONY: restart-offline
restart-offline: down-offline up-offline ## Restart offline mode services

.PHONY: logs-offline
logs-offline: ## Tail offline mode logs
	@$(MAKE) --silent logs-runtime ACP_RUNTIME_OVERLAYS=offline

.PHONY: health-offline
health-offline: ## Run offline mode health checks
	@echo '$(COLOR_BOLD)Running offline health checks...$(COLOR_RESET)'
	@GATEWAY_HOST=localhost LITELLM_PORT=$(LITELLM_PORT) $(COMPOSE_ENV_LITELLM_MASTER_KEY) $(ACPCTL_BIN) health \
		&& echo '$(COLOR_GREEN)✓ Offline health checks passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Offline health checks failed$(COLOR_RESET)'; exit 1; }
