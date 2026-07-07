BIN := bin/gobuddy
GO := go
GOFMT := gofmt

.PHONY: all
all: fmt build

.PHONY: build
build:
	@mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/gobuddy

.PHONY: test
test:
	$(GO) test -v -cover ./...

.PHONY: fmt
fmt:
	$(GOFMT) -w .

.PHONY: check
check:
	@unformatted="$$($(GOFMT) -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt violations:"; echo "$$unformatted"; exit 1; \
	fi
	$(GO) vet ./...
	$(GO) build ./...
	$(GO) test ./...

.PHONY: release-check
release-check: check
	jq empty .claude-plugin/plugin.json
	jq empty .claude-plugin/marketplace.json
	jq -e '.name == "gobuddy"' .claude-plugin/plugin.json > /dev/null
	bash -n hooks/gofmt-check.sh
	test -x hooks/gofmt-check.sh
	test -f commands/check.md
	test -f skills/go-standards/SKILL.md
	test -f CHANGELOG.md
	@echo "release-check: OK"

.PHONY: clean
clean:
	rm -rf bin

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all    - Format code and build (default)"
	@echo "  build  - Build gobuddy binary"
	@echo "  test   - Run unit tests with coverage"
	@echo "  fmt    - Format all Go code"
	@echo "  check  - Run fmt check, vet, build, and tests (CI gate)"
	@echo "  release-check - Run check plus plugin manifest and hook validation"
	@echo "  clean  - Remove build artifacts"
