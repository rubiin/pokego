if [[ $BASH_VERSINFO -lt 4 ]]; then
    echo "This completion script requires bash 4.0 or newer (current is $BASH_VERSION)"
    exit 1
fi

__complgen_match () {
    [[ $# -lt 2 ]] && return 1
    local ignore_case=$1
    local prefix=$2
    [[ -z $prefix ]] && cat
    if [[ $ignore_case = on ]]; then
        prefix=${prefix,,}
        prefix=$(printf '%q' "$prefix")
        while read line; do
            [[ ${line,,} = ${prefix}* ]] && echo $line
        done
    else
        prefix=$(printf '%q' "$prefix")
        while read line; do
            [[ $line = ${prefix}* ]] && echo $line
        done
    fi
}

_pokego_subword () {
    [[ $# -ne 2 ]] && return 1
    local mode=$1
    local word=$2

    local char_index=0
    local matched=0
    while true; do
        if [[ $char_index -ge ${#word} ]]; then
            matched=1
            break
        fi

        local subword=${word:$char_index}

        if [[ -v "literal_transitions[$state]" ]]; then
            declare -A state_transitions
            eval "state_transitions=${literal_transitions[$state]}"

            local literal_matched=0
            for literal_id in $(seq 0 $((${#literals[@]} - 1))); do
                local literal=${literals[$literal_id]}
                local literal_len=${#literal}
                if [[ ${subword:0:$literal_len} = "$literal" ]]; then
                    if [[ -v "state_transitions[$literal_id]" ]]; then
                        state=${state_transitions[$literal_id]}
                        char_index=$((char_index + literal_len))
                        literal_matched=1
                        break
                    fi
                fi
            done
            if [[ $literal_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "nontail_transitions[$state]" ]]; then
            declare -A state_nontails
            eval "state_nontails=${nontail_transitions[$state]}"
            local nontail_matched=0
            for regex_id in "${!state_nontails[@]}"; do
                local regex="^(${regexes[$regex_id]}).*"
                if [[ ${subword} =~ $regex && -n ${BASH_REMATCH[1]} ]]; then
                    match="${BASH_REMATCH[1]}"
                    match_len=${#match}
                    char_index=$((char_index + match_len))
                    state=${state_nontails[$regex_id]}
                    nontail_matched=1
                    break
                fi
            done
            if [[ $nontail_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "match_anything_transitions[$state]" ]]; then
            matched=1
            break
        fi

        break
    done

    if [[ $mode = matches ]]; then
        return $((1 - matched))
    fi


    local matched_prefix="${word:0:$char_index}"
    local completed_prefix="${word:$char_index}"

    local -a candidates=()
    local -a matches=()
    local ignore_case=$(bind -v | grep completion-ignore-case | cut -d' ' -f3)
    for (( subword_fallback_level=0; subword_fallback_level <= max_fallback_level; subword_fallback_level++ )) {
        eval "declare literal_transitions_name=literal_transitions_level_${subword_fallback_level}"
        eval "declare -a transitions=(\${$literal_transitions_name[$state]})"
        for literal_id in "${transitions[@]}"; do
            local literal=${literals[$literal_id]}
            candidates+=("$matched_prefix$literal")
        done
        if [[ ${#candidates[@]} -gt 0 ]]; then
            readarray -t matches < <(printf "%s\n" "${candidates[@]}" | __complgen_match "$ignore_case" "$matched_prefix$completed_prefix")
        fi

        eval "declare commands_name=commands_level_${subword_fallback_level}"
        eval "declare -a transitions=(\${$commands_name[$state]})"
        for command_id in "${transitions[@]}"; do
            readarray -t candidates < <(_pokego_cmd_$command_id "$matched_prefix" | cut -f1)
            for item in "${candidates[@]}"; do
                matches+=("$matched_prefix$item")
            done
        done

        eval "declare commands_name=nontail_commands_level_${subword_fallback_level}"
        eval "declare -a command_transitions=(\${$commands_name[$state]})"
        eval "declare regexes_name=nontail_regexes_level_${fallback_level}"
        eval "declare -a regexes_transitions=(\${$regexes_name[$state]})"
        for (( i = 0; i < ${#command_transitions[@]}; i++ )); do
            local command_id=${command_transitions[$i]}
            readarray -t output < <(_pokego_cmd_$command_id "$matched_prefix" | cut -f1)
            local regex_id=${regexes_transitions[$i]}
            local regex="^(${regexes[$regex_id]}).*"
            declare -a candidates=()
            for line in ${output[@]}; do
                if [[ ${line} =~ $regex && -n ${BASH_REMATCH[1]} ]]; then
                    match="${BASH_REMATCH[1]}"
                    candidates+=("$match")
                fi
            done
            for item in "${candidates[@]}"; do
                matches+=("$matched_prefix$item")
            done
        done

        if [[ ${#matches[@]} -gt 0 ]]; then
            printf '%s\n' "${matches[@]}"
            break
        fi
    }
    return 0
}

_pokego_subword_0 () {
    declare -a literals=(--name=)
    declare -a regexes=()
    declare -A literal_transitions=()
    declare -A nontail_transitions=()
    literal_transitions[0]="([0]=1)"
    declare -A match_anything_transitions=([1]=2)
    declare -A literal_transitions_level_0=([0]="0")
    declare -A nontail_commands_level_0=()
    declare -A nontail_regexes_level_0=()
    declare -A commands_level_0=()
    declare max_fallback_level=0
    declare state=0
    _pokego_subword "$1" "$2"
}

_pokego () {
    if [[ $(type -t _get_comp_words_by_ref) != function ]]; then
        echo _get_comp_words_by_ref: function not defined.  Make sure the bash-completion system package is installed
        return 1
    fi

    local words cword
    _get_comp_words_by_ref -n "$COMP_WORDBREAKS" words cword

    declare -a literals=(--help -h --form --list --no-title --random --shiny --version -V)
    declare -a regexes=()
    declare -A literal_transitions=()
    declare -A nontail_transitions=()
    literal_transitions[1]="([0]=2 [1]=2 [2]=2 [3]=2 [4]=2 [5]=2 [6]=2 [7]=2 [8]=2)"
    declare -A match_anything_transitions=()
    declare -A subword_transitions
    subword_transitions[1]="([0]=2)"

    local state=1
    local word_index=1
    while [[ $word_index -lt $cword ]]; do
        local word=${words[$word_index]}

        if [[ -v "literal_transitions[$state]" ]]; then
            declare -A state_transitions
            eval "state_transitions=${literal_transitions[$state]}"

            local word_matched=0
            for literal_id in $(seq 0 $((${#literals[@]} - 1))); do
                if [[ ${literals[$literal_id]} = "$word" ]]; then
                    if [[ -v "state_transitions[$literal_id]" ]]; then
                        state=${state_transitions[$literal_id]}
                        word_index=$((word_index + 1))
                        word_matched=1
                        break
                    fi
                fi
            done
            if [[ $word_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "subword_transitions[$state]" ]]; then
            declare -A state_transitions
            eval "state_transitions=${subword_transitions[$state]}"

            local subword_matched=0
            for subword_id in "${!state_transitions[@]}"; do
                if _pokego_subword_"${subword_id}" matches "$word"; then
                    subword_matched=1
                    state=${state_transitions[$subword_id]}
                    word_index=$((word_index + 1))
                    break
                fi
            done
            if [[ $subword_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "match_anything_transitions[$state]" ]]; then
            state=${match_anything_transitions[$state]}
            word_index=$((word_index + 1))
            continue
        fi

        return 1
    done

    declare -A literal_transitions_level_0=([1]="0 2 3 4 5 6 7")
    declare -A literal_transitions_level_1=([1]="1 8")
    declare -A subword_transitions_level_0=([1]="0")
    declare -A subword_transitions_level_1=()
    declare -A commands_level_0=()
    declare -A commands_level_1=()
    declare -A nontail_commands_level_0=()
    declare -A nontail_regexes_level_0=()
    declare -A nontail_commands_level_1=()
    declare -A nontail_regexes_level_1=()

    local -a candidates=()
    local -a matches=()
    local ignore_case=$(bind -v | grep completion-ignore-case | cut -d' ' -f3)
    local max_fallback_level=1
    local prefix="${words[$cword]}"
    for (( fallback_level=0; fallback_level <= max_fallback_level; fallback_level++ )) {
        eval "declare literal_transitions_name=literal_transitions_level_${fallback_level}"
        eval "declare -a transitions=(\${$literal_transitions_name[$state]})"
        for literal_id in "${transitions[@]}"; do
            local literal="${literals[$literal_id]}"
            candidates+=("$literal ")
        done
        if [[ ${#candidates[@]} -gt 0 ]]; then
            readarray -t matches < <(printf "%s\n" "${candidates[@]}" | __complgen_match "$ignore_case" "$prefix")
        fi

        eval "declare subword_transitions_name=subword_transitions_level_${fallback_level}"
        eval "declare -a transitions=(\${$subword_transitions_name[$state]})"
        for subword_id in "${transitions[@]}"; do
            readarray -t -O "${#matches[@]}" matches < <(_pokego_subword_$subword_id complete "$prefix")
        done

        eval "declare commands_name=commands_level_${fallback_level}"
        eval "declare -a transitions=(\${$commands_name[$state]})"
        for command_id in "${transitions[@]}"; do
            readarray -t candidates < <(_pokego_cmd_$command_id "$prefix" | cut -f1)
            if [[ ${#candidates[@]} -gt 0 ]]; then
                readarray -t -O "${#matches[@]}" matches < <(printf "%s\n" "${candidates[@]}" | __complgen_match "$ignore_case" "$prefix")
            fi
        done

        eval "declare commands_name=nontail_commands_level_${fallback_level}"
        eval "declare -a command_transitions=(\${$commands_name[$state]})"
        eval "declare regexes_name=nontail_regexes_level_${fallback_level}"
        eval "declare -a regexes_transitions=(\${$regexes_name[$state]})"
        for (( i = 0; i < ${#command_transitions[@]}; i++ )); do
            local command_id=${command_transitions[$i]}
            local regex_id=${regexes_transitions[$i]}
            local regex="^(${regexes[$regex_id]}).*"
            readarray -t output < <(_pokego_cmd_$command_id "$prefix" | cut -f1)
            declare -a candidates=()
            for line in ${output[@]}; do
                if [[ ${line} =~ $regex && -n ${BASH_REMATCH[1]} ]]; then
                    match="${BASH_REMATCH[1]}"
                    candidates+=("$match")
                fi
            done
            if [[ ${#candidates[@]} -gt 0 ]]; then
                readarray -t -O "${#matches[@]}" matches < <(printf "%s\n" "${candidates[@]}" | __complgen_match "$ignore_case" "$prefix")
            fi
        done

        if [[ ${#matches[@]} -gt 0 ]]; then
            local shortest_suffix="$prefix"
            for ((i=0; i < ${#COMP_WORDBREAKS}; i++)); do
                local char="${COMP_WORDBREAKS:$i:1}"
                local candidate=${prefix##*$char}
                if [[ ${#candidate} -lt ${#shortest_suffix} ]]; then
                    shortest_suffix=$candidate
                fi
            done
            local superfluous_prefix=""
            if [[ "$shortest_suffix" != "$prefix" ]]; then
                local superfluous_prefix=${prefix%$shortest_suffix}
            fi
            COMPREPLY=("${matches[@]#$superfluous_prefix}")
            break
        fi
    }

    return 0
}

complete -o nospace -F _pokego pokego
