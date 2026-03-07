#compdef acpctl

_acpctl() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :_acpctl_commands' \
        '*:: :->args'

    case "$line[1]" in
        ci)
            _acpctl_ci
            ;;
        files)
            _acpctl_files
            ;;
        env)
            _acpctl_env
            ;;
        chargeback)
            _acpctl_chargeback
            ;;
        benchmark)
            _acpctl_benchmark
            ;;
        completion)
            _acpctl_completion
            ;;
        deploy)
            _acpctl_deploy
            ;;
        validate)
            _acpctl_validate
            ;;
        db)
            _acpctl_db
            ;;
        key)
            _acpctl_key
            ;;
        host)
            _acpctl_host
            ;;
        demo)
            _acpctl_demo
            ;;
        terraform)
            _acpctl_terraform
            ;;
        helm)
            _acpctl_helm
            ;;
        bridge)
            _acpctl_bridge
            ;;
        *)
            _files
            ;;
    esac
}

_acpctl_commands() {
    local commands=(
        "ci:CI and local gate helpers"
        "files:Typed local file synchronization helpers"
        "env:Strict .env access helpers"
        "chargeback:Typed chargeback rendering helpers"
        "status:Aggregated system health overview"
        "health:Run service health checks"
        "doctor:Environment preflight diagnostics"
        "benchmark:Lightweight local performance baseline"
        "completion:Generate shell completion scripts"
        "onboard:Configure local tools to route through the gateway"
        "deploy:Service lifecycle, release, and deployment operations"
        "validate:Configuration and policy validation operations"
        "db:Database backup, restore, and inspection operations"
        "key:Virtual key lifecycle operations"
        "host:Host-first deployment and operations"
        "demo:Demo scenario, preset, and snapshot operations"
        "terraform:Terraform provisioning workflow helpers"
        "helm:Helm chart validation and smoke tests"
        "bridge:Execute mapped legacy script implementations directly"
        "help:Show this help message"
    )
    _describe -t commands 'acpctl commands' commands "$@"
}

_acpctl_ci() {
    local subcmds=(
        "should-run-runtime:Decide whether runtime checks should run"
        "wait:Wait for services to become healthy"
    )
    _describe -t commands 'ci subcommands' subcmds "$@"
}

_acpctl_files() {
    local subcmds=(
        "sync-helm:Synchronize canonical repository files into Helm chart files/"
    )
    _describe -t commands 'files subcommands' subcmds "$@"
}

_acpctl_env() {
    local subcmds=(
        "get:Read a single env key without shell execution"
    )
    _describe -t commands 'env subcommands' subcmds "$@"
}

_acpctl_chargeback() {
    local subcmds=(
        "render:Render canonical chargeback JSON or CSV"
        "payload:Render canonical chargeback webhook payload JSON"
    )
    _describe -t commands 'chargeback subcommands' subcmds "$@"
}

_acpctl_benchmark() {
    local subcmds=(
        "baseline:Run the local gateway performance baseline"
    )
    _describe -t commands 'benchmark subcommands' subcmds "$@"
}

_acpctl_completion() {
    local subcmds=(
        "bash:Generate Bash completion script"
        "zsh:Generate Zsh completion script"
        "fish:Generate Fish completion script"
    )
    _describe -t commands 'completion subcommands' subcmds "$@"
}

_acpctl_deploy() {
    local subcmds=(
        "up:Start default services"
        "down:Stop default services"
        "restart:Restart default services"
        "health:Run service health checks"
        "logs:Tail service logs"
        "ps:Show running services"
        "up-production:Start production profile services"
        "prod-smoke:Run production smoke tests"
        "up-offline:Start offline mode services"
        "down-offline:Stop offline mode services"
        "health-offline:Run offline mode health checks"
        "up-tls:Start TLS mode services"
        "down-tls:Stop TLS mode services"
        "tls-health:Run TLS health checks"
        "helm-validate:Validate Helm chart"
        "release-bundle:Build deployment release bundle"
        "readiness-evidence:Generate and verify dated readiness evidence"
        "pilot-closeout-bundle:Assemble and verify a pilot closeout evidence bundle"
        "artifact-retention:Enforce document artifact retention policy"
    )
    _describe -t commands 'deploy subcommands' subcmds "$@"
}

_acpctl_validate() {
    local subcmds=(
        "lint:Run static validation/lint gate"
        "config:Validate demo deployment configuration"
        "detections:Validate detection rule output"
        "siem-queries:Validate SIEM query sync"
        "network-contract:Render network contract artifacts"
        "public-hygiene:Fail when local-only files are tracked by git"
        "license:Validate license policy structure and restricted references"
        "supply-chain:Run supply-chain policy and digest validation"
        "secrets-audit:Run deterministic tracked-file secrets audit"
        "compose-healthchecks:Validate Docker Compose healthchecks"
        "security:Run Make-composed security gate (hygiene, secrets, license, supply chain)"
    )
    _describe -t commands 'validate subcommands' subcmds "$@"
}

_acpctl_db() {
    local subcmds=(
        "status:Show database status and statistics"
        "backup:Create database backup"
        "restore:Restore embedded database from backup"
        "shell:Open database shell"
        "dr-drill:Run database DR restore drill"
    )
    _describe -t commands 'db subcommands' subcmds "$@"
}

_acpctl_key() {
    local subcmds=(
        "gen:Generate a standard virtual key"
        "revoke:Revoke a virtual key by alias"
        "gen-dev:Generate a developer key"
        "gen-lead:Generate a team-lead key"
    )
    _describe -t commands 'key subcommands' subcmds "$@"
}

_acpctl_host() {
    local subcmds=(
        "preflight:Validate host readiness"
        "check:Run declarative host preflight/check mode"
        "apply:Run declarative host apply/converge"
        "install:Install systemd service"
        "service-status:Show service status"
    )
    _describe -t commands 'host subcommands' subcmds "$@"
}

_acpctl_demo() {
    local subcmds=(
        "scenario:Run a specific demo scenario"
        "all:Run all demo scenarios"
        "preset:Run a named demo preset"
        "snapshot:Create a named demo snapshot"
        "restore:Restore a named demo snapshot"
    )
    _describe -t commands 'demo subcommands' subcmds "$@"
}

_acpctl_terraform() {
    local subcmds=(
        "init:Initialize Terraform"
        "plan:Run Terraform plan"
        "apply:Run Terraform apply"
        "destroy:Run Terraform destroy"
        "fmt:Format Terraform files"
        "validate:Validate Terraform configuration"
    )
    _describe -t commands 'terraform subcommands' subcmds "$@"
}

_acpctl_helm() {
    local subcmds=(
        "validate:Validate Helm chart"
        "smoke:Run Helm production smoke tests"
    )
    _describe -t commands 'helm subcommands' subcmds "$@"
}

_acpctl_bridge() {
    local subcmds=(
        "host_deploy:Host declarative deployment orchestration"
        "host_install:Systemd host service installation/management"
        "host_preflight:Host readiness preflight checks"
        "host_upgrade_slots:Slot-based host upgrade orchestration"
        "onboard:Tool onboarding workflows"
        "prepare_secrets_env:Host secrets contract refresh/sync"
        "prod_smoke_helm:Helm production smoke workflow"
        "prod_smoke_test:Runtime production smoke checks"
        "release_bundle:Deployment release bundle build/verify"
        "switch_claude_mode:Claude mode switching helper"
    )
    _describe -t commands 'bridge subcommands' subcmds "$@"
}

compdef _acpctl acpctl
