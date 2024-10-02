set dotenv-required

# prints all available commands
default:
	just --list

# clean all auto generated files and generate build
init: clean-files create-dirs build

# clean all auto generated files
clean-files:
	rm -rf builds

create-dirs:
	mkdir -p builds/windows
	mkdir -p builds/macos
	mkdir -p builds/linux

build:
	echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o builds/windows/pokego.exe main.go
	upx builds/windows/pokego.exe

	echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o builds/macos/pokego main.go

	echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o builds/linux/pokego main.go
	upx builds/linux/pokego

