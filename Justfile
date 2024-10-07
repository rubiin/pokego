set dotenv-load
FILENAME := "PKGBUILD"

# prints all available commands
default:
	just --list

# clean all auto generated files and generate build
init: clean-files build

# clean all auto generated files
clean-files:
	rm -rf builds

generate-completion:
	complgen aot --bash-script ./completions/pokego.bash --fish-script ./completions/pokego.fish --zsh-script ./completions/pokego.zsh ./completions/pokego.usage


build:
	echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego main.go
	upx --best --lzma pokego
	tar -czvf pokego-linux-$VERSION.tar.gz pokego LICENSE completions

	echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego.exe main.go
	upx --best --lzma pokego.exea

	echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X 'main.version=$VERSION'" -o pokego main.go
	tar -czvf pokego-mac-$VERSION.tar.gz pokego LICENSE completions

pkgbuild:
	sed -i "s/pkgver=.*/pkgver=$VERSION/" PKGBUILD
	sed -i "s/sha256sums=\"[^\"]*\"/sha256sums=\"$$FILENAME\"/" PKGBUILD
