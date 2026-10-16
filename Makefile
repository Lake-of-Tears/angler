.PHONY: all build test clean run

BINARY_NAME=angler
BUILD_DIR=bin

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR) dist

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)
