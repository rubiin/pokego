set dotenv-load

# prints all available commands
default:
	just --list

# clean all auto generated files and generate build
init: clean-files build

# clean all auto generated files
clean-files:
	rm -rf builds


build:
	echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego.exe main.go
	upx pokego.exe

	echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego main.go
	tar -czvf pokego-mac-$VERSION.tar.gz pokego LICENSE

	echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego main.go
	upx pokego
	tar -czvf pokego-linux-$VERSION.tar.gz pokego LICENSE
