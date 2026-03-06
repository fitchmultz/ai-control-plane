_acpctl_complete() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Main commands
    local commands="ci files status doctor bridge completion deploy validate db key host demo terraform help"
    
    # Complete main command
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi
    
    # Subcommand completions
    case "${COMP_WORDS[1]}" in
        ci)
            local ci_cmds="should-run-runtime help"
            COMPREPLY=( $(compgen -W "${ci_cmds}" -- ${cur}) )
            ;;
        files)
            local file_cmds="sync-helm help"
            COMPREPLY=( $(compgen -W "${file_cmds}" -- ${cur}) )
            ;;
        deploy)
            local deploy_cmds="up down restart health logs ps up-production up-offline up-tls help"
            COMPREPLY=( $(compgen -W "${deploy_cmds}" -- ${cur}) )
            ;;
        validate)
            local validate_cmds="lint config detections siem-queries help"
            COMPREPLY=( $(compgen -W "${validate_cmds}" -- ${cur}) )
            ;;
        db)
            local db_cmds="status backup restore shell dr-drill help"
            COMPREPLY=( $(compgen -W "${db_cmds}" -- ${cur}) )
            ;;
        key)
            local key_cmds="gen revoke gen-dev gen-lead rbac-whoami rbac-roles help"
            COMPREPLY=( $(compgen -W "${key_cmds}" -- ${cur}) )
            ;;
        host)
            local host_cmds="preflight check apply install service-status upgrade-status help"
            COMPREPLY=( $(compgen -W "${host_cmds}" -- ${cur}) )
            ;;
        demo)
            local demo_cmds="scenario all preset snapshot restore status help"
            COMPREPLY=( $(compgen -W "${demo_cmds}" -- ${cur}) )
            ;;
        terraform)
            local tf_cmds="init plan apply destroy fmt validate help"
            COMPREPLY=( $(compgen -W "${tf_cmds}" -- ${cur}) )
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -o default -F _acpctl_complete acpctl
