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
up: validate-config validate-librechat-config ## Start default services
	@echo '$(COLOR_BOLD)Starting services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE) $(COMPOSE_DB_PROFILE) up -d
	@echo '$(COLOR_GREEN)✓ Services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - LiteLLM Gateway: http://127.0.0.1:$(LITELLM_PORT)'
	@echo ''
	@echo 'Run $(COLOR_BOLD)make health$(COLOR_RESET) to verify services.'

.PHONY: up-core
up-core: validate-config ## Start LiteLLM core services only (no managed web UI)
	@echo '$(COLOR_BOLD)Starting LiteLLM core services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE) $(COMPOSE_DB_PROFILE) up -d \
		$(if $(filter embedded,$(DB_MODE)),postgres,) presidio-analyzer presidio-anonymizer litellm
	@echo '$(COLOR_GREEN)✓ LiteLLM core services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - LiteLLM Gateway: http://127.0.0.1:$(LITELLM_PORT)'
	@echo '  - PostgreSQL: internal (embedded mode only)'
	@echo ''
	@echo 'Run $(COLOR_BOLD)make health$(COLOR_RESET) to verify services.'

.PHONY: librechat-up
librechat-up: validate-config validate-librechat-config ## Start managed LibreChat services
	@echo '$(COLOR_BOLD)Starting managed LibreChat services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE) $(COMPOSE_DB_PROFILE) up -d librechat
	@echo '$(COLOR_GREEN)✓ LibreChat services started$(COLOR_RESET)'
	@echo 'Open http://127.0.0.1:$(LIBRECHAT_PORT)'

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
	@cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE) $(COMPOSE_DB_PROFILE) down
	@echo '$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)'

.PHONY: restart
restart: down up ## Restart default services

.PHONY: ps
ps: ## Show running services
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE) ps

.PHONY: logs
logs: ## Tail service logs
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE) logs -f

.PHONY: logs-litellm
logs-litellm: ## Tail LiteLLM logs only
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE) logs -f litellm

.PHONY: logs-db
logs-db: ## Tail database logs only
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE) logs -f postgres

.PHONY: health
health: ## Run service health checks
	@echo '$(COLOR_BOLD)Running health checks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) health \
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
update: ## Update dependencies (pull latest Docker images)
	@echo '$(COLOR_BOLD)Updating dependencies...$(COLOR_RESET)'
	@if ! command -v docker >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker not available$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if ! docker compose version >/dev/null 2>&1 && ! command -v docker-compose >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker Compose not available$(COLOR_RESET)'; \
		exit 2; \
	fi
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE) pull \
		&& echo '$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)' \
		|| { exit_code=$$?; echo '$(COLOR_RED)✗ Docker image pull failed$(COLOR_RESET)'; exit $$exit_code; }
