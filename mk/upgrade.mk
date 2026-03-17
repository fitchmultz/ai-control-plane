# AI Control Plane - Upgrade Targets
#
# Purpose: Host-first upgrade planning, execution, and rollback workflows
# Responsibilities:
#   - Plan supported upgrade paths
#   - Validate host-first upgrade preconditions and convergence
#   - Execute upgrades with tracked rollback artifacts
#   - Restore tracked rollback artifacts
#
# Non-scope:
#   - Does not define release edges itself
#   - Does not mutate VERSION or release notes

INVENTORY ?= deploy/ansible/inventory/hosts.yml
SECRETS_ENV_FILE ?= /etc/ai-control-plane/secrets.env

.PHONY: upgrade-plan
upgrade-plan: install-binary ## Show the supported upgrade plan for FROM_VERSION -> current VERSION
	@$(ACPCTL_BIN) upgrade plan \
		--from "$(FROM_VERSION)" \
		--inventory "$(INVENTORY)" \
		--env-file "$(SECRETS_ENV_FILE)"

.PHONY: upgrade-check
upgrade-check: install-binary ## Validate upgrade prerequisites and dry-run host convergence
	@$(ACPCTL_BIN) upgrade check \
		--from "$(FROM_VERSION)" \
		--inventory "$(INVENTORY)" \
		--env-file "$(SECRETS_ENV_FILE)"

.PHONY: upgrade-execute
upgrade-execute: install-binary ## Execute the upgrade from FROM_VERSION -> current VERSION
	@$(ACPCTL_BIN) upgrade execute \
		--from "$(FROM_VERSION)" \
		--inventory "$(INVENTORY)" \
		--env-file "$(SECRETS_ENV_FILE)"

.PHONY: upgrade-rollback
upgrade-rollback: install-binary ## Roll back using an existing upgrade run directory
	@$(ACPCTL_BIN) upgrade rollback \
		--run-dir "$(UPGRADE_RUN_DIR)" \
		--inventory "$(INVENTORY)" \
		--env-file "$(SECRETS_ENV_FILE)"
