.PHONY: all build build-cairo test clean help

# Binary name
BINARY_NAME=rmc

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Main package path
MAIN_PACKAGE=./cmd/rmc-go

# Test files
TEST_DIR=tests
TEST_FILES=$(wildcard $(TEST_DIR)/*.rm)
TEST_OUTPUT_DIR=test_output

# Default target
all: build

# Build the binary (without Cairo support)
build:
	@echo "Building $(BINARY_NAME) (without Cairo support)..."
	@echo "For native PDF export use: make build-cairo"
	CGO_ENABLED=0 $(GOBUILD) -tags '!cairo' -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✓ Build complete: $(BINARY_NAME)"

# Build the binary with Cairo support (requires CGo and Cairo libraries)
build-cairo:
	@echo "Building $(BINARY_NAME) with Cairo support..."
	@echo "This requires cairo development libraries installed:"
	@echo "  macOS: brew install cairo pkg-config"
	@echo "  Ubuntu/Debian: sudo apt-get install libcairo2-dev"
	@echo "  Fedora: sudo dnf install cairo-devel"
	@echo ""
	CGO_ENABLED=1 $(GOBUILD) -tags cairo -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✓ Build complete: $(BINARY_NAME) (with Cairo support)"

# Run tests with test files
test: build
	@echo "Running tests with .rm files..."
	@mkdir -p $(TEST_OUTPUT_DIR)
	@echo ""
	@echo "Testing SVG export:"
	@echo "==================="
	@for file in $(TEST_FILES); do \
		basename=$$(basename $$file .rm); \
		echo "  Testing $$basename.rm → $$basename.svg"; \
		./$(BINARY_NAME) $$file -o $(TEST_OUTPUT_DIR)/$$basename.svg; \
		if [ -f $(TEST_OUTPUT_DIR)/$$basename.svg ]; then \
			size=$$(ls -lh $(TEST_OUTPUT_DIR)/$$basename.svg | awk '{print $$5}'); \
			echo "    ✓ Generated ($$size)"; \
		else \
			echo "    ✗ Failed"; \
			exit 1; \
		fi; \
	done
	@echo ""
	@echo "Testing PDF export:"
	@echo "==================="
	@for file in $(TEST_FILES); do \
		basename=$$(basename $$file .rm); \
		echo "  Testing $$basename.rm → $$basename.pdf"; \
		./$(BINARY_NAME) $$file -o $(TEST_OUTPUT_DIR)/$$basename.pdf 2>&1; \
		if [ -f $(TEST_OUTPUT_DIR)/$$basename.pdf ]; then \
			size=$$(ls -lh $(TEST_OUTPUT_DIR)/$$basename.pdf | awk '{print $$5}'); \
			echo "    ✓ Generated ($$size)"; \
		else \
			echo "    ⚠ Failed (Inkscape may not be installed)"; \
		fi; \
	done
	@echo ""
	@echo "Test outputs saved to: $(TEST_OUTPUT_DIR)/"
	@echo "✓ All tests completed"

# Run Go unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v ./...

# Clean build artifacts and test outputs
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(TEST_OUTPUT_DIR)
	@echo "✓ Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) -v ./...
	@echo "✓ Dependencies installed"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build        - Build the $(BINARY_NAME) binary (without Cairo)"
	@echo "  make build-cairo  - Build the $(BINARY_NAME) binary with Cairo support"
	@echo "  make test         - Run integration tests with .rm files"
	@echo "  make test-unit    - Run Go unit tests"
	@echo "  make clean        - Remove binary and test outputs"
	@echo "  make deps         - Install Go dependencies"
	@echo "  make all          - Build the binary (default)"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Cairo support:"
	@echo "  Build with 'make build-cairo' to enable native PDF export"
	@echo "  Requires: cairo development libraries and pkg-config"
	@echo ""
	@echo "Test files in $(TEST_DIR)/:"
	@for file in $(TEST_FILES); do \
		echo "  - $$(basename $$file)"; \
	done
