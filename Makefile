.DEFAULT_GOAL:=help
-include .makerc

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .mise .lefthook
	@:

.PHONY: .mise
# Install dependencies
.mise:
ifeq (, $(shell command -v mise))
	$(error $(br)$(br)Please ensure you have 'mise' installed and activated!$(br)$(br)  $$ brew update$(br)  $$ brew install mise$(br)$(br)See the documentation: https://mise.jdx.dev/getting-started.html)
endif
	@mise install

.PHONY: .lefthook
# Configure git hooks for lefthook
.lefthook:
	@lefthook install --reset-hooks-path

### Tasks

.PHONY: check
## Run lint & tests
check: tidy lint.fix test.race audit

.PHONY: lint
## Run linter
lint:
	@echo "〉golangci-lint run"
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@echo "〉golangci-lint run fix"
	@golangci-lint run --fix

.PHONY: test
## Run tests
test:
	@echo "〉go test"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -shuffle=on ./...

.PHONY: test.race
## Run tests with -race
test.race:
	@echo "〉go test -race"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -shuffle=on -race ./...

.PHONY: test.update
## Run tests and update snapshots
test.update:
	@echo "〉go test -update"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -shuffle=on -update ./...

### Dependencies

.PHONY: tidy
## Run go mod tidy
tidy:
	@echo "〉go mod tidy"
	@go mod tidy

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@echo "〉go mod outdated"
	@go-mod-upgrade --list

.PHONY: upgrade
## Interactively select modules to upgrade
upgrade:
	@echo "〉go mod outdated"
	@go-mod-upgrade

### Binary

.PHONY: build
## Build binary
build:
	@echo "〉building"
	@mkdir -p bin
	@go build -tags=safe -o bin/contentserver main.go

.PHONY: install
## Install binary
install:
	@echo "〉installing"
	@go build -tags=safe -o ${GOPATH}/bin/contentserver main.go

.PHONY: release.snapshot
## Create a goreleaser snapshot release
release.snapshot:
	@echo "〉building release snapshot"
	@rm -rf ./dist
	@goreleaser release --snapshot


### Documentation

.PHONY: godocs
## Open go go docs
godocs:
	@echo "〉starting go docs"
	@go doc -http

### Utils

.PHONY: help
# https://patorjk.com/software/taag/#p=display&f=Tmplr&t=CONTENTSERVER&x=none&v=4&h=4&w=80&we=false
## Show help text
help: g=\033[0;32m
help: b=\033[0;34m
help: w=\033[0;90m
help: e=\033[0m
help:
	@echo "$(g)"
	@echo "┏┓┏┓┳┓┏┳┓┏┓┳┓┏┳┓┏┓┏┓┳┓┓┏┏┓┳┓"
	@echo "┃ ┃┃┃┃ ┃ ┣ ┃┃ ┃ ┗┓┣ ┣┫┃┃┣ ┣┫"
	@echo "┗┛┗┛┛┗ ┻ ┗┛┛┗ ┻ ┗┛┗┛┛┗┗┛┗┛┛┗"
	@echo "with ❤ foomo by bestbytes"
	@echo "$(e)"
	@echo "$(b)Usage:$(e)\n  make [task]"
	@awk '{ \
		if($$0 ~ /^### /){ \
			if(help) printf "  %-21s $(w)%s$(e)\n\n", cmd, help; help=""; \
			printf "$(b)\n%s:$(e)\n", substr($$0,5); \
		} else if($$0 ~ /^[a-zA-Z0-9._-]+:/){ \
			cmd = substr($$0, 1, index($$0, ":")-1); \
			if(help) printf "  %-21s $(w)%s$(e)\n", cmd, help; help=""; \
		} else if($$0 ~ /^##/){ \
			help = help ? help "\n                        " substr($$0,3) : substr($$0,3); \
		} else if(help){ \
			print "\n                        $(w)" help "$(e)\n"; help=""; \
		} \
	}' $(MAKEFILE_LIST)
	@echo ""

