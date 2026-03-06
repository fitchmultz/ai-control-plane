# AI Control Plane - Production Deployment Targets
#
# Purpose: Production profile and TLS deployment
# Responsibilities:
#   - Start production profile with OTEL export
#   - TLS mode deployment
#   - Production health checks
#
# Non-scope:
#   - Does not handle host-first production deployment (see host.mk)

.PHONY: up-production
up-production: validate-config-production ## Start production profile with OTEL
	@echo '$(COLOR_BOLD)Starting production profile...$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)Note: Requires OTEL_EXPORTER_OTLP_ENDPOINT to be set$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && \
		OTEL_COLLECTOR_CONFIG_FILE=config.production.yaml \
		$(DOCKER_COMPOSE) --profile production up -d --timeout 120
	@echo '$(COLOR_GREEN)✓ Production services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - LiteLLM Gateway: http://127.0.0.1:$(LITELLM_PORT)'
	@echo '  - OTEL Collector: 127.0.0.1:4317 (gRPC), 127.0.0.1:4318 (HTTP)'
	@echo ''
	@echo 'Run $(COLOR_BOLD)make otel-health$(COLOR_RESET) to verify OTEL collector.'

.PHONY: prod-smoke
prod-smoke: ## Run production smoke tests
	@echo '$(COLOR_BOLD)Running production smoke tests...$(COLOR_RESET)'
	@$(ACPCTL_BIN) smoke \
		&& echo '$(COLOR_GREEN)✓ Production smoke tests passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Production smoke tests failed$(COLOR_RESET)'; exit 1; }

.PHONY: prod-smoke-local-tls
prod-smoke-local-tls: ## Run production smoke tests against local TLS
	@echo '$(COLOR_BOLD)Running production smoke tests against local TLS...$(COLOR_RESET)'
	@GATEWAY_HOST=localhost LITELLM_PORT=$(TLS_PORT) $(ACPCTL_BIN) smoke \
		&& echo '$(COLOR_GREEN)✓ Local TLS smoke tests passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Local TLS smoke tests failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-config-production
validate-config-production: ## Validate production configuration
	@echo '$(COLOR_BOLD)Validating production configuration...$(COLOR_RESET)'
	@echo '  Secrets file: $(SECRETS_ENV_FILE)'
	@$(ACPCTL_BIN) validate config \
		&& echo '$(COLOR_GREEN)✓ Production configuration validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Production configuration validation failed$(COLOR_RESET)'; exit 1; }

# TLS Mode Targets
.PHONY: up-tls
up-tls: ## Start TLS mode services
	@echo '$(COLOR_BOLD)Starting TLS mode services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && \
		$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.tls.yml up -d
	@echo '$(COLOR_GREEN)✓ TLS services started$(COLOR_RESET)'
	@echo ''
	@echo 'Services:'
	@echo '  - LiteLLM Gateway: https://localhost:$(TLS_PORT)'

.PHONY: down-tls
down-tls: ## Stop TLS mode services
	@echo '$(COLOR_BOLD)Stopping TLS mode services...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && \
		$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.tls.yml down
	@echo '$(COLOR_GREEN)✓ TLS services stopped$(COLOR_RESET)'

.PHONY: restart-tls
restart-tls: down-tls up-tls ## Restart TLS mode services

.PHONY: tls-health
tls-health: ## Run TLS health checks
	@echo '$(COLOR_BOLD)Running TLS health checks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) health \
		&& echo '$(COLOR_GREEN)✓ TLS health checks passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ TLS health checks failed$(COLOR_RESET)'; exit 1; }

.PHONY: tls-logs
tls-logs: ## Tail TLS mode logs
	@cd $(COMPOSE_DIR) && \
		$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.tls.yml logs -f

.PHONY: tls-verify
tls-verify: ## Verify TLS token logging safeguards
	@echo '$(COLOR_BOLD)Verifying TLS token logging safeguards...$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)⚠ Automated tls-verify is not included in this public snapshot$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)  Manual check: confirm proxy/service logs do not include Authorization headers$(COLOR_RESET)'
	@exit 2

# OTEL Targets
.PHONY: otel-health
otel-health: ## Check OTEL collector health
	@echo '$(COLOR_BOLD)Checking OTEL collector health...$(COLOR_RESET)'
	@curl -fsS http://localhost:13133/ >/dev/null 2>&1 \
		&& echo '$(COLOR_GREEN)✓ OTEL collector is healthy$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ OTEL collector is not responding$(COLOR_RESET)'; exit 1; }
