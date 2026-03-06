# AI Control Plane - Database Targets
#
# Purpose: Database backup, restore, and inspection operations
# Responsibilities:
#   - Database backup and restore
#   - Database shell access
#   - DR drills and testing
#
# Non-scope:
#   - Does not manage database schema migrations
#   - Does not handle host-level database operations

.PHONY: db-status
db-status: ## Show database status and statistics
	@$(ACPCTL_BIN) db status

.PHONY: chargeback-report
chargeback-report: ## Generate chargeback/showback report artifacts
	@demo/scripts/chargeback_report.sh \
		$(if $(REPORT_MONTH),--month $(REPORT_MONTH),) \
		$(if $(OUTPUT_FORMAT),--format $(OUTPUT_FORMAT),) \
		$(if $(ARCHIVE_DIR),--archive-dir $(ARCHIVE_DIR),) \
		$(if $(VARIANCE_THRESHOLD),--variance-threshold $(VARIANCE_THRESHOLD),) \
		$(if $(ANOMALY_THRESHOLD),--anomaly-threshold $(ANOMALY_THRESHOLD),) \
		$(if $(filter 1 true TRUE yes YES,$(NO_FORECAST)),--no-forecast,) \
		$(if $(BUDGET_ALERT_THRESHOLD),--budget-alert-threshold $(BUDGET_ALERT_THRESHOLD),) \
		$(if $(filter 1 true TRUE yes YES,$(NOTIFY)),--notify,) \
		$(if $(filter 1 true TRUE yes YES,$(VERBOSE)),--verbose,)

.PHONY: db-backup
db-backup: ## Create database backup
	@echo '$(COLOR_BOLD)Creating database backup...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db backup \
		&& echo '$(COLOR_GREEN)✓ Database backup created$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Database backup failed$(COLOR_RESET)'; exit 1; }

.PHONY: db-restore
db-restore: ## Restore database from backup
	@echo '$(COLOR_BOLD)Restoring database from backup...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db restore \
		&& echo '$(COLOR_GREEN)✓ Database restore completed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Database restore failed$(COLOR_RESET)'; exit 1; }

.PHONY: db-shell
db-shell: ## Open database shell
	@echo '$(COLOR_BOLD)Opening database shell...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db shell

.PHONY: dr-drill
dr-drill: ## Run database DR restore drill
	@echo '$(COLOR_BOLD)Running DR restore drill...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db dr-drill \
		&& echo '$(COLOR_GREEN)✓ DR drill completed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ DR drill failed$(COLOR_RESET)'; exit 1; }

# Kubernetes/Helm Database Targets
.PHONY: helm-db-backup-exec
helm-db-backup-exec: ## Run Kubernetes DB backup job
	@echo '$(COLOR_BOLD)Running Kubernetes DB backup job...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db k8s-backup

.PHONY: helm-db-backup-verify
helm-db-backup-verify: ## Verify latest Kubernetes DB backup
	@echo '$(COLOR_BOLD)Verifying latest Kubernetes DB backup...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db k8s-backup-verify

.PHONY: helm-dr-test
helm-dr-test: ## Run Kubernetes DB restore test
	@echo '$(COLOR_BOLD)Running Kubernetes DR test...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db k8s-dr-test

.PHONY: helm-dr-drill
helm-dr-drill: ## Run Kubernetes DR drill
	@echo '$(COLOR_BOLD)Running Kubernetes DR drill...$(COLOR_RESET)'
	@$(ACPCTL_BIN) db k8s-dr-drill
