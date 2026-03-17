_acpctl_complete() {
    local cur
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"

    local commands="ci env chargeback ops status health doctor benchmark smoke completion onboard deploy validate db key host help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        ci)
            local subcmds="should-run-runtime wait"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        env)
            local subcmds="get"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        chargeback)
            local subcmds="report render payload"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        ops)
            local subcmds="report"
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
            local subcmds="release-bundle readiness-evidence pilot-closeout-bundle artifact-retention"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        validate)
            local subcmds="lint config detections siem-queries public-hygiene license supply-chain secrets-audit compose-healthchecks headers env-access security"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        db)
            local subcmds="status backup restore shell dr-drill"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        key)
            local subcmds="gen list inspect rotate revoke gen-dev gen-lead"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        host)
            local subcmds="preflight check apply install uninstall service-status service-start service-stop service-restart"
            COMPREPLY=( $(compgen -W "${subcmds}" -- "${cur}") )
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -o default -F _acpctl_complete acpctl
