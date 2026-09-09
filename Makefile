BINARY := nadi-agent
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build build-linux test lint run mock run-mock clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)

mock:
	go build -o bin/mockserver ./cmd/mockserver

run-mock: mock
	./bin/mockserver

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 ./cmd/$(BINARY)

test:
	go test ./...

lint:
	gofmt -l . && go vet ./...

run:
	go run ./cmd/$(BINARY)

clean:
	rm -rf bin
