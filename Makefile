.PHONY: all build clean

all: build

build:
	@echo "Building binaries..."
	@bash build.sh

clean:
	@echo "Cleaning up..."
	@rm -rf build/
