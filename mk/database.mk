# AI Control Plane - Database Targets
#
# Purpose: Database backup, restore, and inspection operations
# Responsibilities:
#   - Database backup, retention, and restore
#   - Database shell access
#   - Chargeback and operator reporting workflows
#   - DR drills and testing
#
# Non-scope:
#   - Does not manage database schema migrations
#   - Does not handle host-level database operations

.PHONY: db-status
db-status: ## Show database status and statistics
	@$(ACPCTL_BIN) db status

.PHONY: db-schema-check
db-schema-check: ## Verify the pinned runtime exposes the expected LiteLLM core schema
	@echo '$(COLOR_BOLD)Validating LiteLLM core schema...$(COLOR_RESET)'
	@set -euo pipefail; \
	status_file="$$(mktemp)"; \
	trap 'rm -f "$$status_file"' EXIT; \
	$(ACPCTL_BIN) db status >"$$status_file"; \
	if ! grep -Eq 'Core tables:[[:space:]]+4/4' "$$status_file"; then \
		cat "$$status_file"; \
		echo '$(COLOR_RED)✗ LiteLLM core schema drift detected$(COLOR_RESET)'; \
		exit 1; \
	fi; \
	if ! grep -Eq 'Status:[[:space:]]+expected core tables detected' "$$status_file"; then \
		cat "$$status_file"; \
		echo '$(COLOR_RED)✗ LiteLLM core schema readiness check failed$(COLOR_RESET)'; \
		exit 1; \
	fi; \
	echo '$(COLOR_GREEN)✓ LiteLLM core schema matches expected contract$(COLOR_RESET)'

.PHONY: chargeback-report
chargeback-report: install-binary ## Generate chargeback/showback report artifacts
	@$(ACPCTL_BIN) chargeback report \
		$(if $(REPORT_MONTH),--month $(REPORT_MONTH),) \
		$(if $(OUTPUT_FORMAT),--format $(OUTPUT_FORMAT),) \
		$(if $(ARCHIVE_DIR),--archive-dir $(ARCHIVE_DIR),) \
		$(if $(VARIANCE_THRESHOLD),--variance-threshold $(VARIANCE_THRESHOLD),) \
		$(if $(ANOMALY_THRESHOLD),--anomaly-threshold $(ANOMALY_THRESHOLD),) \
		$(if $(filter 1 true TRUE yes YES,$(FORECAST)),--forecast,) \
		$(if $(filter 1 true TRUE yes YES,$(NO_FORECAST)),--no-forecast,) \
		$(if $(BUDGET_ALERT_THRESHOLD),--budget-alert-threshold $(BUDGET_ALERT_THRESHOLD),) \
		$(if $(filter 1 true TRUE yes YES,$(NOTIFY)),--notify,) \
		$(if $(filter 1 true TRUE yes YES,$(VERBOSE)),--verbose,)

.PHONY: operator-report
operator-report: install-binary ## Generate operator runtime report
	@$(ACPCTL_BIN) ops report \
		$(if $(OUTPUT_FORMAT),--format $(OUTPUT_FORMAT),) \
		$(if $(ARCHIVE_DIR),--archive-dir $(ARCHIVE_DIR),) \
		$(if $(filter 1 true TRUE yes YES,$(WIDE)),--wide,)

.PHONY: db-backup
db-backup: ## Create database backup
	@echo '$(COLOR_BOLD)Creating database backup...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db backup \
		&& echo '$(COLOR_GREEN)✓ Database backup created$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Database backup failed$(COLOR_RESET)'; exit 1; }

.PHONY: db-backup-retention
db-backup-retention: ## Enforce backup retention policy (default: check mode, KEEP=7)
	@echo '$(COLOR_BOLD)Evaluating database backup retention...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db backup-retention \
		$(if $(filter 1 true TRUE yes YES,$(APPLY)),--apply,--check) \
		$(if $(KEEP),--keep $(KEEP),)

.PHONY: db-restore
db-restore: ## Restore embedded database from backup
	@echo '$(COLOR_BOLD)Restoring database from backup...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db restore \
		&& echo '$(COLOR_GREEN)✓ Database restore completed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Database restore failed$(COLOR_RESET)'; exit 1; }

.PHONY: db-off-host-drill
db-off-host-drill: ## Validate a staged off-host backup copy and emit replacement-host recovery evidence
	@echo '$(COLOR_BOLD)Running off-host recovery drill...$(COLOR_RESET)'
	@manifest='$(OFF_HOST_RECOVERY_MANIFEST)'; \
	if [ -z "$$manifest" ]; then \
		echo 'OFF_HOST_RECOVERY_MANIFEST is required (example: demo/logs/recovery-inputs/off_host_recovery.yaml)'; \
		exit 64; \
	fi; \
	$(ACPCTL_BIN) db off-host-drill --manifest "$$manifest" \
		$(if $(OUTPUT_ROOT),--output-root $(OUTPUT_ROOT),)

.PHONY: db-shell
db-shell: ## Open database shell
	@echo '$(COLOR_BOLD)Opening database shell...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db shell

.PHONY: dr-drill
dr-drill: ## Run automated database restore verification drill
	@echo '$(COLOR_BOLD)Running automated restore verification...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db dr-drill \
		&& echo '$(COLOR_GREEN)✓ Restore verification completed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Restore verification failed$(COLOR_RESET)'; exit 1; }
