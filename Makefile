.PHONY: help hargen clean install test

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=hargen
BUILD_DIR=./bin
CMD_DIR=./cmd/hargen

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

hargen: ## Build the hargen binary
	@echo "Building hargen..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

build: hargen ## Alias for 'hargen' target

clean: ## Remove built binaries
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "✓ Cleaned"

install: hargen ## Install hargen to $GOPATH/bin
	@echo "Installing hargen..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-hargen: ## Run hargen package tests only
	@echo "Running hargen tests..."
	$(GOTEST) -v ./hargen/...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✓ Dependencies ready"

run-example: hargen ## Build and run hargen with example parameters
	@echo "Running example generation..."
	$(BUILD_DIR)/$(BINARY_NAME) -n 10 -i apple,banana,cherry -l url,request.body -o example.har
	@echo "✓ Generated example.har"

all: deps hargen test ## Download deps, build, and test
