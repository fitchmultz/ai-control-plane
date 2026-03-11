# AI Control Plane - Lint and Validation Targets
#
# Purpose: Run linters and validation checks
# Responsibilities:
#   - Shell script linting (shellcheck)
#   - YAML validation (yamllint)
#   - Docker Compose config validation
#   - SIEM query sync validation
#   - Secrets audit
#   - Deployment configuration validation
#   - Supported-surface deployment validation only
#
# Non-scope:
#   - Does not fix issues automatically
#   - Does not modify source files

.PHONY: lint
lint: ## Run linters (shellcheck, YAML validation, docker-compose config, SIEM sync validation, secrets audit)
	@echo '$(COLOR_BOLD)Running linters...$(COLOR_RESET)'
	@$(MAKE) --silent lint-shell
	@$(MAKE) --silent lint-yaml
	@$(MAKE) --silent lint-go-headers
	@$(MAKE) --silent lint-env-access
	@$(MAKE) --silent validate-acpctl-parity
	@$(MAKE) --silent lint-compose
	@$(MAKE) --silent lint-healthchecks
	@$(MAKE) --silent lint-siem
	@$(MAKE) --silent lint-secrets
	@$(MAKE) --silent lint-config
	@echo '$(COLOR_GREEN)✓ All lint checks passed$(COLOR_RESET)'

.PHONY: lint-shell
lint-shell: ## Run shellcheck on all shell scripts
	@if ! command -v shellcheck >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ shellcheck not installed - required for make lint / make ci$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if [ -z '$(SHELLCHECK_FILES)' ]; then \
		echo '$(COLOR_RED)✗ Could not determine shell script list$(COLOR_RESET)'; \
		exit 2; \
	fi
	@shellcheck --severity=error $(SHELLCHECK_FILES) \
		&& echo '$(COLOR_GREEN)✓ Shell script lint passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Shell script lint failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-yaml
lint-yaml: ## Run yamllint on configuration files
	@if ! command -v yamllint >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ yamllint not installed - required for make lint / make ci$(COLOR_RESET)'; \
		exit 2; \
	fi
	@yamllint $(COMPOSE_DIR)/config/litellm.yaml \
		$(COMPOSE_DIR)/config/litellm-dlp.yaml \
		$(COMPOSE_DIR)/config/litellm-offline.yaml \
		$(COMPOSE_DIR)/config/model_catalog.yaml \
		$(COMPOSE_DIR)/config/detection_rules.yaml \
		$(COMPOSE_DIR)/config/librechat/librechat.yaml \
		&& echo '$(COLOR_GREEN)✓ YAML lint passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ YAML lint failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-go-headers
lint-go-headers: ## Validate Go source file purpose headers
	@echo '$(COLOR_BOLD)Validating Go file headers...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate headers \
		&& echo '$(COLOR_GREEN)✓ Go header policy passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Go header policy failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-env-access
lint-env-access: ## Fail on direct environment access outside internal/config
	@echo '$(COLOR_BOLD)Validating direct environment access policy...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate env-access \
		&& echo '$(COLOR_GREEN)✓ Environment access policy passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Environment access policy failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-compose
lint-compose: ## Validate Docker Compose configurations
	@if docker compose version >/dev/null 2>&1 || command -v docker-compose >/dev/null 2>&1; then \
		$(DOCKER_COMPOSE_PROJECT) -f $(COMPOSE_DIR)/docker-compose.yml config >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Docker Compose config is valid$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Docker Compose config is invalid$(COLOR_RESET)'; exit 1; }; \
		$(DOCKER_COMPOSE_PROJECT) -f $(COMPOSE_DIR)/docker-compose.yml -f $(COMPOSE_DIR)/docker-compose.offline.yml config >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Docker Compose offline config is valid$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Docker Compose offline config is invalid$(COLOR_RESET)'; exit 1; }; \
		$(DOCKER_COMPOSE_PROJECT) -f $(COMPOSE_DIR)/docker-compose.yml -f $(COMPOSE_DIR)/docker-compose.ui.yml config >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Docker Compose UI overlay config is valid$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Docker Compose UI overlay config is invalid$(COLOR_RESET)'; exit 1; }; \
		$(DOCKER_COMPOSE_PROJECT) -f $(COMPOSE_DIR)/docker-compose.yml -f $(COMPOSE_DIR)/docker-compose.dlp.yml config >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Docker Compose DLP overlay config is valid$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Docker Compose DLP overlay config is invalid$(COLOR_RESET)'; exit 1; }; \
		$(DOCKER_COMPOSE_PROJECT) -f $(COMPOSE_DIR)/docker-compose.yml \
			-f $(COMPOSE_DIR)/docker-compose.tls.yml config >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Docker Compose TLS overlay config is valid$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Docker Compose TLS overlay config is invalid$(COLOR_RESET)'; exit 1; }; \
	else \
		echo '$(COLOR_YELLOW)⚠ Docker Compose not available - skipping docker-compose config validation$(COLOR_RESET)'; \
	fi

