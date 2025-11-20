.PHONY: help build clean install test run

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=harific
BUILD_DIR=./bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Show this help message
	@echo 'Harific - Comprehensive HAR File Toolkit'
	@echo ''
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the harific binary
	@echo "Building harific..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "Usage examples:"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME) test.har                    # View HAR file"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME) generate -n 100 -o out.har  # Generate HAR file"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME) version                      # Show version"

clean: ## Remove built binaries
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "✓ Cleaned"

install: build ## Install harific to $GOPATH/bin
	@echo "Installing harific..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-hargen: ## Run hargen package tests
	@echo "Running hargen package tests..."
	$(GOTEST) -v ./hargen/...

test-motor: ## Run motor package tests
	@echo "Running motor package tests..."
	$(GOTEST) -v ./motor/...

test-tui: ## Run tui package tests
	@echo "Running tui package tests..."
	$(GOTEST) -v ./tui/...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✓ Dependencies ready"

run: build ## Build and run harific
	@$(BUILD_DIR)/$(BINARY_NAME)

run-generate: build ## Build and run harific generate with example
	@echo "Running example generation..."
	$(BUILD_DIR)/$(BINARY_NAME) generate -n 10 -i apple,banana,cherry -l url,request.body -o example.har
	@echo "✓ Generated example.har"

all: deps build test ## Download deps, build, and test
