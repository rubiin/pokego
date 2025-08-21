function _pokego_subword
    set mode $argv[1]
    set word $argv[2]

    set char_index 1
    set matched 0
    while true
        if test $char_index -gt (string length -- "$word")
            set matched 1
            break
        end

        set subword (string sub --start=$char_index -- "$word")

        if set --query subword_literal_transitions_inputs[$subword_state] && test -n $subword_literal_transitions_inputs[$subword_state]
            set inputs (string split ' ' $subword_literal_transitions_inputs[$subword_state])
            set tos (string split ' ' $subword_literal_transitions_tos[$subword_state])

            set literal_matched 0
            for literal_id in (seq 1 (count $subword_literals))
                set literal $subword_literals[$literal_id]
                set literal_len (string length -- "$literal")
                set subword_slice (string sub --end=$literal_len -- "$subword")
                if test $subword_slice = $literal
                    set index (contains --index -- "$literal_id" $inputs)
                    set subword_state $tos[$index]
                    set char_index (math $char_index + $literal_len)
                    set literal_matched 1
                    break
                end
            end
            if test $literal_matched -ne 0
                continue
            end
        end

        if set --query subword_nontail_regexes[$subword_state] && test -n $subword_nontail_regexes[$subword_state]
            set regex_ids (string split ' ' $subword_nontail_regexes[$subword_state])
            set tos (string split ' ' $subword_nontail_tos[$subword_state])

            set nontail_matched 0
            for regex_id in $regex_ids
                set regex $subword_regexes[$regex_id]
                string match --regex --quiet "^(?<match>$regex).*" -- $subword
                if test -n "$match"
                    set subword_state $tos[$regex_id]
                    set match_len (string length -- $match)
                    set char_index (math $char_index + $match_len)
                    set nontail_matched 1
                    break
                end
            end
            if test $nontail_matched -ne 0
                continue
            end
        end

        set index (contains --index -- "$subword_state" $subword_match_anything_transitions_from)
        if test -n "$index"
            set subword_state $subword_match_anything_transitions_to[$index]
            set matched 1
            break
        end

        break
    end

    if test $mode = matches
        return (math 1 - $matched)
    end

    set unmatched_suffix (string sub --start=$char_index -- $word)

    set matched_prefix
    if test $char_index -eq 1
        set matched_prefix ""
    else
        set matched_prefix (string sub --end=(math $char_index - 1) -- "$word")
    end

    for fallback_level in (seq 0 $subword_max_fallback_level)
        set candidates
        set froms_name subword_literal_transitions_from_level_$fallback_level
        set froms (string split ' ' $$froms_name)
        if contains $subword_state $froms
            set index (contains --index -- "$subword_state" $froms)
            set transitions_name subword_literal_transitions_level_$fallback_level
            printf 'set transitions (string split \' \' $%s[%d])' $transitions_name $index | source
            for literal_id in $transitions
                set unmatched_suffix_len (string length -- $unmatched_suffix)
                if test $unmatched_suffix_len -gt 0
                    set literal $subword_literals[$literal_id]
                    set slice (string sub --end=$unmatched_suffix_len -- $literal)
                    if test "$slice" != "$unmatched_suffix"
                        continue
                    end
                end
                set subword_descr_index (contains --index -- "$literal_id" $subword_descr_literal_ids)
                if test -n "$subword_descr_index"
                    set --append candidates (printf '%s%s\t%s\n' $matched_prefix $subword_literals[$literal_id] $subword_descrs[$subword_descr_ids[$subword_descr_index]])
                else
                    set --append candidates (printf '%s%s\n' $matched_prefix $subword_literals[$literal_id])
                end
            end
        end

        set name subword_nontail_command_froms_level_$fallback_level
        set commands $$name
        set index (contains --index -- "$subword_state" $commands)
        if test -n "$index"
            set name subword_nontail_commands_level_$fallback_level
            set commands (string split ' ' $$name)
            set function_id $commands[$index]
            set function_name _pokego_cmd_$function_id
            set name subword_nontail_regexes_level_$fallback_level
            set rxs (string split ' ' $$name)
            set rx $subword_regexes[$rxs[$index]]
            for line in ($function_name "$COMP_WORDS[$COMP_CWORD]")
                string match --regex --quiet "^(?<match>$rx).*" -- $line
                if test -n "$match"
                    set --append candidates (printf "%s%s\n" $matched_prefix $match)
                end
            end
        end

        set froms_name subword_commands_from_level_$fallback_level
        set froms (string split ' ' $$froms_name)
        set index (contains --index -- "$subword_state" $froms)
        if test -n "$index"
            printf 'set function_id $subword_commands_level_%s[%d]' $fallback_level $index | source
            set function_name _pokego_cmd_$function_id
            $function_name "$matched_prefix" | while read line
                set --append candidates (printf '%s%s\n' $matched_prefix $line)
            end
        end

        printf '%s\n' $candidates | __complgen_match && break
    end
