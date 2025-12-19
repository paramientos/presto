.PHONY: build install test clean

# Build configuration
BINARY_NAME=presto
BUILD_DIR=bin
INSTALL_PATH=/usr/local/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the binary
build:
	@echo "🎵 Building Presto..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/presto
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	$(GOGET) github.com/spf13/cobra@latest
	$(GOGET) github.com/Masterminds/semver/v3@latest
	$(GOGET) github.com/schollz/progressbar/v3@latest
	$(GOMOD) tidy
	@echo "✅ Dependencies installed"

# Install the binary
install: build
	@echo "📥 Installing Presto to $(INSTALL_PATH)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Presto installed successfully!"
	@echo "Run 'presto --version' to verify"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf vendor
	@echo "✅ Clean complete"

# Build for all platforms
build-all:
	@echo "🎵 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/presto
	
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/presto
	
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/presto
	
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/presto
	
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/presto
	
	@echo "✅ All builds complete!"

# Run the binary
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development mode - build and run
dev:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/presto && ./$(BUILD_DIR)/$(BINARY_NAME)

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@golangci-lint run
	@echo "✅ Lint complete"

# Show help
help:
	@echo "Presto Makefile Commands:"
	@echo "  make build      - Build the binary"
	@echo "  make deps       - Install dependencies"
	@echo "  make install    - Install Presto to system"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make run        - Build and run"
	@echo "  make dev        - Development mode"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Lint code"
