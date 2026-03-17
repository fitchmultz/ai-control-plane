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

.PHONY: validate-runtime-overlays
validate-runtime-overlays: ## Validate supported runtime overlay selection
	@invalid='$(filter-out tls ui dlp offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS)))))'; \
	if [ -n "$$invalid" ]; then \
		echo '$(COLOR_RED)✗ Unsupported ACP_RUNTIME_OVERLAYS values:$(COLOR_RESET)' "$$invalid"; \
		echo 'Supported overlays: tls, ui, dlp, offline'; \
		exit 64; \
	fi

.PHONY: up-runtime
up-runtime: hardened-images-build validate-config validate-runtime-overlays ## Start runtime using the canonical overlay engine
	@echo '$(COLOR_BOLD)Starting runtime overlays: $(if $(strip $(ACP_RUNTIME_OVERLAYS)),$(ACP_RUNTIME_OVERLAYS),base)$(COLOR_RESET)'
	@if printf '%s' '$(strip $(subst $(space),,$(ACP_RUNTIME_OVERLAYS)))' | grep -Eq '(^|,)ui(,|$$)'; then \
		$(MAKE) --silent validate-librechat-config COMPOSE_ENV_FILE="$(COMPOSE_ENV_FILE)"; \
	fi
	@cd $(COMPOSE_DIR) && \
		ACP_DATABASE_MODE=$(DB_MODE) \
		ACP_PULL_POLICY=$(ACP_RUNTIME_PULL_POLICY) \
		ACP_RUNTIME_ENV_FILE="$(COMPOSE_ENV_FILE)" \
		LITELLM_CONFIG_FILE=$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),litellm-offline.yaml,$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),litellm-dlp.yaml,litellm.yaml)) \
		LITELLM_IMAGE=$(ACP_RUNTIME_LITELLM_IMAGE) \
		LIBRECHAT_IMAGE=$(ACP_RUNTIME_LIBRECHAT_IMAGE) \
		OTEL_COLLECTOR_CONFIG_FILE=$(ACP_RUNTIME_OTEL_COLLECTOR_CONFIG_FILE) \
		$(DOCKER_COMPOSE_PROJECT) \
		-f docker-compose.yml \
		$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.dlp.yml,) \
		$(if $(filter ui,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.ui.yml,) \
		$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.offline.yml,) \
		$(if $(filter tls,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.tls.yml,) \
		$(COMPOSE_DB_PROFILE) \
		$(if $(filter 1 true yes,$(ACP_RUNTIME_PRODUCTION_PROFILE)),--profile production,) \
		up -d --timeout 120 \
		$(if $(filter embedded,$(DB_MODE)),postgres,) \
		litellm \
		$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),presidio-analyzer presidio-anonymizer,) \
		$(if $(filter ui,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),librechat-mongodb librechat-meilisearch librechat,) \
		$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),mock-upstream,) \
		$(if $(filter tls,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),caddy,) \
		$(if $(filter 1 true yes,$(ACP_RUNTIME_PRODUCTION_PROFILE)),otel-collector,)
	@echo '$(COLOR_GREEN)✓ Runtime started$(COLOR_RESET)'

.PHONY: down-runtime
down-runtime: validate-runtime-overlays ## Stop runtime using the canonical overlay engine
	@echo '$(COLOR_BOLD)Stopping runtime overlays: $(if $(strip $(ACP_RUNTIME_OVERLAYS)),$(ACP_RUNTIME_OVERLAYS),base)$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && \
		ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE_PROJECT) \
		-f docker-compose.yml \
		$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.dlp.yml,) \
		$(if $(filter ui,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.ui.yml,) \
		$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.offline.yml,) \
		$(if $(filter tls,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.tls.yml,) \
		$(COMPOSE_DB_PROFILE) \
		$(if $(filter 1 true yes,$(ACP_RUNTIME_PRODUCTION_PROFILE)),--profile production,) \
		down
	@echo '$(COLOR_GREEN)✓ Runtime stopped$(COLOR_RESET)'

