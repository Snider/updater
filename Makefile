.PHONY: build dev release-local test coverage

BINARY_NAME=updater
CMD_PATH=./cmd/updater
VERSION ?= $(shell git describe --tags --always)
LDFLAGS = -ldflags "-X 'github.com/snider/updater/cmd.version=$(VERSION)'"

build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)/main.go

dev: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME) --check-update 

release-local:
	@echo "Running local release with GoReleaser..."
	@goreleaser release --snapshot --rm-dist

test:
	@echo "Running tests..."
	@go test ./...

coverage:
	@echo "Generating code coverage report..."
	@go test -coverprofile=coverage.out ./...
	@echo "Coverage report generated: coverage.out"
	@echo "To view in browser: go tool cover -html=coverage.out"
	@echo "To upload to Codecov, ensure you have the Codecov CLI installed (e.g., 'go install github.com/codecov/codecov-cli@latest') and run: codecov -f coverage.out"
