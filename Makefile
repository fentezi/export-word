BINARY_NAME=export-word
BUILD_DIR=./build
EXPORT_BIN=$(BUILD_DIR)/$(BINARY_NAME)
MAIN_FILE=./cmd/export-word/main.go

.PHONY: line run

all: run

lint:
	golangci-lint run

run: build
	$(EXPORT_BIN)

build: clean
	go build -o $(EXPORT_BIN) $(MAIN_FILE)

clean:
	@rm -rf $(BUILD_DIR)