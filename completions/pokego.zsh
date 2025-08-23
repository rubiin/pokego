#compdef pokego

_pokego () {
    declare -a literals=("--help" "-h" "--form" "string" "-f" "string" "--list" "-l" "--name" "-n" "--no-title" "-nt" "--random" "1" "2" "3" "4" "5" "6" "7" "8" "-r" "1" "2" "3" "4" "5" "6" "7" "8" "--shiny" "-s" "--version" "-v")
    declare -A descrs=()
    descrs[0]="display this help text and exit"
    descrs[1]="display Pokémon with alternate forms"
    descrs[2]="Print list of all pokemon"
    descrs[3]="Select pokemon by name"
    descrs[4]="Do not display pokemon name"
    descrs[5]="Show a random pokemon. This flag can optionally be followed by a generation number or range"
    descrs[6]="Show the shiny version of the pokemon instead"
    descrs[7]="Show the cli version"
    declare -A descr_id_from_literal_id=([1]=0 [3]=1 [7]=2 [9]=3 [11]=4 [13]=5 [31]=6 [33]=7)
    declare -a regexes=()
    declare -A literal_transitions=()
    literal_transitions[1]="([1]=2 [2]=2 [3]=3 [5]=4 [7]=2 [8]=2 [9]=3 [10]=4 [11]=2 [12]=2 [13]=5 [22]=6 [31]=2 [32]=2 [33]=2 [34]=2)"
    literal_transitions[3]="([6]=2)"
    literal_transitions[4]="([6]=2)"
    literal_transitions[5]="([23]=2 [24]=2 [25]=2 [26]=2 [27]=2 [28]=2 [29]=2 [30]=2)"
    literal_transitions[6]="([23]=2 [24]=2 [25]=2 [26]=2 [27]=2 [28]=2 [29]=2 [30]=2)"
    declare -A nontail_transitions=()
    declare -A match_anything_transitions=()
    declare -A subword_transitions=()

    declare state=1
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


        if [[ -v "match_anything_transitions[$state]" ]]; then
            state=${match_anything_transitions[$state]}
            word_index=$((word_index + 1))
            continue
        fi

        return 1
    done

    declare -A literal_transitions_level_0=([1]="1 3 7 9 11 13 31 33" [5]="23 24 25 26 27 28 29 30" [3]="6")
    declare -A literal_transitions_level_1=([1]="2 5 8 10 12 22 32 34" [4]="6" [6]="23 24 25 26 27 28 29 30")
    declare -A subword_transitions_level_0=()
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
