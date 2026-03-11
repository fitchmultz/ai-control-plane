# AI Control Plane - Installation Targets
#
# Purpose: Set up dependencies and build tools
# Responsibilities:
#   - Install dependencies (Docker, docker-compose)
#   - Build acpctl binary
#   - Create environment files
#
# Non-scope:
#   - Does not start services
#   - Does not validate configuration

.PHONY: install
install: ## Set up dependencies (Docker, docker-compose, demo/.env) and pull latest images
	@echo '$(COLOR_BOLD)Installing dependencies...$(COLOR_RESET)'
	@$(MAKE) --silent install-env
	@if command -v $(GO) >/dev/null 2>&1; then \
		$(MAKE) --silent install-binary; \
	else \
		echo '$(COLOR_YELLOW)⚠ Go toolchain not found - skipping acpctl binary build$(COLOR_RESET)'; \
		echo '$(COLOR_YELLOW)  Install Go to build typed CLI helpers used by CI scope detection$(COLOR_RESET)'; \
	fi
	@if ! command -v docker >/dev/null 2>&1; then \
		echo '$(COLOR_YELLOW)⚠ Docker is not installed - skipping Docker setup$(COLOR_RESET)'; \
		echo '$(COLOR_YELLOW)  Docker is required for running services locally$(COLOR_RESET)'; \
	else \
		echo '$(COLOR_GREEN)✓ Docker is installed$(COLOR_RESET)'; \
		if docker compose version >/dev/null 2>&1; then \
			echo '$(COLOR_GREEN)✓ Docker Compose V2 detected$(COLOR_RESET)'; \
		elif command -v docker-compose >/dev/null 2>&1; then \
			echo '$(COLOR_GREEN)✓ Docker Compose V1 detected$(COLOR_RESET)'; \
		else \
			echo '$(COLOR_YELLOW)⚠ Docker Compose not found - skipping Docker image pull$(COLOR_RESET)'; \
		fi; \
		echo 'Pulling latest Docker images...'; \
		cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) pull || echo '$(COLOR_YELLOW)⚠ Docker image pull skipped (Docker may not be available)$(COLOR_RESET)'; \
	fi
	@echo '$(COLOR_GREEN)✓ Installation complete$(COLOR_RESET)'

.PHONY: install-env
install-env: ## Ensure demo/.env exists (creates from .env.example if missing)
	@echo '$(COLOR_BOLD)Checking environment setup...$(COLOR_RESET)'
	@if [ ! -f $(COMPOSE_DIR)/.env ]; then \
			echo 'Creating demo/.env from demo/.env.example...'; \
			cp $(COMPOSE_DIR)/.env.example $(COMPOSE_DIR)/.env; \
			echo '$(COLOR_YELLOW)⚠ demo/.env created from demo/.env.example - edit it to add your API keys$(COLOR_RESET)'; \
		else \
			echo '$(COLOR_GREEN)✓ demo/.env already exists$(COLOR_RESET)'; \
		fi

.PHONY: install-ci
install-ci: install-env install-binary ## CI-only: ensures demo/.env + acpctl binary without pulling images (deterministic)

.PHONY: install-binary
install-binary: ## Build local acpctl binary (typed CLI core) into .bin/acpctl
	@echo '$(COLOR_BOLD)Building acpctl binary...$(COLOR_RESET)'
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ go not installed - required for make install-binary$(COLOR_RESET)'; \
		exit 2; \
	fi
	@mkdir -p $(dir $(ACPCTL_BIN))
	@$(GO) build -trimpath -o $(ACPCTL_BIN) ./cmd/acpctl \
		&& echo '$(COLOR_GREEN)✓ Built $(ACPCTL_BIN)$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Failed to build acpctl binary$(COLOR_RESET)'; exit 1; }

.PHONY: completions
completions: install-binary ## Generate acpctl shell completion scripts (bash, zsh, fish)
	@mkdir -p scripts/completions
	@$(ACPCTL_BIN) completion bash > scripts/completions/acpctl.bash
	@$(ACPCTL_BIN) completion zsh > scripts/completions/acpctl.zsh
	@$(ACPCTL_BIN) completion fish > scripts/completions/acpctl.fish
	@echo "Generated completion scripts in scripts/completions/"

.PHONY: generate-reference-docs
generate-reference-docs: install-binary ## Generate tracked reference docs from typed sources of truth
	@$(ACPCTL_BIN) __generate-docs
	@echo "Generated reference docs in docs/reference/"
