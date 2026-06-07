BIN     := bin/rtp-play
CMD     := ./cmd/rtp-play
GOLINT  := golangci-lint

.PHONY: all build run lint vet fmt tidy clean

all: lint build

## build: compile the binary into bin/
build:
	mkdir -p bin
	go build -o $(BIN) $(CMD)

## run: build and execute (streams PCMU to 127.0.0.1:5004)
run: build
	./$(BIN)

## lint: run golangci-lint
lint:
	$(GOLINT) run ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format source with gofmt
fmt:
	gofmt -w -s ./...

## tidy: tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## clean: remove the compiled binary
clean:
	rm -f $(BIN)

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
