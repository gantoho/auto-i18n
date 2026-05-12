BINARY_DIR := dist
BINARY_NAME := auto-i18n
GO := go
LDFLAGS := -ldflags="-s -w"

ifeq ($(OS),Windows_NT)
BINARY_EXT := .exe
MKDIR_CMD := if not exist $(BINARY_DIR) mkdir $(BINARY_DIR)
else
BINARY_EXT :=
MKDIR_CMD := mkdir -p $(BINARY_DIR)
endif

.PHONY: all build build-all clean test help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@$(MKDIR_CMD)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)$(BINARY_EXT) .

build-all: build-windows build-linux build-linux-arm64 build-macos build-macos-arm64

build-windows: export GOOS=windows
build-windows: export GOARCH=amd64
build-windows:
	@echo "Building $(BINARY_NAME) for Windows (amd64)..."
	@$(MKDIR_CMD)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME).exe .

build-linux: export GOOS=linux
build-linux: export GOARCH=amd64
build-linux:
	@echo "Building $(BINARY_NAME) for Linux (amd64)..."
	@$(MKDIR_CMD)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux .

build-linux-arm64: export GOOS=linux
build-linux-arm64: export GOARCH=arm64
build-linux-arm64:
	@echo "Building $(BINARY_NAME) for Linux (arm64)..."
	@$(MKDIR_CMD)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 .

build-macos: export GOOS=darwin
build-macos: export GOARCH=amd64
build-macos:
	@echo "Building $(BINARY_NAME) for macOS (amd64)..."
	@$(MKDIR_CMD)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-macos .

build-macos-arm64: export GOOS=darwin
build-macos-arm64: export GOARCH=arm64
build-macos-arm64:
	@echo "Building $(BINARY_NAME) for macOS (arm64)..."
	@$(MKDIR_CMD)
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
	@echo "  build-linux         Build for Linux (amd64, x86_64)"
	@echo "  build-linux-arm64   Build for Linux (arm64, aarch64)"
	@echo "  build-macos     Build for macOS (amd64)"
	@echo "  build-macos-arm64  Build for macOS (arm64)"
	@echo "  clean           Remove all built binaries"
	@echo "  test            Run all tests"
