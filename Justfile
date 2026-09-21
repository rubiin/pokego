# prints all available commands
default:
	just --list

# compile a stripped release binary to ./pokego (~26 MB vs ~28 MB)
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

# Build the pokego binary
build:
    @echo "Building pokego version: {{version}}"
    CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version={{version}}" -o pokego .

# clean all auto generated files and generate build
init: clean-files release

# clean all auto generated files
clean-files:
	rm -rf build dist

# cut a release
release:
	goreleaser release --clean

test:
	go test ./...

# Run golangci-lint
lint:
	golangci-lint run

# Run golangci-lint with auto-fix
lint-fix:
	golangci-lint run --fix

fmt:
	golangci-lint fmt
