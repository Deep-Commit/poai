# Makefile for POAI

.PHONY: all build test clean run-poaid run-minectl

all: build

build:
	go build -o bin/poaid ./cmd/poaid
	go build -o bin/minectl ./cmd/minectl

test:
	go test ./...

clean:
	rm -rf bin/

run-poaid:
	./bin/poaid

run-minectl:
	./bin/minectl 