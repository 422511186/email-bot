#!/bin/bash

# Ensure build directory exists
mkdir -p build

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o build/email-bot.exe main.go

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o build/email-bot-linux main.go

echo "Building for macOS (Intel amd64)..."
GOOS=darwin GOARCH=amd64 go build -o build/email-bot-mac-intel main.go

echo "Building for macOS (Apple Silicon arm64)..."
GOOS=darwin GOARCH=arm64 go build -o build/email-bot-mac-m1 main.go

echo "Build complete! Check the 'build' directory."
