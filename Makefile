CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --jobs=$(CPUS)

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: format
format: ## Format files
	@cfg=".golangci.yaml"; \
	if [ -f .golangci.local.yaml ]; then \
		go run github.com/mikefarah/yq/v4@v4.45.4 eval-all \
			'select(fileIndex == 0) *+ select(fileIndex == 1)' \
			.golangci.yaml .golangci.local.yaml > /tmp/.golangci.merged.yaml; \
		cfg="/tmp/.golangci.merged.yaml"; \
	fi; \
	if command -v custom-gcl >/dev/null 2>&1; then \
		custom-gcl fmt -c "$$cfg" ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 fmt -c "$$cfg" ./...; \
	fi
	@npx prettier@3.5.3 --write '**/*.md' '**/*.yaml' '**/*.yml' '**/*.json'

.PHONY: lint
lint: ## Lint files
	@cfg=".golangci.yaml"; \
	if [ -f .golangci.local.yaml ]; then \
		go run github.com/mikefarah/yq/v4@v4.45.4 eval-all \
			'select(fileIndex == 0) *+ select(fileIndex == 1)' \
			.golangci.yaml .golangci.local.yaml > /tmp/.golangci.merged.yaml; \
		cfg="/tmp/.golangci.merged.yaml"; \
	fi; \
	if command -v custom-gcl >/dev/null 2>&1; then \
		custom-gcl run --fix -c "$$cfg" ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 run --fix -c "$$cfg" ./...; \
	fi

.PHONY: build
build: ## Build all packages
	@go build ./...

.PHONY: test
test: ## Run tests with coverage and race detection
	@mkdir -p coverage
	@go run gotest.tools/gotestsum@v1.13.0 -- \
		-trimpath -race -count=1 -covermode=atomic \
		-coverprofile=coverage/coverage.cov \
		./...
	@go run github.com/boumenot/gocover-cobertura@v1.4.0 < coverage/coverage.cov > coverage/coverage.xml

.PHONY: bench
bench: ## Run benchmarks
	@go test -bench=. -benchmem ./...
