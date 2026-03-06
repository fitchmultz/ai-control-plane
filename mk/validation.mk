# AI Control Plane - Validation Targets
#
# Purpose: Configuration and policy validation
# Responsibilities:
#   - Validate detection rules
#   - Validate SIEM queries
#   - Validate network contracts
#   - Validate supply chain
#
# Non-scope:
#   - Does not fix validation issues
#   - Does not enforce policies

.PHONY: validate-config
validate-config: ## Validate deployment configuration
	@echo '$(COLOR_BOLD)Validating deployment configuration...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate config \
		&& echo '$(COLOR_GREEN)✓ Configuration validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Configuration validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-librechat-config
validate-librechat-config: ## Validate required LibreChat environment variables
	@echo '$(COLOR_BOLD)Validating LibreChat configuration...$(COLOR_RESET)'
	@if [ ! -f "$(COMPOSE_DIR)/.env" ]; then \
		echo '$(COLOR_RED)✗ Missing demo/.env$(COLOR_RESET)'; \
		echo 'Run make install-env first.'; \
		exit 1; \
	fi
	@missing=''; \
	for key in LIBRECHAT_CREDS_KEY LIBRECHAT_CREDS_IV LIBRECHAT_MEILI_MASTER_KEY LIBRECHAT_LITELLM_API_KEY JWT_SECRET JWT_REFRESH_SECRET; do \
		value=$$(grep -E "^$${key}=" "$(COMPOSE_DIR)/.env" | head -n1 | cut -d= -f2-); \
		if [ -z "$$value" ]; then \
			missing="$$missing $$key"; \
		fi; \
	done; \
	if [ -n "$$missing" ]; then \
		echo '$(COLOR_RED)✗ Missing required LibreChat env vars:$(COLOR_RESET)'$$missing; \
		echo 'Generate and set these values in demo/.env, then retry.'; \
		exit 1; \
	fi
	@echo '$(COLOR_GREEN)✓ LibreChat configuration validation passed$(COLOR_RESET)'

.PHONY: validate-detections
validate-detections: ## Validate detection rule output
	@echo '$(COLOR_BOLD)Validating detection rules...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate detections \
		&& echo '$(COLOR_GREEN)✓ Detection validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Detection validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: detection
detection: validate-detections ## Compatibility alias for validate-detections

.PHONY: detection-normalized
detection-normalized: validate-detections ## Compatibility alias for validate-detections

.PHONY: validate-siem-queries
validate-siem-queries: ## Validate SIEM query sync
	@echo '$(COLOR_BOLD)Validating SIEM queries...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate siem-queries \
		&& echo '$(COLOR_GREEN)✓ SIEM query validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ SIEM query validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-siem-schema
validate-siem-schema: ## Validate SIEM schema mappings
	@echo '$(COLOR_BOLD)Validating SIEM schema mappings...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate siem-queries --validate-schema \
		&& echo '$(COLOR_GREEN)✓ SIEM schema validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ SIEM schema validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: network-contract
network-contract: ## Render network contract artifacts
	@echo '$(COLOR_BOLD)Rendering network contract...$(COLOR_RESET)'
	@$(ACPCTL_BIN) render network-contract \
		&& echo '$(COLOR_GREEN)✓ Network contract rendered$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Network contract rendering failed$(COLOR_RESET)'; exit 1; }

.PHONY: network-contract-check
network-contract-check: ## Check network contract freshness
	@echo '$(COLOR_BOLD)Checking network contract freshness...$(COLOR_RESET)'
	@$(ACPCTL_BIN) render network-contract --check-freshness \
		&& echo '$(COLOR_GREEN)✓ Network contract is fresh$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Network contract is stale$(COLOR_RESET)'; exit 1; }

.PHONY: validate-network-contract
validate-network-contract: ## Validate network contract
	@echo '$(COLOR_BOLD)Validating network contract...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate network-contract \
		&& echo '$(COLOR_GREEN)✓ Network contract validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Network contract validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-compose-healthchecks
validate-compose-healthchecks: ## Validate Docker Compose healthcheck syntax
	@echo '$(COLOR_BOLD)Validating Docker Compose healthchecks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate compose-healthchecks \
		&& echo '$(COLOR_GREEN)✓ Healthcheck validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Healthcheck validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: governance-report
governance-report: ## Compatibility target: governance report generation not included in public snapshot
	@echo '$(COLOR_YELLOW)⚠ Governance scorecard generation is not included in this public snapshot$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)  Use make release-bundle + docs/release/PRESENTATION_READINESS_TRACKER.md for release evidence$(COLOR_RESET)'
	@exit 2

.PHONY: governance-report-json
governance-report-json: governance-report ## Compatibility target for JSON governance scorecard output

.PHONY: governance-report-7d
governance-report-7d: governance-report ## Compatibility target for 7-day governance scorecard output
