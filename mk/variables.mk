# AI Control Plane - Makefile Variables
#
# Purpose: Centralize all Makefile variables and configuration
# Responsibilities:
#   - Define all variables used across Makefile targets
#   - Auto-detect tools and environment
#   - Export essential environment variables
#
# Non-scope:
#   - Does not define targets
#   - Does not include other makefiles

# Environment variables from demo/.env
-include demo/.env
export ACP_SLOT
export LITELLM_MASTER_KEY
export LITELLM_SALT_KEY
export ACP_DATABASE_MODE
export DATABASE_URL

# Directories
COMPOSE_DIR := demo
COMPOSE_FILE := $(COMPOSE_DIR)/docker-compose.yml
COMPOSE_TLS_FILE := $(COMPOSE_DIR)/docker-compose.tls.yml

# Auto-detect Docker Compose: prefer V2 (docker compose) over V1 (docker-compose)
DOCKER_COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose")
ACP_SLOT ?= active
ACP_COMPOSE_PROJECT ?= ai-control-plane-$(ACP_SLOT)
DOCKER_COMPOSE_PROJECT := $(DOCKER_COMPOSE) --project-name $(ACP_COMPOSE_PROJECT)

# Detect local Docker socket for CI runtime
DOCKER_LOCAL_SOCKET := $(firstword $(wildcard /var/run/docker.sock /run/docker.sock))
CI_DOCKER_HOST := $(if $(DOCKER_LOCAL_SOCKET),unix://$(DOCKER_LOCAL_SOCKET),)

# Ports and networking
LITELLM_PORT := 4000
TLS_PORT := 443
LIBRECHAT_PORT ?= 3080

# Database configuration
DB_NAME ?= litellm
DB_USER ?= litellm
DB_MODE ?= $(if $(ACP_DATABASE_MODE),$(ACP_DATABASE_MODE),embedded)
COMPOSE_DB_PROFILE := $(if $(filter embedded,$(DB_MODE)),--profile embedded-db,)

# CI and testing
CI_FULL ?= 0
SCRIPT_TEST_SCOPE ?= auto
SCRIPT_TEST_JOBS ?=
OFFLINE_GATEWAY_READY_MAX_ATTEMPTS ?= 75
PERFORMANCE_GATEWAY_URL ?= http://127.0.0.1:$(LITELLM_PORT)
PERFORMANCE_MODEL ?= mock-gpt
PERFORMANCE_REQUESTS ?= 20
PERFORMANCE_CONCURRENCY ?= 2
PERFORMANCE_MAX_TOKENS ?= 32
PERFORMANCE_WAIT_TIMEOUT ?= 150
PERFORMANCE_PROFILE ?=

# Go toolchain
GO ?= go
ACPCTL_BIN ?= .bin/acpctl
GO_PACKAGES ?= ./...
GO_SOURCES := $(shell find cmd internal pkg -name '*.go' 2>/dev/null)

# Secrets Contract Configuration (RQ-0172)
SECRETS_ENV_FILE ?= /etc/ai-control-plane/secrets.env
HOST_COMPOSE_ENV_FILE ?= $(COMPOSE_DIR)/.env
SECRETS_FETCH_HOOK ?=

# Shellcheck files (tracked shell scripts; BSD/GNU portable)
SHELLCHECK_FILES := $(shell git ls-files '*.sh' 2>/dev/null || true)

# Supply chain configuration
SUPPLY_CHAIN_ALLOWLIST_WARN_DAYS ?= 45
SUPPLY_CHAIN_ALLOWLIST_FAIL_DAYS ?= 14

# Release bundle configuration
RELEASE_BUNDLE_VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
RELEASE_BUNDLE_OUT_DIR ?= demo/logs/release-bundles
RELEASE_BUNDLE_NAME ?= ai-control-plane-deploy-$(RELEASE_BUNDLE_VERSION).tar.gz
RELEASE_BUNDLE_PATH ?= $(RELEASE_BUNDLE_OUT_DIR)/$(RELEASE_BUNDLE_NAME)
READINESS_EVIDENCE_OUT_DIR ?= demo/logs/evidence
READINESS_INCLUDE_PRODUCTION ?= 0
PILOT_CLOSEOUT_OUT_DIR ?= demo/logs/pilot-closeout
PILOT_CUSTOMER ?= Falcon Insurance Group
PILOT_NAME ?= Claims Governance Pilot
PILOT_DECISION ?= EXPAND
PILOT_CHARTER ?= docs/examples/falcon-insurance-group/PILOT_CHARTER.md
PILOT_ACCEPTANCE_MEMO ?= docs/examples/falcon-insurance-group/PILOT_ACCEPTANCE_MEMO.md
PILOT_VALIDATION_CHECKLIST ?= docs/examples/falcon-insurance-group/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md
PILOT_OPERATOR_CHECKLIST ?= docs/examples/falcon-insurance-group/PILOT_OPERATOR_HANDOFF_CHECKLIST.md
PILOT_READINESS_RUN_DIR ?=
PILOT_CLOSEOUT_RUN_DIR ?=
