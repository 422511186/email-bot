BINARY   := email-bot
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-s -w -X main.Version=$(VERSION)"

# ── Default: build for current OS/arch ─────────────────────────
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY) .

# ── Build and Install (System wide) ──────────────────────────────
.PHONY: install
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installation complete."

# ── Upgrade Helper (Fetch, Build, Install) ───────────────────────
.PHONY: upgrade
upgrade:
	@echo "Fetching latest changes..."
	git pull
	@echo "Downloading dependencies..."
	$(MAKE) deps
	@echo "Building new binary..."
	$(MAKE) build
	@echo "Replacing existing installation..."
	@sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Upgrade complete! Please restart your service."

# ── Run (requires config.yaml in current dir) ───────────────────
.PHONY: run
run:
	go run . -config config.yaml

# ── Tests ────────────────────────────────────────────────────────
.PHONY: test
test:
	go test ./...

# ── Lint ─────────────────────────────────────────────────────────
.PHONY: lint
lint:
	go vet ./...

# ── Cross-platform release builds ───────────────────────────────
.PHONY: build-all
build-all: \
	build-windows-amd64 \
	build-linux-amd64 \
	build-linux-arm64 \
	build-darwin-amd64 \
	build-darwin-arm64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) \
		-o dist/$(BINARY)-windows-amd64.exe .

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) \
		-o dist/$(BINARY)-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) \
		-o dist/$(BINARY)-linux-arm64 .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) \
		-o dist/$(BINARY)-darwin-amd64 .

# macOS M-series (Apple Silicon)
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) \
		-o dist/$(BINARY)-darwin-arm64 .

# ── Download deps ────────────────────────────────────────────────
.PHONY: deps
deps:
	go mod tidy
	go mod download

# ── Clean ────────────────────────────────────────────────────────
.PHONY: clean
clean:
	rm -rf dist/ $(BINARY) $(BINARY).exe

# ── Setup config from example ────────────────────────────────────
.PHONY: init-config
init-config:
	@if [ ! -f config.yaml ]; then \
		cp config.yaml.example config.yaml; \
		echo "Created config.yaml — please edit it before running."; \
	else \
		echo "config.yaml already exists, skipping."; \
	fi
