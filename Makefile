# Makefile for labelarr

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=labelarr
BINARY_PATH=./cmd/labelarr

# Build the application
.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -cover ./...

# Run tests with coverage report
.PHONY: test-coverage-html
test-coverage-html:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run linter (requires golangci-lint to be installed)
.PHONY: lint
lint:
	golangci-lint run

# Run the application
.PHONY: run
run: build
	./$(BINARY_NAME)

# Build for multiple platforms
.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 $(BINARY_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-amd64.exe $(BINARY_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin-amd64 $(BINARY_PATH)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  test            - Run all tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  benchmark       - Run benchmark tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  lint            - Run linter"
	@echo "  run             - Build and run the application"
	@echo "  build-all       - Build for multiple platforms"
	@echo "  help            - Show this help message" 