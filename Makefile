BINARY_DIR := dist
BINARY_NAME := auto-i18n
GO := go
LDFLAGS := -ldflags="-s -w"

.PHONY: all build build-all clean test help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o $(BINARY_DIR)/$(BINARY_NAME)$(shell go env GOEXE) .

build-all: build-windows build-linux build-linux-arm64 build-macos build-macos-arm64

build-windows: GOOS=windows
build-windows: GOARCH=amd64
build-windows:
	@echo "Building $(BINARY_NAME) for Windows (amd64)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME).exe .

build-linux: GOOS=linux
build-linux: GOARCH=amd64
build-linux:
	@echo "Building $(BINARY_NAME) for Linux (amd64)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux .

build-linux-arm64: GOOS=linux
build-linux-arm64: GOARCH=arm64
build-linux-arm64:
	@echo "Building $(BINARY_NAME) for Linux (arm64)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 .

build-macos: GOOS=darwin
build-macos: GOARCH=amd64
build-macos:
	@echo "Building $(BINARY_NAME) for macOS (amd64)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-macos .

build-macos-arm64: GOOS=darwin
build-macos-arm64: GOARCH=arm64
build-macos-arm64:
	@echo "Building $(BINARY_NAME) for macOS (arm64)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-macos-arm64 .

clean:
	@echo "Cleaning $(BINARY_DIR)..."
	$(GO) clean
ifeq ($(OS),Windows_NT)
	if exist $(BINARY_DIR) rmdir /s /q $(BINARY_DIR)
else
	rm -rf $(BINARY_DIR)
endif

test:
	$(GO) test ./... -v

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all             Build for current platform (default)"
	@echo "  build           Same as 'all'"
	@echo "  build-all       Build for all platforms"
	@echo "  build-windows   Build for Windows (amd64)"
	@echo "  build-linux     Build for Linux (amd64)"
	@echo "  build-linux-arm64  Build for Linux (arm64)"
	@echo "  build-macos     Build for macOS (amd64)"
	@echo "  build-macos-arm64  Build for macOS (arm64)"
	@echo "  clean           Remove all built binaries"
	@echo "  test            Run all tests"
