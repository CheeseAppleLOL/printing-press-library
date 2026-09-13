.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/swiggy-pp-cli$(BIN_EXT) ./cmd/swiggy-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/swiggy-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/swiggy-pp-mcp$(BIN_EXT) ./cmd/swiggy-pp-mcp

install-mcp:
	go install ./cmd/swiggy-pp-mcp

build-all: build build-mcp
