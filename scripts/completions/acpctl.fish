function __acpctl_complete
    # Fish completion function for acpctl
end

complete -c acpctl -f

# Main commands
complete -c acpctl -n '__fish_use_subcommand' -a "ci" -d "CI and local gate helpers"
complete -c acpctl -n '__fish_use_subcommand' -a "files" -d "File synchronization helpers"
complete -c acpctl -n '__fish_use_subcommand' -a "status" -d "System health overview"
complete -c acpctl -n '__fish_use_subcommand' -a "doctor" -d "Environment diagnostics"
complete -c acpctl -n '__fish_use_subcommand' -a "bridge" -d "Execute legacy scripts"
complete -c acpctl -n '__fish_use_subcommand' -a "completion" -d "Generate completions"
complete -c acpctl -n '__fish_use_subcommand' -a "deploy" -d "Service lifecycle"
complete -c acpctl -n '__fish_use_subcommand' -a "validate" -d "Validation checks"
complete -c acpctl -n '__fish_use_subcommand' -a "db" -d "Database operations"
complete -c acpctl -n '__fish_use_subcommand' -a "key" -d "Virtual key operations"
complete -c acpctl -n '__fish_use_subcommand' -a "host" -d "Host deployment"
complete -c acpctl -n '__fish_use_subcommand' -a "demo" -d "Demo scenarios"
complete -c acpctl -n '__fish_use_subcommand' -a "terraform" -d "Terraform helpers"

# CI subcommands
complete -c acpctl -n '__fish_seen_subcommand_from ci' -a "should-run-runtime" -d "Decide runtime checks"

# Files subcommands
complete -c acpctl -n '__fish_seen_subcommand_from files' -a "sync-helm" -d "Sync Helm files"

# Status options
complete -c acpctl -n '__fish_seen_subcommand_from status' -l json -d "JSON output"
complete -c acpctl -n '__fish_seen_subcommand_from status' -l wide -d "Extended details"
complete -c acpctl -n '__fish_seen_subcommand_from status' -l watch -d "Watch mode"

# Doctor options
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l json -d "JSON output"
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l wide -d "Extended details"
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l fix -d "Auto-remediation"
