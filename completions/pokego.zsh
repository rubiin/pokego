#compdef pokego

_pokego_subword () {
    declare mode=$1
    declare word=$2

    declare char_index=0
    declare matched=0
    while true; do
        if [[ $char_index -ge ${#word} ]]; then
            matched=1
            break
        fi

        declare subword=${word:$char_index}

        if [[ -v "subword_literal_transitions[$subword_state]" ]]; then
            eval "declare -A state_transitions=${subword_literal_transitions[$subword_state]}"

            declare literal_matched=0
            for ((literal_id = 1; literal_id <= $#subword_literals; literal_id++)); do
                declare literal=${subword_literals[$literal_id]}
                declare literal_len=${#literal}
                if [[ ${subword:0:$literal_len} = "$literal" ]]; then
                    if [[ -v "state_transitions[$literal_id]" ]]; then
                        subword_state=${state_transitions[$literal_id]}
                        char_index=$((char_index + literal_len))
                        literal_matched=1
                    fi
                fi
            done
            if [[ $literal_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "subword_nontail_transitions[$subword_state]" ]]; then
            eval "declare -A state_nontails=${subword_nontail_transitions[$subword_state]}"

            declare nontail_matched=0
            for regex_id in "${(k)state_nontails}"; do
                declare regex="^(${subword_regexes[$regex_id]}).*"
                if [[ ${subword} =~ $regex && -n ${match[1]} ]]; then
                    match="${match[1]}"
                    match_len=${#match}
                    char_index=$((char_index + match_len))
                    subword_state=${state_nontails[$regex_id]}
                    nontail_matched=1
                    break
                fi
            done
            if [[ $nontail_matched -ne 0 ]]; then
                continue
            fi
        fi

        if [[ -v "subword_match_anything_transitions[$subword_state]" ]]; then
            subword_state=${subword_match_anything_transitions[$subword_state]}
            matched=1
            break
        fi

        break
    done

    if [[ $mode = matches ]]; then
        return $((1 - matched))
    fi

    declare matched_prefix="${word:0:$char_index}"
    declare completed_prefix="${word:$char_index}"

    subword_completions_no_description_trailing_space=()
    subword_completions_trailing_space=()
    subword_completions_no_trailing_space=()
    subword_suffixes_trailing_space=()
    subword_suffixes_no_trailing_space=()
    subword_descriptions_trailing_space=()
    subword_descriptions_no_trailing_space=()

    for (( subword_fallback_level=0; subword_fallback_level <= subword_max_fallback_level; subword_fallback_level++ )); do
        declare literal_transitions_name=subword_literal_transitions_level_${subword_fallback_level}
        eval "declare initializer=\${${literal_transitions_name}[$subword_state]}"
        eval "declare -a transitions=($initializer)"
        for literal_id in "${transitions[@]}"; do
            declare literal=${subword_literals[$literal_id]}
            if [[ $literal = "${completed_prefix}"* ]]; then
                declare completion="$matched_prefix$literal"
                if [[ -v "subword_descr_id_from_literal_id[$literal_id]" ]]; then
                    declare descr_id=$subword_descr_id_from_literal_id[$literal_id]
                    subword_completions_no_trailing_space+=("${completion}")
                    subword_suffixes_no_trailing_space+=("${completion}")
                    subword_descriptions_no_trailing_space+=("${subword_descrs[$descr_id]}")
                else
                    subword_completions_no_trailing_space+=("${completion}")
                    subword_suffixes_no_trailing_space+=("${literal}")
                    subword_descriptions_no_trailing_space+=('')
                fi
            fi
        done

        declare commands_name=subword_nontail_commands_level_${subword_fallback_level}
        eval "declare commands_initializer=\${${commands_name}[$subword_state]}"
        eval "declare -a command_transitions=($commands_initializer)"
        declare regexes_name=subword_nontail_regexes_level_${subword_fallback_level}
        eval "declare regexes_initializer=\${${regexes_name}[$subword_state]}"
        eval "declare -a regexes_transitions=($regexes_initializer)"
        for (( i=1; i <= ${#command_transitions[@]}; i++ )); do
            declare command_id=${command_transitions[$i]}
            declare regex_id=${regexes_transitions[$i]}
            declare regex="^(${subword_regexes[$regex_id]}).*"
            declare candidates=()
            declare output=$(_pokego_cmd_${command_id} "$matched_prefix")
            declare -a command_completions=("${(@f)output}")
            for line in ${command_completions[@]}; do
                if [[ $line = "${completed_prefix}"* ]]; then
                    declare parts=(${(@s:	:)line})
                    if [[ ${parts[1]} =~ $regex && -n ${match[1]} ]]; then
                        parts[1]=${match[1]}
                        if [[ -v "parts[2]" ]]; then
                            declare completion=$matched_prefix${parts[1]}
                            subword_completions_trailing_space+=("${completion}")
                            subword_suffixes_trailing_space+=("${parts[1]}")
                            subword_descriptions_trailing_space+=("${parts[2]}")
                        else
                            subword_completions_no_description_trailing_space+=("$matched_prefix${parts[1]}")
                        fi
                    fi
                fi
            done
        done

        declare commands_name=subword_commands_level_${subword_fallback_level}
        eval "declare initializer=\${${commands_name}[$subword_state]}"
        eval "declare -a transitions=($initializer)"
        for command_id in "${transitions[@]}"; do
            declare candidates=()
            declare output=$(_pokego_cmd_${command_id} "$matched_prefix")
            declare -a command_completions=("${(@f)output}")
            for line in ${command_completions[@]}; do
                if [[ $line = "${completed_prefix}"* ]]; then
                    declare parts=(${(@s:	:)line})
                    if [[ -v "parts[2]" ]]; then
                        declare completion=$matched_prefix${parts[1]}
                        subword_completions_trailing_space+=("${completion}")
                        subword_suffixes_trailing_space+=("${parts[1]}")
                        subword_descriptions_trailing_space+=("${parts[2]}")
                    else
                        line="$matched_prefix$line"
                        subword_completions_no_description_trailing_space+=("$line")
                    fi
                fi
            done
        done

        declare specialized_commands_name=subword_specialized_commands_level_${subword_fallback_level}
        eval "declare initializer=\${${specialized_commands_name}[$subword_state]}"
        eval "declare -a transitions=($initializer)"
        for command_id in "${transitions[@]}"; do
            declare output=$(_pokego_cmd_${command_id} "$matched_prefix")
            declare -a candidates=("${(@f)output}")
            for line in ${candidates[@]}; do
                if [[ $line = "${completed_prefix}"* ]]; then
                    line="$matched_prefix$line"
                    declare parts=(${(@s:	:)line})
                    if [[ -v "parts[2]" ]]; then
                        subword_completions_trailing_space+=("${parts[1]}")
                        subword_suffixes_trailing_space+=("${parts[1]}")
                        subword_descriptions_trailing_space+=("${parts[2]}")
                    else
                        subword_completions_no_description_trailing_space+=("$line")
                    fi
                fi
            done
        done

        if [[ ${#subword_completions_no_description_trailing_space} -gt 0 || ${#subword_completions_trailing_space} -gt 0 || ${#subword_completions_no_trailing_space} -gt 0 ]]; then
            break
        fi
    done
    return 0
}

_pokego_subword_1 () {
    declare -a subword_literals=("--name=")
    declare -A subword_descrs=()
    subword_descrs[0]="show information for the given Pokémon NAME"
    declare -A subword_descr_id_from_literal_id=([1]=0)
    declare -a subword_regexes=()
    declare -A subword_literal_transitions=()
    subword_literal_transitions[1]="([1]=2)"
    declare -A subword_nontail_transitions=()
    declare -A subword_match_anything_transitions=([2]=3)
    declare -A subword_literal_transitions_level_0=([1]="1")
    declare -A subword_commands_level_0=()
    declare -A subword_specialized_commands_level_0=()
    declare -A subword_nontail_commands_level_0=()
    declare -A subword_nontail_regexes_level_0=()
    declare subword_max_fallback_level=0
    declare subword_state=1
    _pokego_subword "$@"
}

_pokego () {
    declare -a literals=("--help" "-h" "--form" "--list" "--no-title" "--random" "--shiny" "--version" "-V")
    declare -A descrs=()
    descrs[0]="display this help text and exit"
    descrs[1]="display Pokémon with alternate forms"
    descrs[2]="list all available Pokémon"
    descrs[3]="suppress title/header in output"
    descrs[4]="display a random Pokémon"
    descrs[5]="display the shiny version if available"
    descrs[6]="display version information and exit"
    declare -A descr_id_from_literal_id=([1]=0 [3]=1 [4]=2 [5]=3 [6]=4 [7]=5 [8]=6)
    declare -a regexes=()
    declare -A literal_transitions=()
    literal_transitions[2]="([1]=3 [2]=3 [3]=3 [4]=3 [5]=3 [6]=3 [7]=3 [8]=3 [9]=3)"
    declare -A nontail_transitions=()
    declare -A match_anything_transitions=()
    declare -A subword_transitions=()
    subword_transitions[2]="([1]=3)"

    declare state=2
    declare word_index=2
    while [[ $word_index -lt $CURRENT ]]; do
        if [[ -v "literal_transitions[$state]" ]]; then
            eval "declare -A state_transitions=${literal_transitions[$state]}"

            declare word=${words[$word_index]}
            declare word_matched=0
            for ((literal_id = 1; literal_id <= $#literals; literal_id++)); do
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

        if [[ -v "nontail_transitions[$state]" ]]; then
            eval "declare -A state_nontails=${nontail_transitions[$state]}"
            declare nontail_matched=0
            for regex_id in "${(k)state_nontails}"; do
                declare regex="^(${regexes[$regex_id]}).*"
                if [[ ${subword} =~ $regex && -n ${match[1]} ]]; then
                    match="${match[1]}"
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


        if [[ -v "subword_transitions[$state]" ]]; then
            eval "declare -A state_transitions=${subword_transitions[$state]}"

            declare subword_matched=0
            for subword_id in ${(k)state_transitions}; do
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

    declare -A literal_transitions_level_0=([2]="1 3 4 5 6 7 8")
    declare -A literal_transitions_level_1=([2]="2 9")
    declare -A subword_transitions_level_0=([2]="1")
    declare -A subword_transitions_level_1=()
    declare -A commands_level_0=()
    declare -A commands_level_1=()
    declare -A nontail_commands_level_0=()
    declare -A nontail_regexes_level_0=()
    declare -A nontail_commands_level_1=()
    declare -A nontail_regexes_level_1=()
    declare -A specialized_commands_level_0=()
    declare -A specialized_commands_level_1=()

    declare max_fallback_level=1
    for (( fallback_level=0; fallback_level <= max_fallback_level; fallback_level++ )); do
        completions_no_description_trailing_space=()
        completions_no_description_no_trailing_space=()
        completions_trailing_space=()
        suffixes_trailing_space=()
        descriptions_trailing_space=()
        completions_no_trailing_space=()
        suffixes_no_trailing_space=()
        descriptions_no_trailing_space=()
        matches=()

        declare literal_transitions_name=literal_transitions_level_${fallback_level}
        eval "declare initializer=\${${literal_transitions_name}[$state]}"
        eval "declare -a transitions=($initializer)"
        for literal_id in "${transitions[@]}"; do
            if [[ -v "descr_id_from_literal_id[$literal_id]" ]]; then
                declare descr_id=$descr_id_from_literal_id[$literal_id]
                completions_trailing_space+=("${literals[$literal_id]}")
                suffixes_trailing_space+=("${literals[$literal_id]}")
                descriptions_trailing_space+=("${descrs[$descr_id]}")
            else
                completions_no_description_trailing_space+=("${literals[$literal_id]}")
            fi
        done

        declare subword_transitions_name=subword_transitions_level_${fallback_level}
        eval "declare initializer=\${${subword_transitions_name}[$state]}"
        eval "declare -a transitions=($initializer)"
        for subword_id in "${transitions[@]}"; do
            _pokego_subword_${subword_id} complete "${words[$CURRENT]}"
            completions_no_description_trailing_space+=("${subword_completions_no_description_trailing_space[@]}")
            completions_trailing_space+=("${subword_completions_trailing_space[@]}")
            completions_no_trailing_space+=("${subword_completions_no_trailing_space[@]}")
            suffixes_no_trailing_space+=("${subword_suffixes_no_trailing_space[@]}")
            suffixes_trailing_space+=("${subword_suffixes_trailing_space[@]}")
            descriptions_trailing_space+=("${subword_descriptions_trailing_space[@]}")
            descriptions_no_trailing_space+=("${subword_descriptions_no_trailing_space[@]}")
        done

        declare commands_name=commands_level_${fallback_level}
        eval "declare initializer=\${${commands_name}[$state]}"
        eval "declare -a transitions=($initializer)"
        for command_id in "${transitions[@]}"; do
            declare output=$(_pokego_cmd_${command_id} "${words[$CURRENT]}")
            declare -a command_completions=("${(@f)output}")
            for line in ${command_completions[@]}; do
                declare parts=(${(@s:	:)line})
                if [[ -v "parts[2]" ]]; then
                    completions_trailing_space+=("${parts[1]}")
                    suffixes_trailing_space+=("${parts[1]}")
                    descriptions_trailing_space+=("${parts[2]}")
                else
                    completions_no_description_trailing_space+=("${parts[1]}")
                fi
            done
        done

        declare commands_name=nontail_commands_level_${fallback_level}
        eval "declare command_initializer=\${${commands_name}[$state]}"
        eval "declare -a command_transitions=($command_initializer)"
        declare regexes_name=nontail_regexes_level_${fallback_level}
        eval "declare regexes_initializer=\${${regexes_name}[$state]}"
        eval "declare -a regexes_transitions=($regexes_initializer)"
        for (( i=1; i <= ${#command_transitions[@]}; i++ )); do
            declare command_id=${command_transitions[$i]}
            declare regex_id=${regexes_transitions[$i]}
            declare regex="^(${regexes[$regex_id]}).*"
            declare output=$(_pokego_cmd_${command_id} "${words[$CURRENT]}")
            declare -a command_completions=("${(@f)output}")
            for line in ${command_completions[@]}; do
                declare parts=(${(@s:	:)line})
                if [[ ${parts[1]} =~ $regex && -n ${match[1]} ]]; then
                    parts[1]=${match[1]}
                    if [[ -v "parts[2]" ]]; then
                        completions_trailing_space+=("${parts[1]}")
                        suffixes_trailing_space+=("${parts[1]}")
                        descriptions_trailing_space+=("${parts[2]}")
                    else
                        completions_no_description_trailing_space+=("${parts[1]}")
                    fi
                fi
            done
        done

        declare specialized_commands_name=specialized_commands_level_${fallback_level}
        eval "declare initializer=\${${specialized_commands_name}[$state]}"
        eval "declare -a transitions=($initializer)"
        for command_id in "${transitions[@]}"; do
            _pokego_cmd_${command_id} ${words[$CURRENT]}
        done

        declare maxlen=0
        for suffix in ${suffixes_trailing_space[@]}; do
            if [[ ${#suffix} -gt $maxlen ]]; then
                maxlen=${#suffix}
            fi
        done
        for suffix in ${suffixes_no_trailing_space[@]}; do
            if [[ ${#suffix} -gt $maxlen ]]; then
                maxlen=${#suffix}
            fi
        done

        for ((i = 1; i <= $#suffixes_trailing_space; i++)); do
            if [[ -z ${descriptions_trailing_space[$i]} ]]; then
                descriptions_trailing_space[$i]="${(r($maxlen)( ))${suffixes_trailing_space[$i]}}"
            else
                descriptions_trailing_space[$i]="${(r($maxlen)( ))${suffixes_trailing_space[$i]}} -- ${descriptions_trailing_space[$i]}"
            fi
        done

        for ((i = 1; i <= $#suffixes_no_trailing_space; i++)); do
            if [[ -z ${descriptions_no_trailing_space[$i]} ]]; then
                descriptions_no_trailing_space[$i]="${(r($maxlen)( ))${suffixes_no_trailing_space[$i]}}"
            else
                descriptions_no_trailing_space[$i]="${(r($maxlen)( ))${suffixes_no_trailing_space[$i]}} -- ${descriptions_no_trailing_space[$i]}"
            fi
        done

        compadd -O m -a completions_no_description_trailing_space; matches+=("${m[@]}")
        compadd -O m -a completions_no_description_no_trailing_space; matches+=("${m[@]}")
        compadd -O m -a completions_trailing_space; matches+=("${m[@]}")
        compadd -O m -a completions_no_trailing_space; matches+=("${m[@]}")

        if [[ ${#matches} -gt 0 ]]; then
            compadd -Q -a completions_no_description_trailing_space
            compadd -Q -S ' ' -a completions_no_description_no_trailing_space
            compadd -l -Q -a -d descriptions_trailing_space completions_trailing_space
            compadd -l -Q -S '' -a -d descriptions_no_trailing_space completions_no_trailing_space
            return 0
        fi
    done
}

if [[ $ZSH_EVAL_CONTEXT =~ :file$ ]]; then
    compdef _pokego pokego
else
    _pokego
fi
