.PHONY: all build-mac build-mac-universal clean whisper-cpp dev run install-deps test

WHISPER_VERSION := v1.5.4
BINARY_NAME := whisper-tray
APP_NAME := WhisperTray

# Version info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

all: whisper-cpp build

# Detect platform
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    BUILD_TARGET := build-linux
endif
ifeq ($(UNAME_S),Darwin)
    BUILD_TARGET := build-mac
endif

# Build for current platform (auto-detect)
build: whisper-cpp
ifeq ($(UNAME_S),Darwin)
	@$(MAKE) build-mac
else ifeq ($(UNAME_S),Linux)
	@$(MAKE) build-linux
else
	@$(MAKE) build-windows
endif

# Setup whisper.cpp
whisper-cpp:
	@echo "Setting up whisper.cpp..."
	@if [ ! -d "vendor/whisper.cpp" ]; then \
		mkdir -p vendor; \
		git clone --depth 1 --branch $(WHISPER_VERSION) https://github.com/ggerganov/whisper.cpp vendor/whisper.cpp; \
	fi
	@echo "Building whisper.cpp..."
	@cd vendor/whisper.cpp && make libwhisper.a
	@echo "✓ whisper.cpp ready"

# Build for macOS (current arch)
build-mac: whisper-cpp
	@echo "Building for macOS ($(shell uname -m))..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper -framework Accelerate -framework Foundation -framework Metal -framework MetalKit" \
	go build -mod=mod -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/whisper-tray
	@echo "✓ Binary built: bin/$(BINARY_NAME) ($(VERSION) @ $(COMMIT))"

# Build for Linux
build-linux: whisper-cpp
	@echo "Building for Linux..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper" \
	go build -mod=mod -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/whisper-tray
	@echo "✓ Binary built: bin/$(BINARY_NAME) ($(VERSION) @ $(COMMIT))"

# Build for Windows
build-windows: whisper-cpp
	@echo "Building for Windows..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper" \
	go build -mod=mod -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME).exe ./cmd/whisper-tray
	@echo "✓ Binary built: bin/$(BINARY_NAME).exe ($(VERSION) @ $(COMMIT))"

# Build macOS app bundle
build-mac-app: build-mac
	@echo "Creating macOS app bundle..."
	@mkdir -p "bin/$(APP_NAME).app/Contents/MacOS"
	@mkdir -p "bin/$(APP_NAME).app/Contents/Resources"
	@cp bin/$(BINARY_NAME) "bin/$(APP_NAME).app/Contents/MacOS/"
	@cp resources/Info.plist "bin/$(APP_NAME).app/Contents/"
	@if [ -f resources/icon.icns ]; then cp resources/icon.icns "bin/$(APP_NAME).app/Contents/Resources/"; fi
	@echo "✓ App bundle created: bin/$(APP_NAME).app"

# Build universal binary (Intel + Apple Silicon)
build-mac-universal: whisper-cpp
	@echo "Building universal macOS binary..."
	@mkdir -p bin
	CGO_ENABLED=1 GOARCH=arm64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-arm64 ./cmd/whisper-tray
	CGO_ENABLED=1 GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-amd64 ./cmd/whisper-tray
	lipo -create -output bin/$(BINARY_NAME) bin/$(BINARY_NAME)-arm64 bin/$(BINARY_NAME)-amd64
	@rm bin/$(BINARY_NAME)-arm64 bin/$(BINARY_NAME)-amd64
	@echo "✓ Universal binary created: bin/$(BINARY_NAME)"

# Quick dev build (no whisper.cpp rebuild)
dev:
	@echo "Building for development..."
	@if [ -f vendor/whisper.cpp/libwhisper.a ]; then \
		echo "Using existing whisper.cpp build..."; \
		CGO_ENABLED=1 \
		CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
		CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper -framework Accelerate -framework Foundation -framework Metal -framework MetalKit" \
		go build -mod=mod -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/whisper-tray; \
	else \
		echo "No whisper.cpp found, building without it..."; \
		CGO_ENABLED=1 go build -mod=mod -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/whisper-tray; \
	fi
	@echo "✓ Dev binary built: bin/$(BINARY_NAME) ($(VERSION) @ $(COMMIT))"

# Run the application
run: dev
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

# Install Go dependencies
install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

# Run tests (auto-detect platform and allow specific test via TEST variable)
# Usage: make test or make test TEST=./internal/audio
test: whisper-cpp
ifeq ($(UNAME_S),Darwin)
	@$(MAKE) test-osx TEST=$(TEST)
else ifeq ($(UNAME_S),Linux)
	@$(MAKE) test-linux TEST=$(TEST)
else
	@$(MAKE) test-windows TEST=$(TEST)
endif

# Run tests on macOS
# Usage: make test-osx or make test-osx TEST=./internal/audio
test-osx: whisper-cpp
	@echo "Running tests (macOS)..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper -framework Accelerate -framework Foundation -framework Metal -framework MetalKit" \
	go test -mod=mod -v $(if $(TEST),$(TEST),./...)

# Run tests on Linux
# Usage: make test-linux or make test-linux TEST=./internal/audio
test-linux: whisper-cpp
	@echo "Running tests (Linux)..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper" \
	go test -mod=mod -v $(if $(TEST),$(TEST),./...)

# Run tests on Windows
# Usage: make test-windows or make test-windows TEST=./internal/audio
test-windows: whisper-cpp
	@echo "Running tests (Windows)..."
	CGO_ENABLED=1 \
	CGO_CFLAGS="-I$(shell pwd)/vendor/whisper.cpp" \
	CGO_LDFLAGS="-L$(shell pwd)/vendor/whisper.cpp -lwhisper" \
	go test -mod=mod -v $(if $(TEST),$(TEST),./...)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf vendor/whisper.cpp
	@echo "✓ Clean complete"

# Help
help:
	@echo "WhisperTray Makefile Commands:"
	@echo ""
	@echo "  make                    - Build everything (whisper.cpp + binary for current platform)"
	@echo "  make whisper-cpp        - Setup and build whisper.cpp"
	@echo "  make build              - Build binary for current platform (auto-detect)"
	@echo "  make build-mac          - Build macOS binary"
	@echo "  make build-linux        - Build Linux binary"
	@echo "  make build-windows      - Build Windows binary"
	@echo "  make build-mac-app      - Build macOS .app bundle"
	@echo "  make build-mac-universal - Build universal (Intel + Apple Silicon) binary"
	@echo "  make dev                - Quick dev build (skip whisper.cpp)"
	@echo "  make run                - Build and run"
	@echo "  make install-deps       - Install Go dependencies"
	@echo "  make test               - Run tests (auto-detect platform)"
	@echo "  make test TEST=<path>   - Run specific test package"
	@echo "  make test-osx           - Run tests on macOS"
	@echo "  make test-linux         - Run tests on Linux"
	@echo "  make test-windows       - Run tests on Windows"
	@echo "  make clean              - Remove build artifacts"
	@echo ""