.PHONY: logs-runtime
logs-runtime: validate-runtime-overlays ## Tail runtime logs using the canonical overlay engine
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) \
		-f docker-compose.yml \
		$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.dlp.yml,) \
		$(if $(filter ui,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.ui.yml,) \
		$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.offline.yml,) \
		$(if $(filter tls,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.tls.yml,) \
		$(COMPOSE_DB_PROFILE) \
		$(if $(filter 1 true yes,$(ACP_RUNTIME_PRODUCTION_PROFILE)),--profile production,) \
		logs -f

.PHONY: ps-runtime
ps-runtime: validate-runtime-overlays ## Show runtime status using the canonical overlay engine
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) \
		-f docker-compose.yml \
		$(if $(filter dlp,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.dlp.yml,) \
		$(if $(filter ui,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.ui.yml,) \
		$(if $(filter offline,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.offline.yml,) \
		$(if $(filter tls,$(strip $(subst $(comma), ,$(subst $(space),,$(ACP_RUNTIME_OVERLAYS))))),-f docker-compose.tls.yml,) \
		$(COMPOSE_DB_PROFILE) \
		$(if $(filter 1 true yes,$(ACP_RUNTIME_PRODUCTION_PROFILE)),--profile production,) \
		ps

.PHONY: up
up: ## Start supported base services
	@echo '$(COLOR_BOLD)Starting services...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=
	@echo '$(COLOR_GREEN)✓ Services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - Gateway URL: $(EFFECTIVE_GATEWAY_URL)'
	@echo '  - Master key access: ./scripts/acpctl.sh env get LITELLM_MASTER_KEY'
	@echo ''
	@echo 'Run $(COLOR_BOLD)make health$(COLOR_RESET) to verify services.'

.PHONY: up-core
up-core: up ## Compatibility alias for the supported base runtime

.PHONY: up-dlp
up-dlp: ## Start base services with the DLP overlay
	@echo '$(COLOR_BOLD)Starting DLP overlay services...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=dlp
	@echo '$(COLOR_GREEN)✓ DLP overlay services started$(COLOR_RESET)'

.PHONY: up-ui
up-ui: ## Start base services with the managed UI overlay
	@echo '$(COLOR_BOLD)Starting managed UI overlay services...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=ui
	@echo '$(COLOR_GREEN)✓ Managed UI overlay services started$(COLOR_RESET)'

.PHONY: up-full
up-full: ## Start base services with DLP and managed UI overlays
	@echo '$(COLOR_BOLD)Starting full host-first stack...$(COLOR_RESET)'
	@$(MAKE) --silent up-runtime ACP_RUNTIME_OVERLAYS=ui,dlp
	@echo '$(COLOR_GREEN)✓ Full host-first stack started$(COLOR_RESET)'

.PHONY: librechat-health
librechat-health: ## Check managed UI overlay container health
	@echo '$(COLOR_BOLD)Checking managed UI overlay health...$(COLOR_RESET)'
	@failures=''; \
	missing=0; \
	for service in librechat librechat-mongodb librechat-meilisearch; do \
		container_id="$$(cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.ui.yml $(COMPOSE_DB_PROFILE) ps -q $$service)"; \
		container_id="$$(printf '%s\n' "$$container_id" | sed '/^$$/d' | tail -n 1)"; \
		if [ -z "$$container_id" ]; then \
			failures="$$failures $$service=missing"; \
			missing=1; \
			continue; \
		fi; \
		status="$$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}}' "$$container_id" 2>/dev/null || true)"; \
		if [ "$$status" != "healthy" ]; then \
			failures="$$failures $$service=$${status:-inspect-failed}"; \
		fi; \
	done; \
	if [ "$$missing" -eq 1 ]; then \
		echo '$(COLOR_RED)✗ Managed UI overlay is not active in the selected runtime:$(COLOR_RESET)'$$failures; \
		exit 1; \
	fi; \
	if [ -n "$$failures" ]; then \
		echo '$(COLOR_RED)✗ Managed UI overlay services are unhealthy:$(COLOR_RESET)'$$failures; \
		exit 1; \
	fi; \
	echo '$(COLOR_GREEN)✓ Managed UI overlay services are healthy$(COLOR_RESET)'

.PHONY: dlp-health
dlp-health: ## Check Presidio DLP overlay container health
	@echo '$(COLOR_BOLD)Checking DLP overlay health...$(COLOR_RESET)'
	@failures=''; \
	missing=0; \
	for service in presidio-analyzer presidio-anonymizer; do \
		container_id="$$(cd $(COMPOSE_DIR) && ACP_DATABASE_MODE=$(DB_MODE) $(DOCKER_COMPOSE_PROJECT) -f docker-compose.yml -f docker-compose.dlp.yml $(COMPOSE_DB_PROFILE) ps -q $$service)"; \
		container_id="$$(printf '%s\n' "$$container_id" | sed '/^$$/d' | tail -n 1)"; \
		if [ -z "$$container_id" ]; then \
			failures="$$failures $$service=missing"; \
			missing=1; \
			continue; \
		fi; \
		status="$$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}}' "$$container_id" 2>/dev/null || true)"; \
		if [ "$$status" != "healthy" ]; then \
			failures="$$failures $$service=$${status:-inspect-failed}"; \
		fi; \
	done; \
	if [ "$$missing" -eq 1 ]; then \
		echo '$(COLOR_RED)✗ DLP overlay is not active in the selected runtime:$(COLOR_RESET)'$$failures; \
		exit 1; \
	fi; \
	if [ -n "$$failures" ]; then \
		echo '$(COLOR_RED)✗ DLP overlay services are unhealthy:$(COLOR_RESET)'$$failures; \
		exit 1; \
	fi; \
	echo '$(COLOR_GREEN)✓ DLP overlay services are healthy$(COLOR_RESET)'

.PHONY: down
down: ## Stop default services
	@echo '$(COLOR_BOLD)Stopping services...$(COLOR_RESET)'
	@$(MAKE) --silent down-runtime ACP_RUNTIME_OVERLAYS=
	@echo '$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)'

.PHONY: restart
restart: down up ## Restart default services

.PHONY: ps
ps: ## Show running services
	@$(MAKE) --silent ps-runtime ACP_RUNTIME_OVERLAYS=

.PHONY: logs
logs: ## Tail service logs
	@$(MAKE) --silent logs-runtime ACP_RUNTIME_OVERLAYS=

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
	@$(ACPCTL_BIN) doctor \
		$(if $(filter 1 true TRUE yes YES,$(FIX)),--fix,) \
		$(if $(filter 1 true TRUE yes YES,$(NOTIFY)),--notify,) \
		$(if $(filter 1 true TRUE yes YES,$(WIDE)),--wide,)

.PHONY: doctor-json
doctor-json: ## Run doctor diagnostics with JSON output
	@$(ACPCTL_BIN) doctor --json \
		$(if $(filter 1 true TRUE yes YES,$(FIX)),--fix,) \
		$(if $(filter 1 true TRUE yes YES,$(NOTIFY)),--notify,) \
		$(if $(filter 1 true TRUE yes YES,$(WIDE)),--wide,)

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