.PHONY: lint-healthchecks
lint-healthchecks: ## Validate Docker Compose healthcheck configuration
	@echo '$(COLOR_BOLD)Validating healthcheck configuration...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate compose-healthchecks \
		&& echo '$(COLOR_GREEN)✓ Docker Compose healthcheck validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Docker Compose healthcheck validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-siem
lint-siem: ## Validate SIEM query synchronization
	@echo '$(COLOR_BOLD)Validating SIEM query sync...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate siem-queries \
		&& echo '$(COLOR_GREEN)✓ SIEM sync validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ SIEM sync validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: lint-secrets
lint-secrets: ## Run secrets and token leak audit
	@echo '$(COLOR_BOLD)Running secrets audit...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate secrets-audit \
		&& echo '$(COLOR_GREEN)✓ Secrets audit passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Secrets audit failed - leaked credentials detected$(COLOR_RESET)'; exit 1; }

.PHONY: lint-config
lint-config: ## Validate deployment configuration
	@echo '$(COLOR_BOLD)Validating deployment configuration...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate config \
		&& echo '$(COLOR_GREEN)✓ Configuration validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Configuration validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: format
format: ## Format shell scripts with shfmt
	@echo '$(COLOR_BOLD)Formatting shell scripts...$(COLOR_RESET)'
	@if command -v shfmt >/dev/null 2>&1; then \
		shfmt -w -i 4 $(SHELLCHECK_FILES) \
			&& echo '$(COLOR_GREEN)✓ Shell scripts formatted$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Shell script formatting failed$(COLOR_RESET)'; exit 1; }; \
	else \
		echo '$(COLOR_YELLOW)⚠ shfmt not installed - skipping format$(COLOR_RESET)'; \
		echo '$(COLOR_YELLOW)  Install shfmt: https://github.com/mvdan/sh$(COLOR_RESET)'; \
	fi

.PHONY: format-check
format-check: ## Fail when shell scripts are not formatted with shfmt
	@echo '$(COLOR_BOLD)Checking shell script formatting...$(COLOR_RESET)'
	@if command -v shfmt >/dev/null 2>&1; then \
		shfmt -d -i 4 $(SHELLCHECK_FILES) >/dev/null \
			&& echo '$(COLOR_GREEN)✓ Shell formatting is current$(COLOR_RESET)' \
			|| { echo '$(COLOR_RED)✗ Shell formatting drift detected; run make format$(COLOR_RESET)'; exit 1; }; \
	else \
		echo '$(COLOR_YELLOW)⚠ shfmt not installed - skipping format check$(COLOR_RESET)'; \
		echo '$(COLOR_YELLOW)  Install shfmt: https://github.com/mvdan/sh$(COLOR_RESET)'; \
	fi

.PHONY: type-check
type-check: ## Run Go static/type checks via go test
	@echo '$(COLOR_BOLD)Running Go type checks...$(COLOR_RESET)'
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo '$(COLOR_YELLOW)⚠ Go not installed - skipping type-check$(COLOR_RESET)'; \
		exit 0; \
	fi
	@$(GO) build ./... \
		&& echo '$(COLOR_GREEN)✓ Go build passed (type check)$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Go build failed$(COLOR_RESET)'; exit 1; }
	@$(GO) test ./... -run '^$$' >/dev/null 2>&1 \
		&& echo '$(COLOR_GREEN)✓ Go test compile passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Go test compile failed$(COLOR_RESET)'; exit 1; }