end

function _pokego_subword_1
    set mode $argv[1]
    set word $argv[2]

    set --global subword_literals --name=

    set --global subword_descrs
    set subword_descrs[1] "show information for the given Pokémon NAME"
    set --global subword_descr_literal_ids 1
    set --global subword_descr_ids 1
    set --global subword_regexes 
    set --global subword_literal_transitions_inputs
    set --global subword_literal_transitions_inputs[1] 1
    set --global subword_literal_transitions_tos[1] 2

    set --global subword_match_anything_transitions_from 2
    set --global subword_match_anything_transitions_to 3

    set --global subword_literal_transitions_from_level_0 1
    set --global subword_literal_transitions_level_0 1
    set --global subword_nontail_command_froms_level_0 
    set --global subword_nontail_commands_level_0 
    set --global subword_nontail_regexes_level_0 
    set --global subword_commands_from_level_0 
    set --global subword_commands_level_0 
    set --global subword_max_fallback_level 0

    set --global subword_state 1
    _pokego_subword "$mode" "$word"
end


function __complgen_match
    set prefix $argv[1]

    set candidates
    set descriptions
    while read c
        set a (string split --max 1 -- "	" $c)
        set --append candidates $a[1]
        if set --query a[2]
            set --append descriptions $a[2]
        else
            set --append descriptions ""
        end
    end

    if test -z "$candidates"
        return 1
    end

    set escaped_prefix (string escape --style=regex -- $prefix)
    set regex "^$escaped_prefix.*"

    set matches_case_sensitive
    set descriptions_case_sensitive
    for i in (seq 1 (count $candidates))
        if string match --regex --quiet --entire -- $regex $candidates[$i]
            set --append matches_case_sensitive $candidates[$i]
            set --append descriptions_case_sensitive $descriptions[$i]
        end
    end

    if set --query matches_case_sensitive[1]
        for i in (seq 1 (count $matches_case_sensitive))
            printf '%s	%s\n' $matches_case_sensitive[$i] $descriptions_case_sensitive[$i]
        end
        return 0
    end

    set matches_case_insensitive
    set descriptions_case_insensitive
    for i in (seq 1 (count $candidates))
        if string match --regex --quiet --ignore-case --entire -- $regex $candidates[$i]
            set --append matches_case_insensitive $candidates[$i]
            set --append descriptions_case_insensitive $descriptions[$i]
        end
    end

    if set --query matches_case_insensitive[1]
        for i in (seq 1 (count $matches_case_insensitive))
            printf '%s	%s\n' $matches_case_insensitive[$i] $descriptions_case_insensitive[$i]
        end
        return 0
    end

    return 1
end


