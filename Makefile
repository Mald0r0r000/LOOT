APP_NAME := loot
BUILD_DIR := bin
SRC := ./cmd/loot
GO := go

.PHONY: all build run test clean lint deps

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(SRC)

run: build
	@echo "Running $(APP_NAME)..."
	@$(BUILD_DIR)/$(APP_NAME)

test:
	@echo "Running tests..."
	@$(GO) test ./... -v

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

lint:
	@echo "Running go vet..."
	@$(GO) vet ./...

deps:
	@echo "Downloading dependencies..."
	@$(GO) mod tidy
	@$(GO) mod download
