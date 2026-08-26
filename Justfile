# prints all available commands
default:
	just --list

# compile a stripped release binary to ./pokego (~26 MB vs ~28 MB)
build:
	go build -ldflags "-s -w" -o pokego .

# clean all auto generated files and generate build
init: clean-files release

# clean all auto generated files
clean-files:
	rm -rf build dist

# cut a release
release:
	goreleaser release --clean

test:
	go test
