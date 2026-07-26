# Makefile

# Define the output binary name
BINARY_NAME=hecato
MAIN_PATH=cmd/hecato/main.go
MODULE=github.com/curiousjc/hecato

# Build metadata. Overridable from the environment so CI can pin exact values.
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Selects where heclog writes app.log. "development" pins it to the hardcoded
# repo path in internal/heclog; anything else resolves it next to the binary.
BUILD_CONTEXT ?= development

LDFLAGS = -X main.buildContext=$(BUILD_CONTEXT) \
          -X $(MODULE)/internal/version.Version=$(VERSION) \
          -X $(MODULE)/internal/version.Commit=$(COMMIT) \
          -X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME)

# Default target executed when no arguments are given to 'make'
all: build-linux build-windows

# Build the project
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME).exe $(MAIN_PATH)

# What CI builds. Same targets, but the binaries log beside themselves rather
# than into c:/repos/hecato.
release:
	$(MAKE) all BUILD_CONTEXT=release

# Clean the build output
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

# Run tests
test:
	go test ./...

vet:
	go vet ./...

.PHONY: all build-linux build-windows release clean test vet
