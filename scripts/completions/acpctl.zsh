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
        status)
            _arguments \
                '--json[Output in JSON format]' \
                '--wide[Show extended details]' \
                '--watch[Watch mode]'
            ;;
        doctor)
            _arguments \
                '--json[Output in JSON format]' \
                '--wide[Show extended details]' \
                '--fix[Attempt auto-remediation]'
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
        "status:Aggregated system health overview"
        "doctor:Environment preflight diagnostics"
        "bridge:Execute mapped legacy script implementations"
        "completion:Generate shell completion scripts"
        "deploy:Service lifecycle and deployment helpers"
        "validate:Validation and policy checks"
        "db:Database operations"
        "key:Virtual key operations"
        "host:Host deployment and service operations"
        "demo:Demo scenarios and state operations"
        "terraform:Terraform provisioning helpers"
        "help:Show help message"
    )
    _describe -t commands 'acpctl commands' commands "$@"
}

_acpctl_ci() {
    local subcmds=(
        "should-run-runtime:Decide whether runtime checks should run"
        "help:Show help"
    )
    _describe -t commands 'ci subcommands' subcmds "$@"
}

_acpctl_files() {
    local subcmds=(
        "sync-helm:Synchronize files into Helm chart"
        "help:Show help"
    )
    _describe -t commands 'files subcommands' subcmds "$@"
}

compdef _acpctl acpctl
