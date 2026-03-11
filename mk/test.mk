# AI Control Plane - Testing Targets
#
# Purpose: Run all test suites
# Responsibilities:
#   - Run shell/Python contract tests
#   - Run Go tests
#   - Enforce high-risk internal package coverage thresholds
#   - Run health checks as test suite
#
# Non-scope:
#   - Does not run integration tests against live services
#   - Does not replace broader runtime CI verification

GO_COVERAGE_CRITICAL_SPECS := \
	./internal/db:70 \
	./internal/contracts:90 \
	./internal/config:85 \
	./internal/catalog:90 \
	./internal/fsutil:65 \
	./internal/security:80 \
	./internal/validation:80 \
	./internal/bundle:75 \
	./internal/gateway:85 \
	./internal/chargeback:75 \
	./internal/doctor:90 \
	./internal/status:85 \
	./internal/status/collectors:90 \
	./internal/status/runner:85
GO_COVERAGE_ARTIFACT_DIR := local/coverage
GO_COVERAGE_CRITICAL_PROFILE := $(GO_COVERAGE_ARTIFACT_DIR)/critical.out

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

.PHONY: coverage-critical
coverage-critical: test-go-cover ## Enforce minimum coverage for critical high-risk internal packages

.PHONY: test-go-cover
test-go-cover: ## Run and enforce coverage for critical high-risk internal Go packages
	@echo '$(COLOR_BOLD)Running critical-package coverage gate...$(COLOR_RESET)'
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Go is required for coverage checks$(COLOR_RESET)'; \
		exit 2; \
	fi
	@set -euo pipefail; \
	mkdir -p "$(GO_COVERAGE_ARTIFACT_DIR)"; \
	total_profile="$(GO_COVERAGE_CRITICAL_PROFILE)"; \
	printf 'mode: atomic\n' > "$$total_profile"; \
	failed=0; \
	aggregate_statements=0; \
	aggregate_covered=0; \
	for spec in $(GO_COVERAGE_CRITICAL_SPECS); do \
		pkg="$${spec%:*}"; \
		threshold="$${spec##*:}"; \
		profile="$(GO_COVERAGE_ARTIFACT_DIR)/$$(printf '%s' "$$pkg" | tr '/.' '__').out"; \
		output="$$( $(GO) test -covermode=atomic -coverprofile "$$profile" "$$pkg" 2>&1 )" || { \
			printf '%s\n' "$$output"; \
			echo '$(COLOR_RED)✗ Coverage test failed for '$$pkg'$(COLOR_RESET)'; \
			exit 1; \
		}; \
		printf '%s\n' "$$output"; \
		coverage="$$( $(GO) tool cover -func "$$profile" | awk '/^total:/ {gsub("%", "", $$3); print $$3}' )"; \
		if [ -z "$$coverage" ]; then \
			echo '$(COLOR_RED)✗ Unable to determine coverage for '$$pkg'$(COLOR_RESET)'; \
			exit 1; \
		fi; \
		tail -n +2 "$$profile" >> "$$total_profile"; \
		package_totals="$$( awk 'FNR == 1 { next } { statements += $$2; if ($$3 > 0) covered += $$2 } END { printf "%d %d", statements, covered }' "$$profile" )"; \
		package_statements="$${package_totals% *}"; \
		package_covered="$${package_totals#* }"; \
		aggregate_statements=$$((aggregate_statements + package_statements)); \
		aggregate_covered=$$((aggregate_covered + package_covered)); \
		if awk -v actual="$$coverage" -v minimum="$$threshold" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then \
			echo "$(COLOR_GREEN)✓ $$pkg coverage $$coverage% >= $$threshold%$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_RED)✗ $$pkg coverage $$coverage% < $$threshold%$(COLOR_RESET)"; \
			failed=1; \
		fi; \
	done; \
	if [ "$$aggregate_statements" -eq 0 ]; then \
		echo '$(COLOR_RED)✗ Unable to determine aggregate coverage$(COLOR_RESET)'; \
		exit 1; \
	fi; \
	aggregate_percent="$$( awk -v covered="$$aggregate_covered" -v statements="$$aggregate_statements" 'BEGIN { printf "%.1f", (covered / statements) * 100 }' )"; \
	echo "total:												(statements)				$$aggregate_percent%"; \
	if [ "$$failed" -ne 0 ]; then \
		exit 1; \
	fi

.PHONY: coverage-report
coverage-report: test-go-cover ## Print detailed coverage report for critical high-risk internal packages
	@echo '$(COLOR_BOLD)Detailed coverage report: $(GO_COVERAGE_CRITICAL_PROFILE)$(COLOR_RESET)'
	@set -euo pipefail; \
	aggregate_statements=0; \
	aggregate_covered=0; \
	for spec in $(GO_COVERAGE_CRITICAL_SPECS); do \
		pkg="$${spec%:*}"; \
		profile="$(GO_COVERAGE_ARTIFACT_DIR)/$$(printf '%s' "$$pkg" | tr '/.' '__').out"; \
		echo "== $$pkg =="; \
		$(GO) tool cover -func "$$profile" | tail -n1; \
		package_totals="$$( awk 'FNR == 1 { next } { statements += $$2; if ($$3 > 0) covered += $$2 } END { printf "%d %d", statements, covered }' "$$profile" )"; \
		package_statements="$${package_totals% *}"; \
		package_covered="$${package_totals#* }"; \
		aggregate_statements=$$((aggregate_statements + package_statements)); \
		aggregate_covered=$$((aggregate_covered + package_covered)); \
	done; \
	if [ "$$aggregate_statements" -eq 0 ]; then \
		echo '$(COLOR_RED)✗ Unable to determine aggregate coverage$(COLOR_RESET)'; \
		exit 1; \
	fi; \
	aggregate_percent="$$( awk -v covered="$$aggregate_covered" -v statements="$$aggregate_statements" 'BEGIN { printf "%.1f", (covered / statements) * 100 }' )"; \
	echo "Aggregate total: $$aggregate_percent%"

.PHONY: coverage-clean
coverage-clean: ## Remove generated coverage artifacts
	@rm -rf "$(GO_COVERAGE_ARTIFACT_DIR)"
	@echo '$(COLOR_GREEN)✓ Removed coverage artifacts$(COLOR_RESET)'

.PHONY: script-tests
script-tests: ## Run all shell script tests
	@echo '$(COLOR_BOLD)Running shell script tests...$(COLOR_RESET)'
	@bash scripts/tests/acpctl_first_migration_gate_test.sh
	@bash scripts/tests/acpctl_cli_help_test.sh
	@bash scripts/tests/acpctl_cli_delegation_test.sh
	@bash scripts/tests/acpctl_cli_typed_paths_test.sh
	@bash scripts/tests/db_make_contract_test.sh
	@bash scripts/tests/dlp_health_contract_test.sh
	@bash scripts/tests/librechat_health_contract_test.sh
	@bash scripts/tests/chatgpt_login_test.sh
	@bash scripts/tests/chatgpt_auth_cache_copy_test.sh
	@bash scripts/tests/compose_slot_files_test.sh
	@bash scripts/tests/compose_slot_validation_test.sh
	@bash scripts/tests/compose_slot_isolation_test.sh
	@bash scripts/tests/make_env_scope_test.sh
	@bash scripts/tests/onboard_help_contract_test.sh
	@bash scripts/tests/onboard_export_contract_test.sh
	@bash scripts/tests/onboard_verify_mode_test.sh
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
