BINARY := sssh
PKG    := ./cmd/sssh

.PHONY: help build install test race vet fmt guardrails check desktop-check hooks clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the sssh binary
	go build -o $(BINARY) $(PKG)

install: ## Install sssh into $$GOPATH/bin
	go install $(PKG)

test: ## Run the test suite
	go test ./...

race: ## Run the test suite with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Check formatting
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

guardrails: ## Enforce the architectural budgets
	@./scripts/guardrails.sh

check: fmt vet race guardrails ## Everything CI runs. This is the gate before any "done".

desktop-check: ## Gate for the Relay desktop module (guardrails, typecheck, frontend tests, build, Go tests)
	./desktop/scripts/guardrails.sh
	cd desktop/frontend && npm run check 2>&1 | tee /dev/stderr | grep -q " 0 ERRORS " && npm test && npm run build
	cd desktop && go vet . && go test .

hooks: ## Use the repo's git hooks (a pre-push that runs desktop-check on macOS)
	git config core.hooksPath .githooks

clean: ## Remove build output
	rm -f $(BINARY)

.DEFAULT_GOAL := help
