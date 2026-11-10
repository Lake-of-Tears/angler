# bash completion for angler                               -*- shell-script -*-

__angler_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE:-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Homebrew on Macs have version 1.3 of bash-completion which doesn't include
# _init_completion. This is a very minimal version of that function.
__angler_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

__angler_index_of_word()
{
    local w word=$1
    shift
    index=0
    for w in "$@"; do
        [[ $w = "$word" ]] && return
        index=$((index+1))
    done
    index=-1
}

__angler_contains_word()
{
    local w word=$1; shift
    for w in "$@"; do
        [[ $w = "$word" ]] && return
    done
    return 1
}

__angler_handle_go_custom_completion()
{
    __angler_debug "${FUNCNAME[0]}: cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}"

    local shellCompDirectiveError=1
    local shellCompDirectiveNoSpace=2
    local shellCompDirectiveNoFileComp=4
    local shellCompDirectiveFilterFileExt=8
    local shellCompDirectiveFilterDirs=16

    local out requestComp lastParam lastChar comp directive args

    # Prepare the command to request completions for the program.
    # Calling ${words[0]} instead of directly angler allows handling aliases
    args=("${words[@]:1}")
    # Disable ActiveHelp which is not supported for bash completion v1
    requestComp="ANGLER_ACTIVE_HELP=0 ${words[0]} __completeNoDesc ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __angler_debug "${FUNCNAME[0]}: lastParam ${lastParam}, lastChar ${lastChar}"

    if [ -z "${cur}" ] && [ "${lastChar}" != "=" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go method.
        __angler_debug "${FUNCNAME[0]}: Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __angler_debug "${FUNCNAME[0]}: calling ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval "${requestComp}" 2>/dev/null)

    # Extract the directive integer at the very end of the output following a colon (:)
    directive=${out##*:}
    # Remove the directive
    out=${out%:*}
    if [ "${directive}" = "${out}" ]; then
        # There is not directive specified
        directive=0
    fi
    __angler_debug "${FUNCNAME[0]}: the completion directive is: ${directive}"
    __angler_debug "${FUNCNAME[0]}: the completions are: ${out}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        # Error code.  No completion.
        __angler_debug "${FUNCNAME[0]}: received error from custom completion go code"
        return
    else
        if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __angler_debug "${FUNCNAME[0]}: activating no space"
                compopt -o nospace
            fi
        fi
        if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __angler_debug "${FUNCNAME[0]}: activating no file completion"
                compopt +o default
            fi
        fi
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local fullFilter filter filteringCmd
        # Do not use quotes around the $out variable or else newline
        # characters will be kept.
        for filter in ${out}; do
            fullFilter+="$filter|"
        done

        filteringCmd="_filedir $fullFilter"
        __angler_debug "File filtering command: $filteringCmd"
        $filteringCmd
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        # Use printf to strip any trailing newline
        subdir=$(printf "%s" "${out}")
        if [ -n "$subdir" ]; then
            __angler_debug "Listing directories in $subdir"
            __angler_handle_subdirs_in_dir_flag "$subdir"
        else
            __angler_debug "Listing directories in ."
            _filedir -d
        fi
    else
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${out}" -- "$cur")
    fi
}

__angler_handle_reply()
{
    __angler_debug "${FUNCNAME[0]}"
    local comp
    case $cur in
        -*)
            if [[ $(type -t compopt) = "builtin" ]]; then
                compopt -o nospace
            fi
            local allflags
            if [ ${#must_have_one_flag[@]} -ne 0 ]; then
                allflags=("${must_have_one_flag[@]}")
            else
                allflags=("${flags[*]} ${two_word_flags[*]}")
            fi
            while IFS='' read -r comp; do
                COMPREPLY+=("$comp")
            done < <(compgen -W "${allflags[*]}" -- "$cur")
            if [[ $(type -t compopt) = "builtin" ]]; then
                [[ "${COMPREPLY[0]}" == *= ]] || compopt +o nospace
            fi

            # complete after --flag=abc
            if [[ $cur == *=* ]]; then
                if [[ $(type -t compopt) = "builtin" ]]; then
                    compopt +o nospace
                fi

                local index flag
                flag="${cur%=*}"
                __angler_index_of_word "${flag}" "${flags_with_completion[@]}"
                COMPREPLY=()
                if [[ ${index} -ge 0 ]]; then
                    PREFIX=""
                    cur="${cur#*=}"
                    ${flags_completion[${index}]}
                    if [ -n "${ZSH_VERSION:-}" ]; then
                        # zsh completion needs --flag= prefix
                        eval "COMPREPLY=( \"\${COMPREPLY[@]/#/${flag}=}\" )"
                    fi
                fi
            fi

            if [[ -z "${flag_parsing_disabled}" ]]; then
                # If flag parsing is enabled, we have completed the flags and can return.
                # If flag parsing is disabled, we may not know all (or any) of the flags, so we fallthrough
                # to possibly call handle_go_custom_completion.
                return 0;
            fi
            ;;
    esac

    # check if we are handling a flag with special work handling
    local index
    __angler_index_of_word "${prev}" "${flags_with_completion[@]}"
    if [[ ${index} -ge 0 ]]; then
        ${flags_completion[${index}]}
        return
    fi

    # we are parsing a flag and don't have a special handler, no completion
    if [[ ${cur} != "${words[cword]}" ]]; then
        return
    fi

    local completions
    completions=("${commands[@]}")
    if [[ ${#must_have_one_noun[@]} -ne 0 ]]; then
        completions+=("${must_have_one_noun[@]}")
    elif [[ -n "${has_completion_function}" ]]; then
        # if a go completion function is provided, defer to that function
        __angler_handle_go_custom_completion
    fi
    if [[ ${#must_have_one_flag[@]} -ne 0 ]]; then
        completions+=("${must_have_one_flag[@]}")
    fi
    while IFS='' read -r comp; do
        COMPREPLY+=("$comp")
    done < <(compgen -W "${completions[*]}" -- "$cur")

    if [[ ${#COMPREPLY[@]} -eq 0 && ${#noun_aliases[@]} -gt 0 && ${#must_have_one_noun[@]} -ne 0 ]]; then
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${noun_aliases[*]}" -- "$cur")
    fi

    if [[ ${#COMPREPLY[@]} -eq 0 ]]; then
        if declare -F __angler_custom_func >/dev/null; then
            # try command name qualified custom func
            __angler_custom_func
        else
            # otherwise fall back to unqualified for compatibility
            declare -F __custom_func >/dev/null && __custom_func
        fi
    fi

    # available in bash-completion >= 2, not always present on macOS
    if declare -F __ltrim_colon_completions >/dev/null; then
        __ltrim_colon_completions "$cur"
    fi

    # If there is only 1 completion and it is a flag with an = it will be completed
    # but we don't want a space after the =
    if [[ "${#COMPREPLY[@]}" -eq "1" ]] && [[ $(type -t compopt) = "builtin" ]] && [[ "${COMPREPLY[0]}" == --*= ]]; then
       compopt -o nospace
    fi
}

# The arguments should be in the form "ext1|ext2|extn"
__angler_handle_filename_extension_flag()
{
    local ext="$1"
    _filedir "@(${ext})"
}

__angler_handle_subdirs_in_dir_flag()
{
    local dir="$1"
    pushd "${dir}" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
}

__angler_handle_flag()
{
    __angler_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    # if a command required a flag, and we found it, unset must_have_one_flag()
    local flagname=${words[c]}
    local flagvalue=""
    # if the word contained an =
    if [[ ${words[c]} == *"="* ]]; then
        flagvalue=${flagname#*=} # take in as flagvalue after the =
        flagname=${flagname%=*} # strip everything after the =
        flagname="${flagname}=" # but put the = back
    fi
    __angler_debug "${FUNCNAME[0]}: looking for ${flagname}"
    if __angler_contains_word "${flagname}" "${must_have_one_flag[@]}"; then
        must_have_one_flag=()
    fi

    # if you set a flag which only applies to this command, don't show subcommands
    if __angler_contains_word "${flagname}" "${local_nonpersistent_flags[@]}"; then
      commands=()
    fi

    # keep flag value with flagname as flaghash
    # flaghash variable is an associative array which is only supported in bash > 3.
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        if [ -n "${flagvalue}" ] ; then
            flaghash[${flagname}]=${flagvalue}
        elif [ -n "${words[ $((c+1)) ]}" ] ; then
            flaghash[${flagname}]=${words[ $((c+1)) ]}
        else
            flaghash[${flagname}]="true" # pad "true" for bool flag
        fi
    fi

    # skip the argument to a two word flag
    if [[ ${words[c]} != *"="* ]] && __angler_contains_word "${words[c]}" "${two_word_flags[@]}"; then
        __angler_debug "${FUNCNAME[0]}: found a flag ${words[c]}, skip the next argument"
        c=$((c+1))
        # if we are looking for a flags value, don't show commands
        if [[ $c -eq $cword ]]; then
            commands=()
        fi
    fi

    c=$((c+1))

}

__angler_handle_noun()
{
    __angler_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    if __angler_contains_word "${words[c]}" "${must_have_one_noun[@]}"; then
        must_have_one_noun=()
    elif __angler_contains_word "${words[c]}" "${noun_aliases[@]}"; then
        must_have_one_noun=()
    fi

    nouns+=("${words[c]}")
    c=$((c+1))
}

__angler_handle_command()
{
    __angler_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    local next_command
    if [[ -n ${last_command} ]]; then
        next_command="_${last_command}_${words[c]//:/__}"
    else
        if [[ $c -eq 0 ]]; then
            next_command="_angler_root_command"
        else
            next_command="_${words[c]//:/__}"
        fi
    fi
    c=$((c+1))
    __angler_debug "${FUNCNAME[0]}: looking for ${next_command}"
    declare -F "$next_command" >/dev/null && $next_command
}

__angler_handle_word()
{
    if [[ $c -ge $cword ]]; then
        __angler_handle_reply
        return
    fi
    __angler_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"
    if [[ "${words[c]}" == -* ]]; then
        __angler_handle_flag
    elif __angler_contains_word "${words[c]}" "${commands[@]}"; then
        __angler_handle_command
    elif [[ $c -eq 0 ]]; then
        __angler_handle_command
    elif __angler_contains_word "${words[c]}" "${command_aliases[@]}"; then
        # aliashash variable is an associative array which is only supported in bash > 3.
        if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
            words[c]=${aliashash[${words[c]}]}
            __angler_handle_command
        else
            __angler_handle_noun
        fi
    else
        __angler_handle_noun
    fi
    __angler_handle_word
}

_angler_admin_ai-providers_configure()
{
    last_command="angler_admin_ai-providers_configure"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--activate")
    local_nonpersistent_flags+=("--activate")
    flags+=("--api-key=")
    two_word_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key=")
    flags+=("--base-url=")
    two_word_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url=")
    flags+=("--chat-model=")
    two_word_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--config-file=")
    two_word_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file=")
    flags+=("--embedding-model=")
    two_word_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--location=")
    two_word_flags+=("--location")
    local_nonpersistent_flags+=("--location")
    local_nonpersistent_flags+=("--location=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--region=")
    two_word_flags+=("--region")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    flags+=("--service-account-file=")
    two_word_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file=")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers_deprecation()
{
    last_command="angler_admin_ai-providers_deprecation"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers_discover-models()
{
    last_command="angler_admin_ai-providers_discover-models"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--api-key=")
    two_word_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key=")
    flags+=("--base-url=")
    two_word_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url=")
    flags+=("--chat-model=")
    two_word_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--config-file=")
    two_word_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file=")
    flags+=("--embedding-model=")
    two_word_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--location=")
    two_word_flags+=("--location")
    local_nonpersistent_flags+=("--location")
    local_nonpersistent_flags+=("--location=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--region=")
    two_word_flags+=("--region")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    flags+=("--service-account-file=")
    two_word_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file=")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers_dismiss-deprecation()
{
    last_command="angler_admin_ai-providers_dismiss-deprecation"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers_help()
{
    last_command="angler_admin_ai-providers_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_ai-providers_list()
{
    last_command="angler_admin_ai-providers_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers_test()
{
    last_command="angler_admin_ai-providers_test"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--api-key=")
    two_word_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key")
    local_nonpersistent_flags+=("--api-key=")
    flags+=("--base-url=")
    two_word_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url")
    local_nonpersistent_flags+=("--base-url=")
    flags+=("--chat-model=")
    two_word_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model")
    local_nonpersistent_flags+=("--chat-model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--config-file=")
    two_word_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file=")
    flags+=("--embedding-model=")
    two_word_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model")
    local_nonpersistent_flags+=("--embedding-model=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--location=")
    two_word_flags+=("--location")
    local_nonpersistent_flags+=("--location")
    local_nonpersistent_flags+=("--location=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--region=")
    two_word_flags+=("--region")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    flags+=("--service-account-file=")
    two_word_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file=")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_ai-providers()
{
    last_command="angler_admin_ai-providers"

    command_aliases=()

    commands=()
    commands+=("configure")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("set")
        aliashash["set"]="configure"
        command_aliases+=("update")
        aliashash["update"]="configure"
    fi
    commands+=("deprecation")
    commands+=("discover-models")
    commands+=("dismiss-deprecation")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("get")
        aliashash["get"]="list"
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("test")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_help()
{
    last_command="angler_admin_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_services_help()
{
    last_command="angler_admin_services_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_services_list()
{
    last_command="angler_admin_services_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_services_restart()
{
    last_command="angler_admin_services_restart"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_services_start()
{
    last_command="angler_admin_services_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_services_stop()
{
    last_command="angler_admin_services_stop"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_services()
{
    last_command="angler_admin_services"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("restart")
    commands+=("start")
    commands+=("stop")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_sessions_help()
{
    last_command="angler_admin_sessions_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_sessions_list()
{
    last_command="angler_admin_sessions_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_sessions_revoke()
{
    last_command="angler_admin_sessions_revoke"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_sessions()
{
    last_command="angler_admin_sessions"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("revoke")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_settings_get()
{
    last_command="angler_admin_settings_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_settings_help()
{
    last_command="angler_admin_settings_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_settings_update()
{
    last_command="angler_admin_settings_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--gemini-key=")
    two_word_flags+=("--gemini-key")
    local_nonpersistent_flags+=("--gemini-key")
    local_nonpersistent_flags+=("--gemini-key=")
    flags+=("--gemini-model=")
    two_word_flags+=("--gemini-model")
    local_nonpersistent_flags+=("--gemini-model")
    local_nonpersistent_flags+=("--gemini-model=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--max-hours=")
    two_word_flags+=("--max-hours")
    local_nonpersistent_flags+=("--max-hours")
    local_nonpersistent_flags+=("--max-hours=")
    flags+=("--session-timeout=")
    two_word_flags+=("--session-timeout")
    local_nonpersistent_flags+=("--session-timeout")
    local_nonpersistent_flags+=("--session-timeout=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_settings()
{
    last_command="angler_admin_settings"

    command_aliases=()

    commands=()
    commands+=("get")
    commands+=("help")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults_configure()
{
    last_command="angler_admin_vaults_configure"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--auth-method=")
    two_word_flags+=("--auth-method")
    local_nonpersistent_flags+=("--auth-method")
    local_nonpersistent_flags+=("--auth-method=")
    flags+=("--client-id=")
    two_word_flags+=("--client-id")
    local_nonpersistent_flags+=("--client-id")
    local_nonpersistent_flags+=("--client-id=")
    flags+=("--client-secret=")
    two_word_flags+=("--client-secret")
    local_nonpersistent_flags+=("--client-secret")
    local_nonpersistent_flags+=("--client-secret=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--config-file=")
    two_word_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--mount-point=")
    two_word_flags+=("--mount-point")
    local_nonpersistent_flags+=("--mount-point")
    local_nonpersistent_flags+=("--mount-point=")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    local_nonpersistent_flags+=("--namespace")
    local_nonpersistent_flags+=("--namespace=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--region=")
    two_word_flags+=("--region")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    flags+=("--role-id=")
    two_word_flags+=("--role-id")
    local_nonpersistent_flags+=("--role-id")
    local_nonpersistent_flags+=("--role-id=")
    flags+=("--secret-id=")
    two_word_flags+=("--secret-id")
    local_nonpersistent_flags+=("--secret-id")
    local_nonpersistent_flags+=("--secret-id=")
    flags+=("--service-account-file=")
    two_word_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file=")
    flags+=("--tenant-id=")
    two_word_flags+=("--tenant-id")
    local_nonpersistent_flags+=("--tenant-id")
    local_nonpersistent_flags+=("--tenant-id=")
    flags+=("--vault-token=")
    two_word_flags+=("--vault-token")
    local_nonpersistent_flags+=("--vault-token")
    local_nonpersistent_flags+=("--vault-token=")
    flags+=("--vault-url=")
    two_word_flags+=("--vault-url")
    local_nonpersistent_flags+=("--vault-url")
    local_nonpersistent_flags+=("--vault-url=")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults_get()
{
    last_command="angler_admin_vaults_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults_help()
{
    last_command="angler_admin_vaults_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_admin_vaults_list()
{
    last_command="angler_admin_vaults_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults_scopes()
{
    last_command="angler_admin_vaults_scopes"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults_test()
{
    last_command="angler_admin_vaults_test"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--auth-method=")
    two_word_flags+=("--auth-method")
    local_nonpersistent_flags+=("--auth-method")
    local_nonpersistent_flags+=("--auth-method=")
    flags+=("--client-id=")
    two_word_flags+=("--client-id")
    local_nonpersistent_flags+=("--client-id")
    local_nonpersistent_flags+=("--client-id=")
    flags+=("--client-secret=")
    two_word_flags+=("--client-secret")
    local_nonpersistent_flags+=("--client-secret")
    local_nonpersistent_flags+=("--client-secret=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--config-file=")
    two_word_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file")
    local_nonpersistent_flags+=("--config-file=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--mount-point=")
    two_word_flags+=("--mount-point")
    local_nonpersistent_flags+=("--mount-point")
    local_nonpersistent_flags+=("--mount-point=")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    local_nonpersistent_flags+=("--namespace")
    local_nonpersistent_flags+=("--namespace=")
    flags+=("--project-id=")
    two_word_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id")
    local_nonpersistent_flags+=("--project-id=")
    flags+=("--region=")
    two_word_flags+=("--region")
    local_nonpersistent_flags+=("--region")
    local_nonpersistent_flags+=("--region=")
    flags+=("--role-id=")
    two_word_flags+=("--role-id")
    local_nonpersistent_flags+=("--role-id")
    local_nonpersistent_flags+=("--role-id=")
    flags+=("--secret-id=")
    two_word_flags+=("--secret-id")
    local_nonpersistent_flags+=("--secret-id")
    local_nonpersistent_flags+=("--secret-id=")
    flags+=("--service-account-file=")
    two_word_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file")
    local_nonpersistent_flags+=("--service-account-file=")
    flags+=("--tenant-id=")
    two_word_flags+=("--tenant-id")
    local_nonpersistent_flags+=("--tenant-id")
    local_nonpersistent_flags+=("--tenant-id=")
    flags+=("--vault-token=")
    two_word_flags+=("--vault-token")
    local_nonpersistent_flags+=("--vault-token")
    local_nonpersistent_flags+=("--vault-token=")
    flags+=("--vault-url=")
    two_word_flags+=("--vault-url")
    local_nonpersistent_flags+=("--vault-url")
    local_nonpersistent_flags+=("--vault-url=")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin_vaults()
{
    last_command="angler_admin_vaults"

    command_aliases=()

    commands=()
    commands+=("configure")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("set")
        aliashash["set"]="configure"
        command_aliases+=("update")
        aliashash["update"]="configure"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("scopes")
    commands+=("test")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_admin()
{
    last_command="angler_admin"

    command_aliases=()

    commands=()
    commands+=("ai-providers")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ai-provider")
        aliashash["ai-provider"]="ai-providers"
    fi
    commands+=("help")
    commands+=("services")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("service")
        aliashash["service"]="services"
        command_aliases+=("svc")
        aliashash["svc"]="services"
    fi
    commands+=("sessions")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("session")
        aliashash["session"]="sessions"
    fi
    commands+=("settings")
    commands+=("vaults")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("vault")
        aliashash["vault"]="vaults"
        command_aliases+=("vault-providers")
        aliashash["vault-providers"]="vaults"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_chat()
{
    last_command="angler_ai_chat"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--mode=")
    two_word_flags+=("--mode")
    two_word_flags+=("-m")
    local_nonpersistent_flags+=("--mode")
    local_nonpersistent_flags+=("--mode=")
    local_nonpersistent_flags+=("-m")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--no-stream")
    local_nonpersistent_flags+=("--no-stream")
    flags+=("--stream")
    local_nonpersistent_flags+=("--stream")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_context()
{
    last_command="angler_ai_context"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_embeddings()
{
    last_command="angler_ai_embeddings"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_enrich()
{
    last_command="angler_ai_enrich"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--table=")
    two_word_flags+=("--table")
    local_nonpersistent_flags+=("--table")
    local_nonpersistent_flags+=("--table=")
    flags+=("--type=")
    two_word_flags+=("--type")
    local_nonpersistent_flags+=("--type")
    local_nonpersistent_flags+=("--type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_help()
{
    last_command="angler_ai_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_ai_models()
{
    last_command="angler_ai_models"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_retrieve()
{
    last_command="angler_ai_retrieve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--score-threshold=")
    two_word_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold=")
    flags+=("--table=")
    two_word_flags+=("--table")
    local_nonpersistent_flags+=("--table")
    local_nonpersistent_flags+=("--table=")
    flags+=("--top-k=")
    two_word_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_table-from-prompt()
{
    last_command="angler_ai_table-from-prompt"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_test-key()
{
    last_command="angler_ai_test-key"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--key=")
    two_word_flags+=("--key")
    local_nonpersistent_flags+=("--key")
    local_nonpersistent_flags+=("--key=")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-index_help()
{
    last_command="angler_ai_vector-index_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_ai_vector-index_list()
{
    last_command="angler_ai_vector-index_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-index_search()
{
    last_command="angler_ai_vector-index_search"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--score-threshold=")
    two_word_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold=")
    flags+=("--table=")
    two_word_flags+=("--table")
    local_nonpersistent_flags+=("--table")
    local_nonpersistent_flags+=("--table=")
    flags+=("--top-k=")
    two_word_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-index_sync()
{
    last_command="angler_ai_vector-index_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-index()
{
    last_command="angler_ai_vector-index"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("search")
    commands+=("sync")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-stores_help()
{
    last_command="angler_ai_vector-stores_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_ai_vector-stores_list()
{
    last_command="angler_ai_vector-stores_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-stores_search()
{
    last_command="angler_ai_vector-stores_search"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--score-threshold=")
    two_word_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold")
    local_nonpersistent_flags+=("--score-threshold=")
    flags+=("--top-k=")
    two_word_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k")
    local_nonpersistent_flags+=("--top-k=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai_vector-stores()
{
    last_command="angler_ai_vector-stores"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("search")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ai()
{
    last_command="angler_ai"

    command_aliases=()

    commands=()
    commands+=("chat")
    commands+=("context")
    commands+=("embeddings")
    commands+=("enrich")
    commands+=("help")
    commands+=("models")
    commands+=("retrieve")
    commands+=("table-from-prompt")
    commands+=("test-key")
    commands+=("vector-index")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("vector")
        aliashash["vector"]="vector-index"
        command_aliases+=("vectors")
        aliashash["vectors"]="vector-index"
    fi
    commands+=("vector-stores")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("vector-store")
        aliashash["vector-store"]="vector-stores"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_ask()
{
    last_command="angler_ask"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_auth_help()
{
    last_command="angler_auth_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_auth_login()
{
    last_command="angler_auth_login"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--email=")
    two_word_flags+=("--email")
    local_nonpersistent_flags+=("--email")
    local_nonpersistent_flags+=("--email=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--password=")
    two_word_flags+=("--password")
    local_nonpersistent_flags+=("--password")
    local_nonpersistent_flags+=("--password=")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--token")
    local_nonpersistent_flags+=("--token=")
    local_nonpersistent_flags+=("-t")
    flags+=("--token-stdin")
    local_nonpersistent_flags+=("--token-stdin")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_auth_logout()
{
    last_command="angler_auth_logout"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_auth_status()
{
    last_command="angler_auth_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_auth()
{
    last_command="angler_auth"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("login")
    commands+=("logout")
    commands+=("status")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_deploy()
{
    last_command="angler_bundle_deploy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_destroy()
{
    last_command="angler_bundle_destroy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--bundle-id=")
    two_word_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--purge-storage")
    local_nonpersistent_flags+=("--purge-storage")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_generate()
{
    last_command="angler_bundle_generate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--deploy")
    local_nonpersistent_flags+=("--deploy")
    flags+=("--dry-run")
    local_nonpersistent_flags+=("--dry-run")
    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--out=")
    two_word_flags+=("--out")
    local_nonpersistent_flags+=("--out")
    local_nonpersistent_flags+=("--out=")
    flags+=("--prompt=")
    two_word_flags+=("--prompt")
    local_nonpersistent_flags+=("--prompt")
    local_nonpersistent_flags+=("--prompt=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_get()
{
    last_command="angler_bundle_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--bundle-id=")
    two_word_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--manifest")
    local_nonpersistent_flags+=("--manifest")
    flags+=("--no-drift")
    local_nonpersistent_flags+=("--no-drift")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_help()
{
    last_command="angler_bundle_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_bundle_init()
{
    last_command="angler_bundle_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--template=")
    two_word_flags+=("--template")
    local_nonpersistent_flags+=("--template")
    local_nonpersistent_flags+=("--template=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_ls()
{
    last_command="angler_bundle_ls"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_run()
{
    last_command="angler_bundle_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--bundle-id=")
    two_word_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id")
    local_nonpersistent_flags+=("--bundle-id=")
    flags+=("--conf=")
    two_word_flags+=("--conf")
    local_nonpersistent_flags+=("--conf")
    local_nonpersistent_flags+=("--conf=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--poll-interval=")
    two_word_flags+=("--poll-interval")
    local_nonpersistent_flags+=("--poll-interval")
    local_nonpersistent_flags+=("--poll-interval=")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle_validate()
{
    last_command="angler_bundle_validate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--local")
    local_nonpersistent_flags+=("--local")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_bundle()
{
    last_command="angler_bundle"

    command_aliases=()

    commands=()
    commands+=("deploy")
    commands+=("destroy")
    commands+=("generate")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ai")
        aliashash["ai"]="generate"
        command_aliases+=("gen")
        aliashash["gen"]="generate"
    fi
    commands+=("get")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("drift")
        aliashash["drift"]="get"
        command_aliases+=("inspect")
        aliashash["inspect"]="get"
        command_aliases+=("status")
        aliashash["status"]="get"
    fi
    commands+=("help")
    commands+=("init")
    commands+=("ls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="ls"
    fi
    commands+=("run")
    commands+=("validate")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--target=")
    two_word_flags+=("--target")
    two_word_flags+=("-T")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access_approve()
{
    last_command="angler_catalog_access_approve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access_help()
{
    last_command="angler_catalog_access_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_access_list()
{
    last_command="angler_catalog_access_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access_reject()
{
    last_command="angler_catalog_access_reject"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access_request()
{
    last_command="angler_catalog_access_request"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--message=")
    two_word_flags+=("--message")
    two_word_flags+=("-m")
    local_nonpersistent_flags+=("--message")
    local_nonpersistent_flags+=("--message=")
    local_nonpersistent_flags+=("-m")
    flags+=("--mode=")
    two_word_flags+=("--mode")
    local_nonpersistent_flags+=("--mode")
    local_nonpersistent_flags+=("--mode=")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access_revoke()
{
    last_command="angler_catalog_access_revoke"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_access()
{
    last_command="angler_catalog_access"

    command_aliases=()

    commands=()
    commands+=("approve")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("reject")
    commands+=("request")
    commands+=("revoke")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_anomalies_detect()
{
    last_command="angler_catalog_anomalies_detect"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--contamination=")
    two_word_flags+=("--contamination")
    local_nonpersistent_flags+=("--contamination")
    local_nonpersistent_flags+=("--contamination=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--source=")
    two_word_flags+=("--source")
    local_nonpersistent_flags+=("--source")
    local_nonpersistent_flags+=("--source=")
    flags+=("--table=")
    two_word_flags+=("--table")
    local_nonpersistent_flags+=("--table")
    local_nonpersistent_flags+=("--table=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_anomalies_help()
{
    last_command="angler_catalog_anomalies_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_anomalies()
{
    last_command="angler_catalog_anomalies"

    command_aliases=()

    commands=()
    commands+=("detect")
    commands+=("help")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_create()
{
    last_command="angler_catalog_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_delete()
{
    last_command="angler_catalog_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--purge")
    local_nonpersistent_flags+=("--purge")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_directory()
{
    last_command="angler_catalog_directory"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_get()
{
    last_command="angler_catalog_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_help()
{
    last_command="angler_catalog_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_ls()
{
    last_command="angler_catalog_ls"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_manifest()
{
    last_command="angler_catalog_manifest"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas_create()
{
    last_command="angler_catalog_schemas_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas_delete()
{
    last_command="angler_catalog_schemas_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cascade")
    local_nonpersistent_flags+=("--cascade")
    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas_get()
{
    last_command="angler_catalog_schemas_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas_help()
{
    last_command="angler_catalog_schemas_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_schemas_list()
{
    last_command="angler_catalog_schemas_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas_update()
{
    last_command="angler_catalog_schemas_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_schemas()
{
    last_command="angler_catalog_schemas"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_tables_delete()
{
    last_command="angler_catalog_tables_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_tables_get()
{
    last_command="angler_catalog_tables_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_tables_help()
{
    last_command="angler_catalog_tables_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_tables_list()
{
    last_command="angler_catalog_tables_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_tables_update()
{
    last_command="angler_catalog_tables_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--s3-path=")
    two_word_flags+=("--s3-path")
    local_nonpersistent_flags+=("--s3-path")
    local_nonpersistent_flags+=("--s3-path=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_tables()
{
    last_command="angler_catalog_tables"

    command_aliases=()

    commands=()
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_toon()
{
    last_command="angler_catalog_toon"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--raw")
    local_nonpersistent_flags+=("--raw")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_trash_help()
{
    last_command="angler_catalog_trash_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_catalog_trash_list()
{
    last_command="angler_catalog_trash_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_trash_purge()
{
    last_command="angler_catalog_trash_purge"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_trash_purge-expired()
{
    last_command="angler_catalog_trash_purge-expired"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_trash_restore()
{
    last_command="angler_catalog_trash_restore"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--new-name=")
    two_word_flags+=("--new-name")
    local_nonpersistent_flags+=("--new-name")
    local_nonpersistent_flags+=("--new-name=")
    flags+=("--strategy=")
    two_word_flags+=("--strategy")
    local_nonpersistent_flags+=("--strategy")
    local_nonpersistent_flags+=("--strategy=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_trash()
{
    last_command="angler_catalog_trash"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("purge")
    commands+=("purge-expired")
    commands+=("restore")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("undrop")
        aliashash["undrop"]="restore"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_undrop()
{
    last_command="angler_catalog_undrop"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--new-name=")
    two_word_flags+=("--new-name")
    local_nonpersistent_flags+=("--new-name")
    local_nonpersistent_flags+=("--new-name=")
    flags+=("--strategy=")
    two_word_flags+=("--strategy")
    local_nonpersistent_flags+=("--strategy")
    local_nonpersistent_flags+=("--strategy=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog_update()
{
    last_command="angler_catalog_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    local_nonpersistent_flags+=("-d")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_catalog()
{
    last_command="angler_catalog"

    command_aliases=()

    commands=()
    commands+=("access")
    commands+=("anomalies")
    commands+=("create")
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("directory")
    commands+=("get")
    commands+=("help")
    commands+=("ls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="ls"
    fi
    commands+=("manifest")
    commands+=("schemas")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("schema")
        aliashash["schema"]="schemas"
    fi
    commands+=("tables")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("table")
        aliashash["table"]="tables"
    fi
    commands+=("toon")
    commands+=("trash")
    commands+=("undrop")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_completion_bash()
{
    last_command="angler_completion_bash"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-descriptions")
    local_nonpersistent_flags+=("--no-descriptions")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_completion_fish()
{
    last_command="angler_completion_fish"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-descriptions")
    local_nonpersistent_flags+=("--no-descriptions")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_completion_help()
{
    last_command="angler_completion_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_completion_powershell()
{
    last_command="angler_completion_powershell"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-descriptions")
    local_nonpersistent_flags+=("--no-descriptions")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_completion_zsh()
{
    last_command="angler_completion_zsh"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-descriptions")
    local_nonpersistent_flags+=("--no-descriptions")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_completion()
{
    last_command="angler_completion"

    command_aliases=()

    commands=()
    commands+=("bash")
    commands+=("fish")
    commands+=("help")
    commands+=("powershell")
    commands+=("zsh")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_compute_help()
{
    last_command="angler_compute_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_compute_list()
{
    last_command="angler_compute_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_restart()
{
    last_command="angler_compute_restart"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_services()
{
    last_command="angler_compute_services"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_start()
{
    last_command="angler_compute_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_status()
{
    last_command="angler_compute_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_stop()
{
    last_command="angler_compute_stop"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute_topology()
{
    last_command="angler_compute_topology"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_compute()
{
    last_command="angler_compute"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("restart")
    commands+=("services")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("service")
        aliashash["service"]="services"
        command_aliases+=("svc")
        aliashash["svc"]="services"
    fi
    commands+=("start")
    commands+=("status")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("health")
        aliashash["health"]="status"
    fi
    commands+=("stop")
    commands+=("topology")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("topo")
        aliashash["topo"]="topology"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_context_help()
{
    last_command="angler_context_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_context_list()
{
    last_command="angler_context_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_context_set()
{
    last_command="angler_context_set"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--endpoint")
    local_nonpersistent_flags+=("--endpoint=")
    local_nonpersistent_flags+=("-e")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    local_nonpersistent_flags+=("--workspace-id")
    local_nonpersistent_flags+=("--workspace-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_context_use()
{
    last_command="angler_context_use"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_context()
{
    last_command="angler_context"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    commands+=("set")
    commands+=("use")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_create()
{
    last_command="angler_dashboards_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--published")
    local_nonpersistent_flags+=("--published")
    flags+=("--spec=")
    two_word_flags+=("--spec")
    local_nonpersistent_flags+=("--spec")
    local_nonpersistent_flags+=("--spec=")
    flags+=("--template=")
    two_word_flags+=("--template")
    local_nonpersistent_flags+=("--template")
    local_nonpersistent_flags+=("--template=")
    flags+=("--title=")
    two_word_flags+=("--title")
    local_nonpersistent_flags+=("--title")
    local_nonpersistent_flags+=("--title=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_delete()
{
    last_command="angler_dashboards_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_get()
{
    last_command="angler_dashboards_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_help()
{
    last_command="angler_dashboards_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_dashboards_list()
{
    last_command="angler_dashboards_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_stats()
{
    last_command="angler_dashboards_stats"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_widgets_get-data()
{
    last_command="angler_dashboards_widgets_get-data"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--time-range=")
    two_word_flags+=("--time-range")
    local_nonpersistent_flags+=("--time-range")
    local_nonpersistent_flags+=("--time-range=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_widgets_help()
{
    last_command="angler_dashboards_widgets_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_dashboards_widgets_list()
{
    last_command="angler_dashboards_widgets_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards_widgets()
{
    last_command="angler_dashboards_widgets"

    command_aliases=()

    commands=()
    commands+=("get-data")
    commands+=("help")
    commands+=("list")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_dashboards()
{
    last_command="angler_dashboards"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("stats")
    commands+=("widgets")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("widget")
        aliashash["widget"]="widgets"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_docs()
{
    last_command="angler_docs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--clean")
    local_nonpersistent_flags+=("--clean")
    flags+=("--completions-dir=")
    two_word_flags+=("--completions-dir")
    local_nonpersistent_flags+=("--completions-dir")
    local_nonpersistent_flags+=("--completions-dir=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_down()
{
    last_command="angler_down"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--volumes")
    flags+=("-v")
    local_nonpersistent_flags+=("--volumes")
    local_nonpersistent_flags+=("-v")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_cat()
{
    last_command="angler_fs_cat"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--length=")
    two_word_flags+=("--length")
    local_nonpersistent_flags+=("--length")
    local_nonpersistent_flags+=("--length=")
    flags+=("--offset=")
    two_word_flags+=("--offset")
    local_nonpersistent_flags+=("--offset")
    local_nonpersistent_flags+=("--offset=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_cp()
{
    last_command="angler_fs_cp"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_download()
{
    last_command="angler_fs_download"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_help()
{
    last_command="angler_fs_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_fs_ls()
{
    last_command="angler_fs_ls"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--absolute")
    flags+=("-a")
    local_nonpersistent_flags+=("--absolute")
    local_nonpersistent_flags+=("-a")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    local_nonpersistent_flags+=("-l")
    flags+=("--recursive")
    flags+=("-r")
    local_nonpersistent_flags+=("--recursive")
    local_nonpersistent_flags+=("-r")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_mkdir()
{
    last_command="angler_fs_mkdir"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_rm()
{
    last_command="angler_fs_rm"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--recursive")
    flags+=("-r")
    local_nonpersistent_flags+=("--recursive")
    local_nonpersistent_flags+=("-r")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_stat()
{
    last_command="angler_fs_stat"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_stats()
{
    last_command="angler_fs_stats"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs_upload()
{
    last_command="angler_fs_upload"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--prefix=")
    two_word_flags+=("--prefix")
    two_word_flags+=("-p")
    local_nonpersistent_flags+=("--prefix")
    local_nonpersistent_flags+=("--prefix=")
    local_nonpersistent_flags+=("-p")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_fs()
{
    last_command="angler_fs"

    command_aliases=()

    commands=()
    commands+=("cat")
    commands+=("cp")
    commands+=("download")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("get")
        aliashash["get"]="download"
    fi
    commands+=("help")
    commands+=("ls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="ls"
    fi
    commands+=("mkdir")
    commands+=("rm")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("delete")
        aliashash["delete"]="rm"
    fi
    commands+=("stat")
    commands+=("stats")
    commands+=("upload")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("put")
        aliashash["put"]="upload"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_help()
{
    last_command="angler_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_jobs_cancel()
{
    last_command="angler_jobs_cancel"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_delete()
{
    last_command="angler_jobs_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_get()
{
    last_command="angler_jobs_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_help()
{
    last_command="angler_jobs_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_jobs_list()
{
    last_command="angler_jobs_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_logs()
{
    last_command="angler_jobs_logs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--try=")
    two_word_flags+=("--try")
    local_nonpersistent_flags+=("--try")
    local_nonpersistent_flags+=("--try=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_pause()
{
    last_command="angler_jobs_pause"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_run()
{
    last_command="angler_jobs_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--conf=")
    two_word_flags+=("--conf")
    local_nonpersistent_flags+=("--conf")
    local_nonpersistent_flags+=("--conf=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--poll-interval=")
    two_word_flags+=("--poll-interval")
    local_nonpersistent_flags+=("--poll-interval")
    local_nonpersistent_flags+=("--poll-interval=")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_cancel()
{
    last_command="angler_jobs_cancel"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_runs_get()
{
    last_command="angler_jobs_runs_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_runs_help()
{
    last_command="angler_jobs_runs_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_jobs_runs_list()
{
    last_command="angler_jobs_runs_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    local_nonpersistent_flags+=("-l")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_runs_logs()
{
    last_command="angler_jobs_runs_logs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--try=")
    two_word_flags+=("--try")
    local_nonpersistent_flags+=("--try")
    local_nonpersistent_flags+=("--try=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_runs()
{
    last_command="angler_jobs_runs"

    command_aliases=()

    commands=()
    commands+=("cancel")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("stop")
        aliashash["stop"]="cancel"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("logs")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_schedule()
{
    last_command="angler_jobs_schedule"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_schedule-notebook()
{
    last_command="angler_jobs_schedule-notebook"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cpu=")
    two_word_flags+=("--cpu")
    local_nonpersistent_flags+=("--cpu")
    local_nonpersistent_flags+=("--cpu=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--memory=")
    two_word_flags+=("--memory")
    local_nonpersistent_flags+=("--memory")
    local_nonpersistent_flags+=("--memory=")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--schedule=")
    two_word_flags+=("--schedule")
    local_nonpersistent_flags+=("--schedule")
    local_nonpersistent_flags+=("--schedule=")
    flags+=("--title=")
    two_word_flags+=("--title")
    local_nonpersistent_flags+=("--title")
    local_nonpersistent_flags+=("--title=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_settings_get()
{
    last_command="angler_jobs_settings_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_settings_help()
{
    last_command="angler_jobs_settings_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_jobs_settings_update()
{
    last_command="angler_jobs_settings_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--color=")
    two_word_flags+=("--color")
    local_nonpersistent_flags+=("--color")
    local_nonpersistent_flags+=("--color=")
    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--label=")
    two_word_flags+=("--label")
    local_nonpersistent_flags+=("--label")
    local_nonpersistent_flags+=("--label=")
    flags+=("--timeline-visible")
    local_nonpersistent_flags+=("--timeline-visible")
    flags+=("--webhook-enabled")
    local_nonpersistent_flags+=("--webhook-enabled")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_settings()
{
    last_command="angler_jobs_settings"

    command_aliases=()

    commands=()
    commands+=("get")
    commands+=("help")
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_timeline()
{
    last_command="angler_jobs_timeline"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--date=")
    two_word_flags+=("--date")
    local_nonpersistent_flags+=("--date")
    local_nonpersistent_flags+=("--date=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--tz=")
    two_word_flags+=("--tz")
    local_nonpersistent_flags+=("--tz")
    local_nonpersistent_flags+=("--tz=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_unpause()
{
    last_command="angler_jobs_unpause"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_webhook_disable()
{
    last_command="angler_jobs_webhook_disable"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_webhook_enable()
{
    last_command="angler_jobs_webhook_enable"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_webhook_help()
{
    last_command="angler_jobs_webhook_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_jobs_webhook_trigger()
{
    last_command="angler_jobs_webhook_trigger"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--hmac")
    local_nonpersistent_flags+=("--hmac")
    flags+=("--payload=")
    two_word_flags+=("--payload")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--payload")
    local_nonpersistent_flags+=("--payload=")
    local_nonpersistent_flags+=("-d")
    flags+=("--payload-file=")
    two_word_flags+=("--payload-file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--payload-file")
    local_nonpersistent_flags+=("--payload-file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--secret=")
    two_word_flags+=("--secret")
    local_nonpersistent_flags+=("--secret")
    local_nonpersistent_flags+=("--secret=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs_webhook()
{
    last_command="angler_jobs_webhook"

    command_aliases=()

    commands=()
    commands+=("disable")
    commands+=("enable")
    commands+=("help")
    commands+=("trigger")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_jobs()
{
    last_command="angler_jobs"

    command_aliases=()

    commands=()
    commands+=("cancel")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("stop")
        aliashash["stop"]="cancel"
    fi
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("logs")
    commands+=("pause")
    commands+=("run")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("trigger")
        aliashash["trigger"]="run"
    fi
    commands+=("runs")
    commands+=("schedule")
    commands+=("schedule-notebook")
    commands+=("settings")
    commands+=("timeline")
    commands+=("unpause")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("resume")
        aliashash["resume"]="unpause"
    fi
    commands+=("webhook")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_datasets()
{
    last_command="angler_lakeset_datasets"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--search=")
    two_word_flags+=("--search")
    two_word_flags+=("-q")
    local_nonpersistent_flags+=("--search")
    local_nonpersistent_flags+=("--search=")
    local_nonpersistent_flags+=("-q")
    flags+=("--tier=")
    two_word_flags+=("--tier")
    local_nonpersistent_flags+=("--tier")
    local_nonpersistent_flags+=("--tier=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_explore()
{
    last_command="angler_lakeset_explore"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--agg=")
    two_word_flags+=("--agg")
    local_nonpersistent_flags+=("--agg")
    local_nonpersistent_flags+=("--agg=")
    flags+=("--chart-type=")
    two_word_flags+=("--chart-type")
    local_nonpersistent_flags+=("--chart-type")
    local_nonpersistent_flags+=("--chart-type=")
    flags+=("--dataset=")
    two_word_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset=")
    flags+=("--dimension=")
    two_word_flags+=("--dimension")
    local_nonpersistent_flags+=("--dimension")
    local_nonpersistent_flags+=("--dimension=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--metric=")
    two_word_flags+=("--metric")
    local_nonpersistent_flags+=("--metric")
    local_nonpersistent_flags+=("--metric=")
    flags+=("--sql=")
    two_word_flags+=("--sql")
    local_nonpersistent_flags+=("--sql")
    local_nonpersistent_flags+=("--sql=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_guest-token()
{
    last_command="angler_lakeset_guest-token"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dashboard-id=")
    two_word_flags+=("--dashboard-id")
    local_nonpersistent_flags+=("--dashboard-id")
    local_nonpersistent_flags+=("--dashboard-id=")
    flags+=("--dataset=")
    two_word_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--raw")
    local_nonpersistent_flags+=("--raw")
    flags+=("--ttl=")
    two_word_flags+=("--ttl")
    local_nonpersistent_flags+=("--ttl")
    local_nonpersistent_flags+=("--ttl=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_help()
{
    last_command="angler_lakeset_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_lakeset_open()
{
    last_command="angler_lakeset_open"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dataset=")
    two_word_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset")
    local_nonpersistent_flags+=("--dataset=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--no-browser")
    local_nonpersistent_flags+=("--no-browser")
    flags+=("--print")
    local_nonpersistent_flags+=("--print")
    flags+=("--sql=")
    two_word_flags+=("--sql")
    local_nonpersistent_flags+=("--sql")
    local_nonpersistent_flags+=("--sql=")
    flags+=("--tab=")
    two_word_flags+=("--tab")
    local_nonpersistent_flags+=("--tab")
    local_nonpersistent_flags+=("--tab=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_status()
{
    last_command="angler_lakeset_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset_sync()
{
    last_command="angler_lakeset_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--force")
    local_nonpersistent_flags+=("--force")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_lakeset()
{
    last_command="angler_lakeset"

    command_aliases=()

    commands=()
    commands+=("datasets")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="datasets"
        command_aliases+=("ls")
        aliashash["ls"]="datasets"
    fi
    commands+=("explore")
    commands+=("guest-token")
    commands+=("help")
    commands+=("open")
    commands+=("status")
    commands+=("sync")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_help()
{
    last_command="angler_mcp_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_mcp_install()
{
    last_command="angler_mcp_install"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--client=")
    two_word_flags+=("--client")
    local_nonpersistent_flags+=("--client")
    local_nonpersistent_flags+=("--client=")
    flags+=("--config-path=")
    two_word_flags+=("--config-path")
    local_nonpersistent_flags+=("--config-path")
    local_nonpersistent_flags+=("--config-path=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--print")
    local_nonpersistent_flags+=("--print")
    flags+=("--remote")
    local_nonpersistent_flags+=("--remote")
    flags+=("--transport=")
    two_word_flags+=("--transport")
    local_nonpersistent_flags+=("--transport")
    local_nonpersistent_flags+=("--transport=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_read-resource()
{
    last_command="angler_mcp_read-resource"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_resources()
{
    last_command="angler_mcp_resources"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_serve()
{
    last_command="angler_mcp_serve"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--remote")
    local_nonpersistent_flags+=("--remote")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_status()
{
    last_command="angler_mcp_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp_tools()
{
    last_command="angler_mcp_tools"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_mcp()
{
    last_command="angler_mcp"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("install")
    commands+=("read-resource")
    commands+=("resources")
    commands+=("serve")
    commands+=("status")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("info")
        aliashash["info"]="status"
    fi
    commands+=("tools")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_query()
{
    last_command="angler_query"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--execute=")
    two_word_flags+=("--execute")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--execute")
    local_nonpersistent_flags+=("--execute=")
    local_nonpersistent_flags+=("-e")
    flags+=("--explain")
    local_nonpersistent_flags+=("--explain")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--warehouse=")
    two_word_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_acls_delete()
{
    last_command="angler_secrets_acls_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_acls_help()
{
    last_command="angler_secrets_acls_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_secrets_acls_list()
{
    last_command="angler_secrets_acls_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_acls_set()
{
    last_command="angler_secrets_acls_set"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--permission=")
    two_word_flags+=("--permission")
    local_nonpersistent_flags+=("--permission")
    local_nonpersistent_flags+=("--permission=")
    flags+=("--principal=")
    two_word_flags+=("--principal")
    local_nonpersistent_flags+=("--principal")
    local_nonpersistent_flags+=("--principal=")
    flags+=("--principal-type=")
    two_word_flags+=("--principal-type")
    local_nonpersistent_flags+=("--principal-type")
    local_nonpersistent_flags+=("--principal-type=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_acls()
{
    last_command="angler_secrets_acls"

    command_aliases=()

    commands=()
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("revoke")
        aliashash["revoke"]="delete"
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("set")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("grant")
        aliashash["grant"]="set"
        command_aliases+=("put")
        aliashash["put"]="set"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_delete()
{
    last_command="angler_secrets_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_get()
{
    last_command="angler_secrets_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--raw")
    local_nonpersistent_flags+=("--raw")
    flags+=("--show-value")
    local_nonpersistent_flags+=("--show-value")
    flags+=("--unmask")
    local_nonpersistent_flags+=("--unmask")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_help()
{
    last_command="angler_secrets_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_secrets_keys_help()
{
    last_command="angler_secrets_keys_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_secrets_keys_list()
{
    last_command="angler_secrets_keys_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_keys()
{
    last_command="angler_secrets_keys"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_list()
{
    last_command="angler_secrets_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_list-scopes()
{
    last_command="angler_secrets_list-scopes"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_put()
{
    last_command="angler_secrets_put"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--string-value=")
    two_word_flags+=("--string-value")
    local_nonpersistent_flags+=("--string-value")
    local_nonpersistent_flags+=("--string-value=")
    flags+=("--value-stdin")
    local_nonpersistent_flags+=("--value-stdin")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_scopes_create()
{
    last_command="angler_secrets_scopes_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--backend=")
    two_word_flags+=("--backend")
    local_nonpersistent_flags+=("--backend")
    local_nonpersistent_flags+=("--backend=")
    flags+=("--config=")
    two_word_flags+=("--config")
    local_nonpersistent_flags+=("--config")
    local_nonpersistent_flags+=("--config=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_scopes_delete()
{
    last_command="angler_secrets_scopes_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_scopes_get()
{
    last_command="angler_secrets_scopes_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_scopes_help()
{
    last_command="angler_secrets_scopes_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_secrets_scopes_list()
{
    last_command="angler_secrets_scopes_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets_scopes()
{
    last_command="angler_secrets_scopes"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_secrets()
{
    last_command="angler_secrets"

    command_aliases=()

    commands=()
    commands+=("acls")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("acl")
        aliashash["acl"]="acls"
    fi
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("keys")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("list-scopes")
    commands+=("put")
    commands+=("scopes")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("scope")
        aliashash["scope"]="scopes"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_cancel()
{
    last_command="angler_sql_cancel"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_exec()
{
    last_command="angler_sql_exec"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--execute=")
    two_word_flags+=("--execute")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--execute")
    local_nonpersistent_flags+=("--execute=")
    local_nonpersistent_flags+=("-e")
    flags+=("--explain")
    local_nonpersistent_flags+=("--explain")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--warehouse=")
    two_word_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_explain()
{
    last_command="angler_sql_explain"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--execute=")
    two_word_flags+=("--execute")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--execute")
    local_nonpersistent_flags+=("--execute=")
    local_nonpersistent_flags+=("-e")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_fix()
{
    last_command="angler_sql_fix"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--error=")
    two_word_flags+=("--error")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--error")
    local_nonpersistent_flags+=("--error=")
    local_nonpersistent_flags+=("-e")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--run")
    flags+=("-x")
    local_nonpersistent_flags+=("--run")
    local_nonpersistent_flags+=("-x")
    flags+=("--statement=")
    two_word_flags+=("--statement")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--statement")
    local_nonpersistent_flags+=("--statement=")
    local_nonpersistent_flags+=("-s")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_generate()
{
    last_command="angler_sql_generate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--current-sql=")
    two_word_flags+=("--current-sql")
    local_nonpersistent_flags+=("--current-sql")
    local_nonpersistent_flags+=("--current-sql=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--model=")
    two_word_flags+=("--model")
    local_nonpersistent_flags+=("--model")
    local_nonpersistent_flags+=("--model=")
    flags+=("--run")
    flags+=("-x")
    local_nonpersistent_flags+=("--run")
    local_nonpersistent_flags+=("-x")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_get()
{
    last_command="angler_sql_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql_help()
{
    last_command="angler_sql_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_sql_history()
{
    last_command="angler_sql_history"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--status=")
    two_word_flags+=("--status")
    local_nonpersistent_flags+=("--status")
    local_nonpersistent_flags+=("--status=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_sql()
{
    last_command="angler_sql"

    command_aliases=()

    commands=()
    commands+=("cancel")
    commands+=("exec")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("execute")
        aliashash["execute"]="exec"
        command_aliases+=("run")
        aliashash["run"]="exec"
    fi
    commands+=("explain")
    commands+=("fix")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("repair")
        aliashash["repair"]="fix"
    fi
    commands+=("generate")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("create")
        aliashash["create"]="generate"
        command_aliases+=("gen")
        aliashash["gen"]="generate"
    fi
    commands+=("get")
    commands+=("help")
    commands+=("history")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("list")
        aliashash["list"]="history"
        command_aliases+=("ls")
        aliashash["ls"]="history"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--catalog=")
    two_word_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog")
    local_nonpersistent_flags+=("--catalog=")
    flags+=("--execute=")
    two_word_flags+=("--execute")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--execute")
    local_nonpersistent_flags+=("--execute=")
    local_nonpersistent_flags+=("-e")
    flags+=("--explain")
    local_nonpersistent_flags+=("--explain")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--limit=")
    two_word_flags+=("--limit")
    local_nonpersistent_flags+=("--limit")
    local_nonpersistent_flags+=("--limit=")
    flags+=("--params=")
    two_word_flags+=("--params")
    local_nonpersistent_flags+=("--params")
    local_nonpersistent_flags+=("--params=")
    flags+=("--schema=")
    two_word_flags+=("--schema")
    local_nonpersistent_flags+=("--schema")
    local_nonpersistent_flags+=("--schema=")
    flags+=("--warehouse=")
    two_word_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse")
    local_nonpersistent_flags+=("--warehouse=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_status()
{
    last_command="angler_status"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_tokens_create()
{
    last_command="angler_tokens_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--comment=")
    two_word_flags+=("--comment")
    local_nonpersistent_flags+=("--comment")
    local_nonpersistent_flags+=("--comment=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--lifetime-days=")
    two_word_flags+=("--lifetime-days")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--lifetime-days")
    local_nonpersistent_flags+=("--lifetime-days=")
    local_nonpersistent_flags+=("-d")
    flags+=("--name=")
    two_word_flags+=("--name")
    two_word_flags+=("-n")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    local_nonpersistent_flags+=("-n")
    flags+=("--role=")
    two_word_flags+=("--role")
    two_word_flags+=("-r")
    local_nonpersistent_flags+=("--role")
    local_nonpersistent_flags+=("--role=")
    local_nonpersistent_flags+=("-r")
    flags+=("--scopes=")
    two_word_flags+=("--scopes")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--scopes")
    local_nonpersistent_flags+=("--scopes=")
    local_nonpersistent_flags+=("-s")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    local_nonpersistent_flags+=("--workspace")
    local_nonpersistent_flags+=("--workspace=")
    local_nonpersistent_flags+=("-w")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_tokens_help()
{
    last_command="angler_tokens_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_tokens_list()
{
    last_command="angler_tokens_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    flags+=("-a")
    local_nonpersistent_flags+=("--all")
    local_nonpersistent_flags+=("-a")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_tokens_revoke()
{
    last_command="angler_tokens_revoke"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_tokens()
{
    last_command="angler_tokens"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("revoke")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("delete")
        aliashash["delete"]="revoke"
        command_aliases+=("rm")
        aliashash["rm"]="revoke"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_tool-defs()
{
    last_command="angler_tool-defs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--format=")
    two_word_flags+=("--format")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--include-global-flags")
    local_nonpersistent_flags+=("--include-global-flags")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_up()
{
    last_command="angler_up"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--detach")
    flags+=("-d")
    local_nonpersistent_flags+=("--detach")
    local_nonpersistent_flags+=("-d")
    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_users_get()
{
    last_command="angler_users_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_users_help()
{
    last_command="angler_users_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_users_list()
{
    last_command="angler_users_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_users_update()
{
    last_command="angler_users_update"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--active")
    local_nonpersistent_flags+=("--active")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--role=")
    two_word_flags+=("--role")
    local_nonpersistent_flags+=("--role")
    local_nonpersistent_flags+=("--role=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_users()
{
    last_command="angler_users"

    command_aliases=()

    commands=()
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("update")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_version()
{
    last_command="angler_version"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_webhooks_help()
{
    last_command="angler_webhooks_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_webhooks_secret_get()
{
    last_command="angler_webhooks_secret_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--raw")
    local_nonpersistent_flags+=("--raw")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_webhooks_secret_help()
{
    last_command="angler_webhooks_secret_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_webhooks_secret_rotate()
{
    last_command="angler_webhooks_secret_rotate"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_webhooks_secret()
{
    last_command="angler_webhooks_secret"

    command_aliases=()

    commands=()
    commands+=("get")
    commands+=("help")
    commands+=("rotate")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("regenerate")
        aliashash["regenerate"]="rotate"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_webhooks_send()
{
    last_command="angler_webhooks_send"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--file=")
    two_word_flags+=("--file")
    two_word_flags+=("-f")
    local_nonpersistent_flags+=("--file")
    local_nonpersistent_flags+=("--file=")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--hmac")
    local_nonpersistent_flags+=("--hmac")
    flags+=("--payload=")
    two_word_flags+=("--payload")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--payload")
    local_nonpersistent_flags+=("--payload=")
    local_nonpersistent_flags+=("-d")
    flags+=("--secret=")
    two_word_flags+=("--secret")
    local_nonpersistent_flags+=("--secret")
    local_nonpersistent_flags+=("--secret=")
    flags+=("--trigger-pipeline=")
    two_word_flags+=("--trigger-pipeline")
    local_nonpersistent_flags+=("--trigger-pipeline")
    local_nonpersistent_flags+=("--trigger-pipeline=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_webhooks()
{
    last_command="angler_webhooks"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("secret")
    commands+=("send")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ingest")
        aliashash["ingest"]="send"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_delete()
{
    last_command="angler_workspace_delete"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--recursive")
    flags+=("-r")
    local_nonpersistent_flags+=("--recursive")
    local_nonpersistent_flags+=("-r")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_export()
{
    last_command="angler_workspace_export"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dest=")
    two_word_flags+=("--dest")
    local_nonpersistent_flags+=("--dest")
    local_nonpersistent_flags+=("--dest=")
    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_help()
{
    last_command="angler_workspace_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_workspace_import()
{
    last_command="angler_workspace_import"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--overwrite")
    local_nonpersistent_flags+=("--overwrite")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_list()
{
    last_command="angler_workspace_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--format=")
    two_word_flags+=("--format")
    local_nonpersistent_flags+=("--format")
    local_nonpersistent_flags+=("--format=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--recursive")
    flags+=("-r")
    local_nonpersistent_flags+=("--recursive")
    local_nonpersistent_flags+=("-r")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_mkdirs()
{
    last_command="angler_workspace_mkdirs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_run()
{
    last_command="angler_workspace_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--kernel=")
    two_word_flags+=("--kernel")
    two_word_flags+=("-k")
    local_nonpersistent_flags+=("--kernel")
    local_nonpersistent_flags+=("--kernel=")
    local_nonpersistent_flags+=("-k")
    flags+=("--save=")
    two_word_flags+=("--save")
    two_word_flags+=("-s")
    local_nonpersistent_flags+=("--save")
    local_nonpersistent_flags+=("--save=")
    local_nonpersistent_flags+=("-s")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_runs()
{
    last_command="angler_workspace_runs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_search()
{
    last_command="angler_workspace_search"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--subpath=")
    two_word_flags+=("--subpath")
    local_nonpersistent_flags+=("--subpath")
    local_nonpersistent_flags+=("--subpath=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_sessions_help()
{
    last_command="angler_workspace_sessions_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_workspace_sessions_interrupt()
{
    last_command="angler_workspace_sessions_interrupt"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_sessions_restart()
{
    last_command="angler_workspace_sessions_restart"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_sessions_start()
{
    last_command="angler_workspace_sessions_start"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--kernel=")
    two_word_flags+=("--kernel")
    two_word_flags+=("-k")
    local_nonpersistent_flags+=("--kernel")
    local_nonpersistent_flags+=("--kernel=")
    local_nonpersistent_flags+=("-k")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace_sessions()
{
    last_command="angler_workspace_sessions"

    command_aliases=()

    commands=()
    commands+=("help")
    commands+=("interrupt")
    commands+=("restart")
    commands+=("start")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspace()
{
    last_command="angler_workspace"

    command_aliases=()

    commands=()
    commands+=("delete")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="delete"
    fi
    commands+=("export")
    commands+=("help")
    commands+=("import")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("mkdirs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("mkdir")
        aliashash["mkdir"]="mkdirs"
    fi
    commands+=("run")
    commands+=("runs")
    commands+=("search")
    commands+=("sessions")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_create()
{
    last_command="angler_workspaces_create"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--description=")
    two_word_flags+=("--description")
    local_nonpersistent_flags+=("--description")
    local_nonpersistent_flags+=("--description=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--slug=")
    two_word_flags+=("--slug")
    local_nonpersistent_flags+=("--slug")
    local_nonpersistent_flags+=("--slug=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_get()
{
    last_command="angler_workspaces_get"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_help()
{
    last_command="angler_workspaces_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_workspaces_list()
{
    last_command="angler_workspaces_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_members_add()
{
    last_command="angler_workspaces_members_add"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--email=")
    two_word_flags+=("--email")
    local_nonpersistent_flags+=("--email")
    local_nonpersistent_flags+=("--email=")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--role=")
    two_word_flags+=("--role")
    local_nonpersistent_flags+=("--role")
    local_nonpersistent_flags+=("--role=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_members_help()
{
    last_command="angler_workspaces_members_help"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    has_completion_function=1
    noun_aliases=()
}

_angler_workspaces_members_list()
{
    last_command="angler_workspaces_members_list"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_members_remove()
{
    last_command="angler_workspaces_members_remove"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--force")
    flags+=("-f")
    local_nonpersistent_flags+=("--force")
    local_nonpersistent_flags+=("-f")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces_members()
{
    last_command="angler_workspaces_members"

    command_aliases=()

    commands=()
    commands+=("add")
    commands+=("help")
    commands+=("list")
    commands+=("remove")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("rm")
        aliashash["rm"]="remove"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_workspaces()
{
    last_command="angler_workspaces"

    command_aliases=()

    commands=()
    commands+=("create")
    commands+=("get")
    commands+=("help")
    commands+=("list")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ls")
        aliashash["ls"]="list"
    fi
    commands+=("members")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("member")
        aliashash["member"]="members"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_angler_root_command()
{
    last_command="angler"

    command_aliases=()

    commands=()
    commands+=("admin")
    commands+=("ai")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("genie")
        aliashash["genie"]="ai"
    fi
    commands+=("ask")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("sonar")
        aliashash["sonar"]="ask"
    fi
    commands+=("auth")
    commands+=("bundle")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("bundles")
        aliashash["bundles"]="bundle"
        command_aliases+=("lab")
        aliashash["lab"]="bundle"
    fi
    commands+=("catalog")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("net")
        aliashash["net"]="catalog"
    fi
    commands+=("completion")
    commands+=("compute")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("clusters")
        aliashash["clusters"]="compute"
    fi
    commands+=("context")
    commands+=("dashboards")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("dash")
        aliashash["dash"]="dashboards"
        command_aliases+=("dashboard")
        aliashash["dashboard"]="dashboards"
    fi
    commands+=("docs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("doc")
        aliashash["doc"]="docs"
    fi
    commands+=("down")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("dock")
        aliashash["dock"]="down"
    fi
    commands+=("fs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("storage")
        aliashash["storage"]="fs"
    fi
    commands+=("help")
    commands+=("jobs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("job")
        aliashash["job"]="jobs"
        command_aliases+=("pipeline")
        aliashash["pipeline"]="jobs"
        command_aliases+=("pipelines")
        aliashash["pipelines"]="jobs"
    fi
    commands+=("lakeset")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("bi")
        aliashash["bi"]="lakeset"
    fi
    commands+=("mcp")
    commands+=("query")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("cast")
        aliashash["cast"]="query"
    fi
    commands+=("secrets")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("secret")
        aliashash["secret"]="secrets"
    fi
    commands+=("sql")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("cast")
        aliashash["cast"]="sql"
        command_aliases+=("query")
        aliashash["query"]="sql"
    fi
    commands+=("status")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ping")
        aliashash["ping"]="status"
    fi
    commands+=("tokens")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("pat")
        aliashash["pat"]="tokens"
        command_aliases+=("token")
        aliashash["token"]="tokens"
    fi
    commands+=("tool-defs")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("schema")
        aliashash["schema"]="tool-defs"
        command_aliases+=("tools")
        aliashash["tools"]="tool-defs"
    fi
    commands+=("up")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("launch")
        aliashash["launch"]="up"
    fi
    commands+=("users")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("user")
        aliashash["user"]="users"
    fi
    commands+=("version")
    commands+=("webhooks")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("webhook")
        aliashash["webhook"]="webhooks"
    fi
    commands+=("workspace")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ws")
        aliashash["ws"]="workspace"
    fi
    commands+=("workspaces")
    if [[ -z "${BASH_VERSION:-}" || "${BASH_VERSINFO[0]:-}" -gt 3 ]]; then
        command_aliases+=("ws-admin")
        aliashash["ws-admin"]="workspaces"
    fi

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--context=")
    two_word_flags+=("--context")
    two_word_flags+=("-c")
    flags+=("--endpoint=")
    two_word_flags+=("--endpoint")
    flags+=("--force")
    flags+=("--help")
    flags+=("-h")
    local_nonpersistent_flags+=("--help")
    local_nonpersistent_flags+=("-h")
    flags+=("--json")
    flags+=("--non-interactive")
    flags+=("--on-behalf-of=")
    two_word_flags+=("--on-behalf-of")
    flags+=("--output=")
    two_word_flags+=("--output")
    two_word_flags+=("-o")
    flags+=("--token=")
    two_word_flags+=("--token")
    two_word_flags+=("-t")
    flags+=("--workspace=")
    two_word_flags+=("--workspace")
    two_word_flags+=("-w")
    flags+=("--workspace-id=")
    two_word_flags+=("--workspace-id")
    flags+=("--yes")
    flags+=("-y")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

__start_angler()
{
    local cur prev words cword split
    declare -A flaghash 2>/dev/null || :
    declare -A aliashash 2>/dev/null || :
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -s || return
    else
        __angler_init_completion -n "=" || return
    fi

    local c=0
    local flag_parsing_disabled=
    local flags=()
    local two_word_flags=()
    local local_nonpersistent_flags=()
    local flags_with_completion=()
    local flags_completion=()
    local commands=("angler")
    local command_aliases=()
    local must_have_one_flag=()
    local must_have_one_noun=()
    local has_completion_function=""
    local last_command=""
    local nouns=()
    local noun_aliases=()

    __angler_handle_word
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_angler angler
else
    complete -o default -o nospace -F __start_angler angler
fi

# ex: ts=4 sw=4 et filetype=sh
