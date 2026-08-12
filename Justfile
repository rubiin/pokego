# prints all available commands
default:
	just --list

# compile a stripped release binary to ./pokego (~26 MB vs ~28 MB)
build:
	go build -ldflags "-s -w" -o pokego .

# clean all auto generated files and generate build
init: clean-files generate-completion release

# clean all auto generated files
clean-files:
	rm -rf build dist

generate-completion:
	complgen --bash ./completions/pokego.bash --fish ./completions/pokego.fish --zsh ./completions/pokego.zsh ./completions/pokego.usage

release:
	goreleaser release
