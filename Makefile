.DEFAULT_GOAL:=help
-include .makerc

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .husky
	@:

.PHONY: .husky
# Configure git hooks for husky
.husky:
	@if ! command -v husky &> /dev/null; then \
		echo "ERROR: missing executeable 'husky', please run:"; \
		echo "\n$ go install github.com/go-courier/husky/cmd/husky@latest\n"; \
	fi
	@git config core.hooksPath .husky

## === Tasks ===

.PHONY: doc
## Open go docs
doc:
	@open "http://localhost:6060/pkg/github.com/foomo/contentserver/"
	@godoc -http=localhost:6060 -play

.PHONY: test
## Run tests
test:
	@GO_TEST_TAGS=-skip go test -v -tags=safe -coverprofile=coverage.out -race -count=1 ./...
	#@GO_TEST_TAGS=-skip go test -tags=safe -coverprofile=coverage.out -race -json ./... | gotestfmt

.PHONY: test.update
## Run tests and update snapshots
test.update:
	@GO_TEST_TAGS=-skip go test -update -v -tags=safe -coverprofile=coverage.out -race ./...
	#@GO_TEST_TAGS=-skip go test -update -tags=safe -coverprofile=coverage.out -race -json ./... | gotestfmt

.PHONY: lint
## Run linter
lint:
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@golangci-lint run --fix

.PHONY: tidy
## Run go mod tidy
tidy:
	@go mod tidy

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@go list -u -m -json all | go-mod-outdated -update -direct

.PHONY: install
## Install binary
install:
	@go build -tags=safe -o ${GOPATH}/bin/contentserver main.go

.PHONY: build
## Build binary
build:
	@mkdir -p bin
	@go build -tags=safe -o bin/contentserver main.go

## === Utils ===

.PHONY: help
## Show help text
help:
	@awk '{ \
		if ($$0 ~ /^.PHONY: [a-zA-Z\-\_0-9]+$$/) { \
			helpCommand = substr($$0, index($$0, ":") + 2); \
			if (helpMessage) { \
				printf "\033[36m%-23s\033[0m %s\n", \
					helpCommand, helpMessage; \
				helpMessage = ""; \
			} \
		} else if ($$0 ~ /^[a-zA-Z\-\_0-9.]+:/) { \
			helpCommand = substr($$0, 0, index($$0, ":")); \
			if (helpMessage) { \
				printf "\033[36m%-23s\033[0m %s\n", \
					helpCommand, helpMessage"\n"; \
				helpMessage = ""; \
			} \
		} else if ($$0 ~ /^##/) { \
			if (helpMessage) { \
				helpMessage = helpMessage"\n                        "substr($$0, 3); \
			} else { \
				helpMessage = substr($$0, 3); \
			} \
		} else { \
			if (helpMessage) { \
				print "\n                        "helpMessage"\n" \
			} \
			helpMessage = ""; \
		} \
	}' \
	$(MAKEFILE_LIST)
