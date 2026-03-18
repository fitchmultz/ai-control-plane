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
        env)
            _acpctl_env
            ;;
        chargeback)
            _acpctl_chargeback
            ;;
        ops)
            _acpctl_ops
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
        cert)
            _acpctl_cert
            ;;
        upgrade)
            _acpctl_upgrade
            ;;
        host)
            _acpctl_host
            ;;
        *)
            _files
            ;;
    esac
}

_acpctl_commands() {
    local commands=(
        "ci:CI and local gate helpers"
        "env:Strict .env access helpers"
        "chargeback:Typed chargeback rendering helpers"
        "ops:Operator reporting workflows"
        "status:Aggregated system health overview"
        "health:Run service health checks"
        "doctor:Environment preflight diagnostics"
        "benchmark:Lightweight local performance baseline"
        "smoke:Run truthful runtime smoke checks"
        "completion:Generate shell completion scripts"
        "onboard:Launch the guided onboarding wizard"
        "deploy:Typed evidence and artifact workflows"
        "validate:Configuration and policy validation operations"
        "db:Database backup, restore, and inspection operations"
        "key:Virtual key lifecycle operations"
        "cert:TLS certificate lifecycle operations"
        "upgrade:Plan, validate, execute, and roll back host-first upgrades"
        "host:Host-first deployment and operations"
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

_acpctl_env() {
    local subcmds=(
        "get:Read a single env key without shell execution"
    )
    _describe -t commands 'env subcommands' subcmds "$@"
}

_acpctl_chargeback() {
    local subcmds=(
        "report:Generate canonical chargeback report artifacts"
        "render:Render canonical chargeback JSON or CSV"
        "payload:Render canonical chargeback webhook payload JSON"
    )
    _describe -t commands 'chargeback subcommands' subcmds "$@"
}

_acpctl_ops() {
    local subcmds=(
        "report:Render a canonical operator status report"
    )
    _describe -t commands 'ops subcommands' subcmds "$@"
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
        "config:Validate deployment configuration (use --production for host contract checks)"
        "detections:Validate detection rule output"
        "siem-queries:Validate SIEM query sync"
        "public-hygiene:Fail when local-only files are tracked by git"
        "license:Validate license policy structure and restricted references"
        "supply-chain:Run supply-chain policy and digest validation"
        "secrets-audit:Run deterministic tracked-file secrets audit"
        "compose-healthchecks:Validate Docker Compose healthchecks"
        "headers:Validate Go source file header policy"
        "env-access:Fail on direct environment access outside internal/config"
        "security:Run Make-composed security gate (hygiene, secrets, license, supply chain)"
    )
    _describe -t commands 'validate subcommands' subcmds "$@"
}

_acpctl_db() {
    local subcmds=(
        "status:Show database status and statistics"
        "backup:Create database backup"
        "backup-retention:Enforce backup retention policy"
        "restore:Restore embedded database from backup"
        "off-host-drill:Validate a staged off-host backup copy and emit replacement-host recovery evidence"
        "shell:Open database shell"
        "dr-drill:Create a fresh backup and verify restore into a scratch database"
    )
    _describe -t commands 'db subcommands' subcmds "$@"
}

_acpctl_key() {
    local subcmds=(
        "gen:Generate a standard virtual key"
        "list:List virtual keys"
        "inspect:Inspect a virtual key and its usage"
        "rotate:Stage rotation for a virtual key"
        "revoke:Revoke a virtual key by alias"
        "gen-dev:Generate a developer key"
        "gen-lead:Generate a team-lead key"
    )
    _describe -t commands 'key subcommands' subcmds "$@"
}

_acpctl_cert() {
    local subcmds=(
        "list:List tracked TLS certificates"
        "inspect:Inspect one certificate"
        "check:Validate certificate expiry and live TLS state"
        "renew:Trigger controlled certificate reissuance"
        "renew-auto:Install the automatic certificate renewal timer"
    )
    _describe -t commands 'cert subcommands' subcmds "$@"
}

_acpctl_upgrade() {
    local subcmds=(
        "plan:Show the explicit supported upgrade plan"
        "check:Validate the upgrade path, config migrations, and host convergence"
        "execute:Execute the supported host-first upgrade workflow"
        "rollback:Restore the pre-upgrade snapshots from a recorded upgrade run"
    )
    _describe -t commands 'upgrade subcommands' subcmds "$@"
}

_acpctl_host() {
    local subcmds=(
        "preflight:Validate host readiness"
        "check:Run declarative host preflight/check mode"
        "apply:Run declarative host apply/converge"
        "install:Install systemd service and automated backup timer"
        "uninstall:Uninstall systemd service and automated backup timer"
        "service-status:Show service and backup timer status"
        "service-start:Start the systemd service"
        "service-stop:Stop the systemd service"
        "service-restart:Restart the systemd service"
    )
    _describe -t commands 'host subcommands' subcmds "$@"
}

compdef _acpctl acpctl
