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

    set literals --help -h --form string -f string --list -l --name -n --no-title -nt --random 1 2 3 4 5 6 7 8 -r 1 2 3 4 5 6 7 8 --shiny -s --version -v

    set descrs
    set descrs[1] "Display this help text and exit"
    set descrs[2] "Show an alternate form of a pokemon"
    set descrs[3] "Print list of all pokemon"
    set descrs[4] "Select pokemon by name"
    set descrs[5] "Do not display pokemon name"
    set descrs[6] "Show a random pokemon. This flag can optionally be followed by a generation number or range"
    set descrs[7] "Show the shiny version of the pokemon instead"
    set descrs[8] "Show the cli version"
    set descr_literal_ids 1 3 7 9 11 13 31 33
    set descr_ids 1 2 3 4 5 6 7 8
    set regexes 
    set literal_transitions_inputs
    set nontail_transitions
    set literal_transitions_inputs[1] "1 2 3 5 7 8 9 10 11 12 13 22 31 32 33 34"
    set literal_transitions_tos[1] "2 2 3 4 2 2 3 4 2 2 5 6 2 2 2 2"
    set literal_transitions_inputs[3] 6
    set literal_transitions_tos[3] 2
    set literal_transitions_inputs[4] 6
    set literal_transitions_tos[4] 2
    set literal_transitions_inputs[5] "23 24 25 26 27 28 29 30"
    set literal_transitions_tos[5] "2 2 2 2 2 2 2 2"
    set literal_transitions_inputs[6] "23 24 25 26 27 28 29 30"
    set literal_transitions_tos[6] "2 2 2 2 2 2 2 2"

    set match_anything_transitions_from 
    set match_anything_transitions_to 

    set state 1
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

        if set --query match_anything_transitions_from[$state] && test -n $match_anything_transitions_from[$state]
            set index (contains --index -- "$state" $match_anything_transitions_from)
            set state $match_anything_transitions_to[$index]
            set word_index (math $word_index + 1)
            continue
        end

        return 1
    end

    set literal_froms_level_0 1 5 3
    set literal_inputs_level_0 "1 3 7 9 11 13 31 33|23 24 25 26 27 28 29 30|6"
    set literal_froms_level_1 1 4 6
    set literal_inputs_level_1 "2 5 8 10 12 22 32 34|6|23 24 25 26 27 28 29 30"
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
        printf '%s\n' $candidates | __complgen_match $COMP_WORDS[$word_index] && return 0
    end
end

complete --erase pokego
complete --command pokego --no-files --arguments "(_pokego)"
