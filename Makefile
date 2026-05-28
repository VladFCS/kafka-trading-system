.DEFAULT_GOAL := help

GO ?= go
GOFMT ?= gofmt
PROTOC ?= protoc
GOLANGCI_LINT ?= golangci-lint
SQLC ?= sqlc
COMPOSE ?= docker compose

GOLANGCI_LINT_PKG ?= github.com/golangci/golangci-lint/cmd/golangci-lint@latest
PROTOC_GEN_GO_PKG ?= google.golang.org/protobuf/cmd/protoc-gen-go@latest
PROTOC_GEN_GO_GRPC_PKG ?= google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
SQLC_PKG ?= github.com/sqlc-dev/sqlc/cmd/sqlc@latest

GOFLAGS ?= -buildvcs=false
GOCACHE ?= $(CURDIR)/.cache/go-build
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint
BIN_DIR ?= $(CURDIR)/bin
PROTO_DIR ?= api/proto
GEN_DIR ?= gen

SERVICE_DIRS := $(sort $(dir $(wildcard cmd/*/main.go)))
SERVICE_NAMES := $(patsubst cmd/%/,%,$(SERVICE_DIRS))

.PHONY: help doctor fmt lint lint-install vet test check tidy proto proto-check \
	proto-install sqlc-generate sqlc-verify sqlc-install build build-services \
	compose-up compose-down compose-logs clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

doctor: ## Check required local tooling
	@command -v $(GO) >/dev/null || { echo "$(GO) is not installed or not in PATH"; exit 1; }
	@command -v $(GOFMT) >/dev/null || { echo "$(GOFMT) is not installed or not in PATH"; exit 1; }
	@command -v $(PROTOC) >/dev/null || { echo "$(PROTOC) is not installed or not in PATH"; exit 1; }
	@command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go is not installed or not in PATH"; echo "install it with: make proto-install"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc is not installed or not in PATH"; echo "install it with: make proto-install"; exit 1; }
	@echo "basic tooling looks good"

fmt: ## Format Go code
	@files=$$(find . -type f -name '*.go' -not -path './vendor/*'); \
	if [ -z "$$files" ]; then \
		echo "no Go files to format"; \
	else \
		$(GOFMT) -w $$files; \
	fi

lint: ## Run golangci-lint
	@command -v $(GOLANGCI_LINT) >/dev/null || { \
		echo "$(GOLANGCI_LINT) is not installed or not in PATH"; \
		echo "install it with: make lint-install"; \
		exit 1; \
	}
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run --timeout=5m ./...

lint-install: ## Install or rebuild golangci-lint
	GOCACHE=$(GOCACHE) $(GO) install $(GOLANGCI_LINT_PKG)

vet: ## Run go vet
	GOCACHE=$(GOCACHE) $(GO) vet $(GOFLAGS) ./...

test: ## Run all Go tests
	GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) ./...

check: fmt lint vet test ## Run the default verification suite

tidy: ## Run go mod tidy
	$(GO) mod tidy

proto: ## Generate protobuf and gRPC code into ./gen
	@proto_files=$$(find "$(PROTO_DIR)" -type f -name '*.proto'); \
	if [ -z "$$proto_files" ]; then \
		echo "no .proto files found under $(PROTO_DIR)"; \
		exit 0; \
	fi; \
	mkdir -p "$(GEN_DIR)"; \
	PATH="$(PATH):$(HOME)/go/bin" $(PROTOC) -I "$(PROTO_DIR)" \
		--go_out="$(GEN_DIR)" --go_opt=paths=source_relative \
		--go-grpc_out="$(GEN_DIR)" --go-grpc_opt=paths=source_relative \
		$$proto_files

proto-check: ## Regenerate protobufs and fail if generated files changed
	@$(MAKE) proto
	@git diff --quiet -- $(GEN_DIR) || { \
		echo "generated protobuf files are out of date"; \
		echo "run 'make proto' and commit the updated files"; \
		git diff -- $(GEN_DIR); \
		exit 1; \
	}

proto-install: ## Install protoc Go plugins
	GOCACHE=$(GOCACHE) $(GO) install $(PROTOC_GEN_GO_PKG)
	GOCACHE=$(GOCACHE) $(GO) install $(PROTOC_GEN_GO_GRPC_PKG)

sqlc-generate: ## Generate Go code from SQL using sqlc
	$(SQLC) generate

sqlc-verify: ## Verify sqlc queries and config
	$(SQLC) vet

sqlc-install: ## Install or rebuild sqlc
	GOCACHE=$(GOCACHE) $(GO) install $(SQLC_PKG)

build: build-services ## Build all discovered service binaries into ./bin

build-services: ## Build all services found under ./cmd
	@mkdir -p "$(BIN_DIR)"
	@if [ -z "$(SERVICE_NAMES)" ]; then \
		echo "no runnable services found under ./cmd"; \
		exit 0; \
	fi
	@for service in $(SERVICE_NAMES); do \
		echo "building $$service"; \
		GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/$$service" "./cmd/$$service"; \
	done

compose-up: ## Start local docker compose stack
	$(COMPOSE) up -d

compose-down: ## Stop local docker compose stack
	$(COMPOSE) down

compose-logs: ## Tail docker compose logs
	$(COMPOSE) logs -f

clean: ## Remove local build artifacts and caches
	rm -rf "$(BIN_DIR)" "$(CURDIR)/.cache"
