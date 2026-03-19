# AI Control Plane - Host Deployment Targets
#
# Purpose: Host-first deployment and operations
# Responsibilities:
#   - Host preflight checks
#   - Host deployment (check/apply)
#   - Systemd service, backup-timer, and certificate lifecycle management
#
# Non-scope:
#   - Does not manage Docker containers directly
#   - Does not handle Kubernetes deployments

INVENTORY ?= deploy/ansible/inventory/hosts.yml
HA_FAILOVER_MANIFEST ?= demo/logs/recovery-inputs/ha_failover_drill.yaml

.PHONY: cert-status
cert-status: ## Check certificate lifecycle status
	@$(ACPCTL_BIN) cert check \
		$(if $(DOMAIN),--domain "$(DOMAIN)",) \
		$(if $(THRESHOLD_DAYS),--threshold-days "$(THRESHOLD_DAYS)",) \
		$(if $(CRITICAL_DAYS),--critical-days "$(CRITICAL_DAYS)",) \
		$(if $(JSON),--json,)

.PHONY: cert-renew
cert-renew: ## Trigger certificate renewal
	@$(ACPCTL_BIN) cert renew \
		$(if $(DOMAIN),--domain "$(DOMAIN)",) \
		$(if $(THRESHOLD_DAYS),--threshold-days "$(THRESHOLD_DAYS)",) \
		$(if $(filter 1 true TRUE yes YES,$(DRY_RUN)),--dry-run,) \
		$(if $(filter 1 true TRUE yes YES,$(FORCE)),--force,) \
		$(if $(OUTPUT_DIR),--output-dir "$(OUTPUT_DIR)",) \
		$(if $(JSON),--json,)

.PHONY: cert-renew-install
cert-renew-install: ## Install automatic certificate renewal timer
	@$(ACPCTL_BIN) cert renew-auto \
		--env-file "$(SECRETS_ENV_FILE)" \
		$(if $(SERVICE_USER),--service-user "$(SERVICE_USER)",) \
		$(if $(SERVICE_GROUP),--service-group "$(SERVICE_GROUP)",) \
		$(if $(ON_CALENDAR),--on-calendar "$(ON_CALENDAR)",) \
		$(if $(RANDOMIZED_DELAY),--randomized-delay "$(RANDOMIZED_DELAY)",) \
		$(if $(THRESHOLD_DAYS),--threshold-days "$(THRESHOLD_DAYS)",) \
		$(if $(JSON),--json,)

.PHONY: host-preflight
host-preflight: ## Validate host readiness
	@echo '$(COLOR_BOLD)Running host preflight checks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host preflight --secrets-env-file "$(SECRETS_ENV_FILE)"

.PHONY: host-check
host-check: ## Run declarative host preflight/check mode
	@echo '$(COLOR_BOLD)Running host check mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host check --inventory "$(INVENTORY)"

.PHONY: host-apply
host-apply: ## Run declarative host apply/converge
	@echo '$(COLOR_BOLD)Running host apply mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host apply --inventory "$(INVENTORY)"

.PHONY: ha-failover-drill
ha-failover-drill: ## Validate a customer-operated active-passive failover drill manifest
	@echo '$(COLOR_BOLD)Archiving active-passive failover drill evidence...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host failover-drill --manifest "$(HA_FAILOVER_MANIFEST)"

.PHONY: host-install
host-install: ## Install systemd service and automated backup timer
	@echo '$(COLOR_BOLD)Installing systemd service and backup timer...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host install --env-file "$(SECRETS_ENV_FILE)"

.PHONY: host-uninstall
host-uninstall: ## Uninstall systemd service
	@echo '$(COLOR_BOLD)Uninstalling systemd service...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host uninstall

.PHONY: host-service-status
host-service-status: ## Show service and backup timer status
	@$(ACPCTL_BIN) host service-status

.PHONY: host-service-start
host-service-start: ## Start service
	@$(ACPCTL_BIN) host service-start --env-file "$(SECRETS_ENV_FILE)"

.PHONY: host-service-stop
host-service-stop: ## Stop service
	@$(ACPCTL_BIN) host service-stop

.PHONY: host-service-restart
host-service-restart: ## Restart service
	@$(ACPCTL_BIN) host service-restart --env-file "$(SECRETS_ENV_FILE)"
