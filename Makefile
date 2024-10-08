# Makefile

# Define the output binary name
BINARY_NAME=hecato
MAIN_PATH=cmd/hecato/main.go

# Default target executed when no arguments are given to 'make'
all: build-linux build-windows

# Build the project
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) $(MAIN_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(MAIN_PATH)

# Clean the build output
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

# Run tests
test:
	go test ./...

.PHONY: all build clean test