function _pokego
    set COMP_LINE (commandline --cut-at-cursor)

    set COMP_WORDS
    echo $COMP_LINE | read --tokenize --array COMP_WORDS
    if string match --quiet --regex '.*\s$' $COMP_LINE
        set COMP_CWORD (math (count $COMP_WORDS) + 1)
    else
        set COMP_CWORD (count $COMP_WORDS)
    end

    set literals --help -h --form --list --no-title --random --shiny --version -V

    set descrs
    set descrs[1] "display this help text and exit"
    set descrs[2] "display Pokémon with alternate forms"
    set descrs[3] "list all available Pokémon"
    set descrs[4] "suppress title/header in output"
    set descrs[5] "display a random Pokémon"
    set descrs[6] "display the shiny version if available"
    set descrs[7] "display version information and exit"
    set descr_literal_ids 1 3 4 5 6 7 8
    set descr_ids 1 2 3 4 5 6 7
    set regexes 
    set literal_transitions_inputs
    set nontail_transitions
    set literal_transitions_inputs[2] "1 2 3 4 5 6 7 8 9"
    set literal_transitions_tos[2] "3 3 3 3 3 3 3 3 3"

    set match_anything_transitions_from 
    set match_anything_transitions_to 
    set subword_transitions_ids[2] 1
    set subword_transitions_tos[2] 3

    set state 2
    set word_index 2
    while test $word_index -lt $COMP_CWORD
        set -- word $COMP_WORDS[$word_index]

        if set --query literal_transitions_inputs[$state] && test -n $literal_transitions_inputs[$state]
            set inputs (string split ' ' $literal_transitions_inputs[$state])
            set tos (string split ' ' $literal_transitions_tos[$state])

            set literal_id (contains --index -- "$word" $literals)
            if test -n "$literal_id"
                set index (contains --index -- "$literal_id" $inputs)
                set state $tos[$index]
                set word_index (math $word_index + 1)
                continue
            end
        end

        if set --query subword_transitions_ids[$state] && test -n $subword_transitions_ids[$state]
            set subword_ids (string split ' ' $subword_transitions_ids[$state])
            set tos $subword_transitions_tos[$state]

            set subword_matched 0
            for subword_id in $subword_ids
                if _pokego_subword_$subword_id matches "$word"
                    set subword_matched 1
                    set state $tos[$subword_id]
                    set word_index (math $word_index + 1)
                    break
                end
            end
            if test $subword_matched -ne 0
                continue
            end
        end

        if set --query match_anything_transitions_from[$state] && test -n $match_anything_transitions_from[$state]
            set index (contains --index -- "$state" $match_anything_transitions_from)
            set state $match_anything_transitions_to[$index]
            set word_index (math $word_index + 1)
            continue
        end

        return 1
    end

    set literal_froms_level_0 2
    set literal_inputs_level_0 "1 3 4 5 6 7 8"
    set literal_froms_level_1 2
    set literal_inputs_level_1 "2 9"
    set subword_froms_level_0 2
    set subwords_level_0 "1"
    set subword_froms_level_1 
    set subwords_level_1 
    set nontail_command_froms_level_0 
    set nontail_commands_level_0 
    set nontail_regexes_level_0 
    set nontail_command_froms_level_1 
    set nontail_commands_level_1 
    set nontail_regexes_level_1 

    for fallback_level in (seq 0 1)
        set candidates
        set froms_name literal_froms_level_$fallback_level
        set froms $$froms_name
        set index (contains --index -- "$state" $froms)
        if test -n "$index"
            set level_inputs_name literal_inputs_level_$fallback_level
            set input_assoc_values (string split '|' $$level_inputs_name)
            set state_inputs (string split ' ' $input_assoc_values[$index])
            for literal_id in $state_inputs
                set descr_index (contains --index -- "$literal_id" $descr_literal_ids)
                if test -n "$descr_index"
                    set --append candidates (printf '%s\t%s\n' $literals[$literal_id] $descrs[$descr_ids[$descr_index]])
                else
                    set --append candidates (printf '%s\n' $literals[$literal_id])
                end
            end
        end

        set subwords_name subword_froms_level_$fallback_level
        set subwords $$subwords_name
        set index (contains --index -- "$state" $subwords)
        if test -n "$index"
            set subwords_name subwords_level_$fallback_level
            set subwords (string split ' ' $$subwords_name)
            for id in $subwords
                set function_name _pokego_subword_$id
                set --append candidates ($function_name complete "$COMP_WORDS[$COMP_CWORD]")
            end
        end
        printf '%s\n' $candidates | __complgen_match $COMP_WORDS[$word_index] && return 0
    end
end

complete --erase pokego
complete --command pokego --no-files --arguments "(_pokego)"
