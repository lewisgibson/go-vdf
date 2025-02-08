CPUS ?= $(shell (nproc --all || sysctl -n hw.ncpu) 2>/dev/null || echo 1)
MAKEFLAGS += --jobs=$(CPUS)

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: format
format: ## Format files
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 fmt ./...

.PHONY: lint
lint: ## Lint files
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 run ./...

.PHONY: build
build: ## Build the application
	@go build ./...

.PHONY: test
test: ## Run tests
	@mkdir -p coverage
	@go run gotest.tools/gotestsum@latest -- \
		-race -count=1 -covermode=atomic \
		-coverprofile=coverage/coverage.cov \
		./...
	@go run github.com/axw/gocov/gocov@latest convert coverage/coverage.cov | go run github.com/AlekSi/gocov-xml@latest > coverage/coverage.xml

.PHONY: bench
bench: ## Run benchmarks
	@go test -bench=. -benchmem ./...
