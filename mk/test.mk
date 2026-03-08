# AI Control Plane - Testing Targets
#
# Purpose: Run all test suites
# Responsibilities:
#   - Run shell/Python contract tests
#   - Run Go tests
#   - Run health checks as test suite
#
# Non-scope:
#   - Does not run integration tests against live services
#   - Does not generate coverage reports

.PHONY: test
test: ## Run tests (health checks are the test suite for infrastructure project)
	@echo '$(COLOR_BOLD)Running tests...$(COLOR_RESET)'
	@$(MAKE) --silent test-health
	@bash scripts/tests/supply_chain_allowlist_expiry_check_test.sh
	@$(MAKE) --silent test-go

.PHONY: test-health
test-health: ## Run health checks as test suite
	@echo 'Running health check test suite...'
	@$(COMPOSE_ENV_LITELLM_MASTER_KEY) $(ACPCTL_BIN) health \
		&& echo '$(COLOR_GREEN)✓ Health check test suite passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Health check test suite failed$(COLOR_RESET)'; exit 1; }

.PHONY: test-go
test-go: ## Run Go unit tests
	@echo 'Running Go tests...'
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo '$(COLOR_YELLOW)⚠ Go not installed - skipping Go tests$(COLOR_RESET)'; \
		exit 0; \
	fi
	@$(GO) test -v ./... \
		&& echo '$(COLOR_GREEN)✓ Go tests passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Go tests failed$(COLOR_RESET)'; exit 1; }

.PHONY: script-tests
script-tests: ## Run all shell script tests
	@echo '$(COLOR_BOLD)Running shell script tests...$(COLOR_RESET)'
	@bash scripts/tests/acpctl_first_migration_gate_test.sh
	@bash scripts/tests/acpctl_cli_contract_test.sh
	@bash scripts/tests/onboard_test.sh
	@bash scripts/tests/chatgpt_login_test.sh
	@bash scripts/tests/chatgpt_auth_cache_copy_test.sh
	@bash scripts/tests/compose_slot_config_test.sh
	@bash scripts/tests/make_env_scope_test.sh
	@bash scripts/tests/supply_chain_allowlist_expiry_check_test.sh
	@echo '$(COLOR_GREEN)✓ Shell script tests passed$(COLOR_RESET)'

.PHONY: test-detection-rules
test-detection-rules: ## Run detection rule tests
	@echo '$(COLOR_BOLD)Running detection rule tests...$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)⚠ Detection scripts migrated to Go$(COLOR_RESET)'

.PHONY: performance-baseline
performance-baseline: ## Run the local gateway performance baseline against the current stack
	@echo '$(COLOR_BOLD)Running local performance baseline...$(COLOR_RESET)'
	@set -euo pipefail; \
	key="$$( $(ACPCTL_BIN) env get --file "$(COMPOSE_ENV_FILE)" LITELLM_MASTER_KEY 2>/dev/null || true )"; \
	LITELLM_MASTER_KEY="$$key" "$(ACPCTL_BIN)" ci wait --timeout "$(PERFORMANCE_WAIT_TIMEOUT)"; \
	set -- "$(ACPCTL_BIN)" benchmark baseline \
		--gateway-url "$(PERFORMANCE_GATEWAY_URL)" \
		--model "$(PERFORMANCE_MODEL)" \
		--max-tokens "$(PERFORMANCE_MAX_TOKENS)"; \
	if [ -n "$(PERFORMANCE_PROFILE)" ]; then \
		set -- "$$@" --profile "$(PERFORMANCE_PROFILE)"; \
	else \
		set -- "$$@" --requests "$(PERFORMANCE_REQUESTS)" --concurrency "$(PERFORMANCE_CONCURRENCY)"; \
	fi; \
	LITELLM_MASTER_KEY="$$key" "$$@"
	@echo '$(COLOR_GREEN)✓ Performance baseline complete$(COLOR_RESET)'
