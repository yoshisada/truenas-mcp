BINARY_NAME=truenas-mcp
BUILD_DIR=.

# Release version: taken from the git tag (v stripped), e.g. v0.0.5 -> 0.0.5.
# Override with: make build VERSION=1.2.3
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

# Default transport: http for container builds, override with TRANSPORT=stdio
TRANSPORT ?= http

# Build for local platform
build:
	@echo "Building $(BINARY_NAME) $(VERSION) for local platform..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/truenas-mcp

# Build for all platforms
build-all:
	@echo "Building $(VERSION) for all platforms..."
	@echo "Building for macOS (ARM64)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/truenas-mcp
	@echo "Building for macOS (AMD64)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/truenas-mcp
	@echo "Building for Linux (AMD64)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/truenas-mcp
	@echo "Building for Windows (AMD64)..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/truenas-mcp
	@echo "All builds complete!"

# Build Docker image
docker:
	@echo "Building Docker image $(BINARY_NAME):$(VERSION)..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "Tagged as $(BINARY_NAME):$(VERSION)"

# Run locally in HTTP mode (for testing)
run-http:
	@echo "Running $(BINARY_NAME) in HTTP mode on :8080..."
	go run $(LDFLAGS) ./cmd/truenas-mcp --transport=http --http-addr=:8080

# Run locally in stdio mode (for testing)
run-stdio:
	@echo "Running $(BINARY_NAME) in stdio mode..."
	go run $(LDFLAGS) ./cmd/truenas-mcp --transport=stdio

clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f $(BUILD_DIR)/$(BINARY_NAME)-*

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linters..."
	go vet ./...
	go fmt ./...
