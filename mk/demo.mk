# AI Control Plane - Demo Scenario Targets
#
# Purpose: Demo scenario, preset, and snapshot operations
# Responsibilities:
#   - Run demo scenarios
#   - Manage demo presets
#   - Create and restore snapshots
#
# Non-scope:
#   - Does not manage production deployments
#   - Does not handle host operations

.PHONY: demo-scenario
demo-scenario: ## Run a specific demo scenario
	@$(ACPCTL_BIN) demo scenario SCENARIO=$(SCENARIO)

.PHONY: demo-all
demo-all: ## Run all demo scenarios
	@$(ACPCTL_BIN) demo all

.PHONY: demo-help
demo-help: ## Show demo scenario help
	@$(ACPCTL_BIN) demo help

.PHONY: demo-preset
demo-preset: ## Run a named demo preset
	@$(ACPCTL_BIN) demo preset PRESET=$(PRESET)

.PHONY: demo-preset-list
demo-preset-list: ## List available demo presets
	@$(ACPCTL_BIN) demo preset-list

.PHONY: demo-snapshot
demo-snapshot: ## Create a named demo snapshot
	@$(ACPCTL_BIN) demo snapshot NAME=$(NAME)

.PHONY: demo-restore
demo-restore: ## Restore a named demo snapshot
	@$(ACPCTL_BIN) demo restore NAME=$(NAME)

.PHONY: demo-reset
demo-reset: ## Reset demo state to baseline
	@$(ACPCTL_BIN) demo reset

.PHONY: demo-status
demo-status: ## Show current demo state
	@$(ACPCTL_BIN) demo status

.PHONY: demo-snapshots
demo-snapshots: ## List demo snapshots
	@$(ACPCTL_BIN) demo snapshots
