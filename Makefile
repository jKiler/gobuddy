GOBUDDY_BIN := bin/gobuddy
TEST_CLIENT_BIN := bin/test-client
GO := go
GOFMT := gofmt
GOTEST := $(GO) test
BIN_DIR := bin

.PHONY: all
all: fmt build

.PHONY: build
build: $(GOBUDDY_BIN) $(TEST_CLIENT_BIN)

$(GOBUDDY_BIN):
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(GOBUDDY_BIN) ./cmd/gobuddy

$(TEST_CLIENT_BIN):
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(TEST_CLIENT_BIN) ./cmd/test-client

.PHONY: test
test: build
	$(GOTEST) -v ./...
	@echo "\nRunning integration test..."
	./$(TEST_CLIENT_BIN)

.PHONY: fmt
fmt:
	$(GOFMT) -w .

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Format code and build all binaries (default)"
	@echo "  build        - Build all binaries"
	@echo "  test         - Run tests and integration tests"
	@echo "  fmt          - Format all Go code with gofmt"
	@echo "  clean        - Remove build artifacts"
	@echo "  help         - Show this help message"
