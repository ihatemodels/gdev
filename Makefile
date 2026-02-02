BINARY_NAME := gdev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: build run clean

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME) $(ARGS)

clean:
	rm -f $(BINARY_NAME)
