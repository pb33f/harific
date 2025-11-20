.PHONY: help build clean install

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=harific
BUILD_DIR=./bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

help: ## Show this help message
	@echo 'HARific - High-Performance HAR File Toolkit'
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