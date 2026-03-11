# AI Control Plane - Deployment Targets
#
# Purpose: Service lifecycle management
# Responsibilities:
#   - Start/stop/restart services
#   - Health checks
#   - Log viewing
#
# Non-scope:
#   - Does not handle host-level deployment

.PHONY: up
up: hardened-images-build validate-config ## Start supported base services
	@echo '$(COLOR_BOLD)Starting services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) ACP_PULL_POLICY=never ACP_RUNTIME_ENV_FILE="$(COMPOSE_ENV_FILE)" LITELLM_CONFIG_FILE=litellm.yaml LITELLM_IMAGE=ai-control-plane/litellm-hardened:local $(DOCKER_COMPOSE_PROJECT) $(COMPOSE_DB_PROFILE) up -d $(if $(filter embedded,$(DB_MODE)),postgres,) litellm
	@echo '$(COLOR_GREEN)✓ Services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - LiteLLM Gateway: http://127.0.0.1:$(LITELLM_PORT)'
	@echo ''
	@echo 'Run $(COLOR_BOLD)make health$(COLOR_RESET) to verify services.'

.PHONY: up-core
up-core: up ## Compatibility alias for the supported base runtime

.PHONY: up-dlp
up-dlp: hardened-images-build validate-config ## Start base services with the DLP overlay
	@echo '$(COLOR_BOLD)Starting DLP overlay services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) ACP_PULL_POLICY=never ACP_RUNTIME_ENV_FILE="$(COMPOSE_ENV_FILE)" LITELLM_CONFIG_FILE=litellm-dlp.yaml LITELLM_IMAGE=ai-control-plane/litellm-hardened:local $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.dlp.yml $(COMPOSE_DB_PROFILE) up -d $(if $(filter embedded,$(DB_MODE)),postgres,) litellm presidio-analyzer presidio-anonymizer
	@echo '$(COLOR_GREEN)✓ DLP overlay services started$(COLOR_RESET)'

.PHONY: up-ui
up-ui: hardened-images-build validate-config validate-librechat-config ## Start base services with the managed UI overlay
	@echo '$(COLOR_BOLD)Starting managed UI overlay services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) ACP_PULL_POLICY=never ACP_RUNTIME_ENV_FILE="$(COMPOSE_ENV_FILE)" LITELLM_CONFIG_FILE=litellm.yaml LITELLM_IMAGE=ai-control-plane/litellm-hardened:local LIBRECHAT_IMAGE=ai-control-plane/librechat-hardened:local $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.ui.yml $(COMPOSE_DB_PROFILE) up -d $(if $(filter embedded,$(DB_MODE)),postgres,) litellm librechat-mongodb librechat-meilisearch librechat
	@echo '$(COLOR_GREEN)✓ Managed UI overlay services started$(COLOR_RESET)'

.PHONY: up-full
up-full: hardened-images-build validate-config validate-librechat-config ## Start base services with DLP and managed UI overlays
	@echo '$(COLOR_BOLD)Starting full host-first stack...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) ACP_PULL_POLICY=never ACP_RUNTIME_ENV_FILE="$(COMPOSE_ENV_FILE)" LITELLM_CONFIG_FILE=litellm-dlp.yaml LITELLM_IMAGE=ai-control-plane/litellm-hardened:local LIBRECHAT_IMAGE=ai-control-plane/librechat-hardened:local $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.dlp.yml -f docker-compose.ui.yml $(COMPOSE_DB_PROFILE) up -d $(if $(filter embedded,$(DB_MODE)),postgres,) litellm presidio-analyzer presidio-anonymizer librechat-mongodb librechat-meilisearch librechat
	@echo '$(COLOR_GREEN)✓ Full host-first stack started$(COLOR_RESET)'

.PHONY: librechat-health
librechat-health: ## Check LibreChat HTTP health endpoint
	@echo '$(COLOR_BOLD)Checking LibreChat health...$(COLOR_RESET)'
	@code=$$(curl -sS -o /dev/null -w "%{http_code}" "http://127.0.0.1:$(LIBRECHAT_PORT)/health"); \
	if [ "$$code" = "200" ]; then \
		echo '$(COLOR_GREEN)✓ LibreChat health endpoint is accessible (HTTP 200)$(COLOR_RESET)'; \
	else \
		echo '$(COLOR_RED)✗ LibreChat health endpoint returned HTTP '$$code'$(COLOR_RESET)'; \
		exit 1; \
	fi

.PHONY: down
down: ## Stop default services
	@echo '$(COLOR_BOLD)Stopping services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE_PROJECT) $(COMPOSE_DB_PROFILE) down
	@echo '$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)'

.PHONY: restart
restart: down up ## Restart default services

.PHONY: ps
ps: ## Show running services
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) ps

.PHONY: logs
logs: ## Tail service logs
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) logs -f

.PHONY: logs-litellm
logs-litellm: ## Tail LiteLLM logs only
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) logs -f litellm

.PHONY: logs-db
logs-db: ## Tail database logs only
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) logs -f postgres

.PHONY: health
health: ## Run service health checks
	@echo '$(COLOR_BOLD)Running health checks...$(COLOR_RESET)'
	@$(COMPOSE_ENV_LITELLM_MASTER_KEY) $(ACPCTL_BIN) health \
		&& echo '$(COLOR_GREEN)✓ Health checks passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Health checks failed$(COLOR_RESET)'; exit 1; }

.PHONY: status
status: ## Show service status via acpctl
	@$(ACPCTL_BIN) status

.PHONY: status-json
status-json: ## Show service status as JSON
	@$(ACPCTL_BIN) status --json

.PHONY: status-watch
status-watch: ## Watch service status (continuous monitoring)
	@$(ACPCTL_BIN) status --watch

.PHONY: doctor
doctor: ## Run environment preflight diagnostics
	@$(ACPCTL_BIN) doctor

.PHONY: doctor-json
doctor-json: ## Run doctor diagnostics with JSON output
	@$(ACPCTL_BIN) doctor --json

.PHONY: update
update: install-binary ## Update dependencies (pull base images, rebuild local hardened images, refresh generated files)
	@echo '$(COLOR_BOLD)Updating dependencies...$(COLOR_RESET)'
	@if ! command -v docker >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker not available$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if ! docker compose version >/dev/null 2>&1 && ! command -v docker-compose >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker Compose not available$(COLOR_RESET)'; \
		exit 2; \
	fi
	@$(MAKE) --silent generate
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) pull \
		&& $(MAKE) --silent hardened-images-build \
		&& echo '$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)' \
		|| { exit_code=$$?; echo '$(COLOR_RED)✗ Docker image pull failed$(COLOR_RESET)'; exit $$exit_code; }
