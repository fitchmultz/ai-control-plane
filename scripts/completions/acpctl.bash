_acpctl_complete() {
    local cur
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"

    local commands="ci files env chargeback status health doctor benchmark completion onboard deploy validate db key host demo terraform helm bridge help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        ci)
            local subcmds="should-run-runtime wait"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        files)
            local subcmds="sync-helm"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        env)
            local subcmds="get"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        chargeback)
            local subcmds="render payload"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        benchmark)
            local subcmds="baseline"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        completion)
            local subcmds="bash zsh fish"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        deploy)
            local subcmds="up down restart health logs ps up-production prod-smoke up-offline down-offline health-offline up-tls down-tls tls-health helm-validate release-bundle readiness-evidence pilot-closeout-bundle artifact-retention"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        validate)
            local subcmds="lint config detections siem-queries network-contract public-hygiene license supply-chain secrets-audit compose-healthchecks security"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        db)
            local subcmds="status backup restore shell dr-drill"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        key)
            local subcmds="gen revoke gen-dev gen-lead"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        host)
            local subcmds="preflight check apply install service-status"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        demo)
            local subcmds="scenario all preset snapshot restore"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        terraform)
            local subcmds="init plan apply destroy fmt validate"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        helm)
            local subcmds="validate smoke"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        bridge)
            local subcmds="host_deploy host_install host_preflight host_upgrade_slots onboard prepare_secrets_env prod_smoke_helm prod_smoke_test release_bundle switch_claude_mode"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -o default -F _acpctl_complete acpctl
