# Set Shell to bash, otherwise some targets fail with dash/zsh etc.
SHELL := /bin/bash

# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.SUFFIXES:
.SECONDARY:
.DEFAULT_GOAL := help

# General variables
include Makefile.vars.mk

# Following includes do not print warnings or error if files aren't found
# Optional Documentation module.
-include docs/antora-preview.mk docs/antora-build.mk
-include Makefile.compose.mk

.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-bin build-docker ## All-in-one build

.PHONY: build-bin
build-bin: export CGO_ENABLED = 1
build-bin: fmt vet ## Build binary
	@go build -o $(BIN_FILENAME) github.com/vshn/vshn-sli-reporting

.PHONY: build-docker
build-docker: build-bin ## Build docker image
	$(DOCKER_CMD) build -t $(CONTAINER_IMG) .

.PHONY: ensure-prometheus
ensure-prometheus: .cache/prometheus ## Ensures that Prometheus is installed in the project dir. Downloads it if necessary.

.PHONY: test
test: ensure-prometheus
	go test ./... -tags integration -coverprofile cover.out -covermode atomic

.PHONY: fmt
fmt: ## Run 'go fmt' and `jsonnetfmt` against code
	go fmt ./...
	find . \( -name '*.jsonnet' -o -name '*.libsonnet' \) -exec jsonnetfmt -i -- {} \;

.PHONY: vet
vet: ## Run 'go vet' against code
	go vet ./...

.PHONY: lint
lint: fmt vet generate ## All-in-one linting
	@echo 'Check for uncommitted changes ...'
	git diff --exit-code

.PHONY: generate
generate: ## Generate additional code and artifacts
	@go generate ./...

.PHONY: clean
clean:
	rm -rf docs/node_modules $(docs_out_dir) dist .cache

.cache/prometheus:
	mkdir -p .cache
	curl -fsSLo .cache/prometheus.tar.gz $(PROMETHEUS_DOWNLOAD_LINK)
	tar -xzf .cache/prometheus.tar.gz -C .cache
	mv .cache/prometheus-$(PROMETHEUS_VERSION).$(PROMETHEUS_DIST)-$(PROMETHEUS_ARCH) .cache/prometheus
	rm -rf .cache/*.tar.gz

# current date in UTC in ISO 8601 format (RFC 3339) with Z as timezone that works on both linux and macos
.PHONY: current-date
current-date:
	date -u +"%Y-%m-%dT%H:%M:%SZ"
