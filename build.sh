#!/bin/bash

APP_NAME="pokego"

# Create output directories
mkdir -p builds/windows
mkdir -p builds/macos
mkdir -p builds/linux

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -o builds/windows/${APP_NAME}.exe main.go

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -o builds/macos/${APP_NAME} main.go

# Build for Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o builds/linux/${APP_NAME} main.go

# Package the binaries
echo "Packaging binaries..."
cd builds

# Create ZIP files for each platform
zip -r ${APP_NAME}_windows.zip windows/${APP_NAME}.exe
zip -r ${APP_NAME}_macos.zip macos/${APP_NAME}
zip -r ${APP_NAME}_linux.zip linux/${APP_NAME}

# Clean up build directories
cd ..
rm -rf builds/windows builds/macos builds/linux

echo "Build and packaging completed!"
