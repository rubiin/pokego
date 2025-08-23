# prints all available commands
default:
	just --list

# clean all auto generated files and generate build
init: clean-files generate-completion release

# clean all auto generated files
clean-files:
	rm -rf build dist

generate-completion:
	complgen --bash ./completions/pokego.bash --fish ./completions/pokego.fish --zsh ./completions/pokego.zsh ./completions/pokego.usage

release:
	goreleaser release
