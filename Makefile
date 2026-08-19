# LibBusinessID spec repository.
#
# Every generation tool is version locked in tools/go.mod. `make generate`
# rebuilds the Protobuf code, and `make compile` rebuilds every artifact.

SHELL := /bin/bash
GO ?= go
BIN := $(CURDIR)/bin
DIST ?= $(CURDIR)/dist
RULES_VERSION := $(shell cat RULES_VERSION)
SOURCE_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct 2>/dev/null || echo 0)
COVERAGE_LINE_MIN ?= 95
COVERAGE_BRANCH_MIN ?= 90

export SOURCE_DATE_EPOCH

.PHONY: all
all: fmt lint test compile

.PHONY: tools
tools:
	@mkdir -p $(BIN)
	cd tools/pinned && $(GO) build -o $(BIN)/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
	cd tools/pinned && $(GO) build -o $(BIN)/goimports golang.org/x/tools/cmd/goimports

.PHONY: generate
generate: tools
	PATH="$(BIN):$$PATH" buf generate
	$(GO) run tools/genfixtures.go

.PHONY: fmt
fmt: tools
	$(GO) run ./cmd/businessidc fmt
	$(BIN)/goimports -w -local github.com/libbusinessid/spec ./cmd ./internal
	gofmt -w ./cmd ./internal

.PHONY: fmt-check
fmt-check: tools
	@test -z "$$(gofmt -l ./cmd ./internal)" || { echo "gofmt reported files:"; gofmt -l ./cmd ./internal; exit 1; }
	@test -z "$$($(BIN)/goimports -l -local github.com/libbusinessid/spec ./cmd ./internal)" || \
		{ echo "goimports reported files:"; $(BIN)/goimports -l -local github.com/libbusinessid/spec ./cmd ./internal; exit 1; }
	$(GO) run ./cmd/businessidc fmt --check

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint: vet
	golangci-lint run
	$(GO) run ./cmd/businessidc lint

.PHONY: proto-lint
proto-lint:
	buf lint

.PHONY: proto-breaking
proto-breaking:
	buf breaking --against '.git#branch=main,subdir=.'

.PHONY: test
test:
	$(GO) test ./...

.PHONY: race
race:
	$(GO) test -race ./...

.PHONY: cover
cover:
	$(GO) test -covermode=atomic -coverprofile=coverage.out -coverpkg=./cmd/...,./internal/... ./...
	$(GO) run ./tools/coverage -profile coverage.out -line-min $(COVERAGE_LINE_MIN) -branch-min $(COVERAGE_BRANCH_MIN)

.PHONY: fuzz-smoke
fuzz-smoke:
	$(GO) test -run '^$$' -fuzz FuzzParseHCL -fuzztime 20s ./internal/hcllang
	$(GO) test -run '^$$' -fuzz FuzzReadJSONL -fuzztime 20s ./internal/conformance
	$(GO) test -run '^$$' -fuzz FuzzLoadRuleset -fuzztime 20s ./internal/artifact
	$(GO) test -run '^$$' -fuzz FuzzMutateBundle -fuzztime 20s ./internal/artifact
	$(GO) test -run '^$$' -fuzz FuzzValidateInput -fuzztime 20s ./internal/reference
	$(GO) test -run '^$$' -fuzz FuzzCompileUnit -fuzztime 20s ./internal/typecheck

.PHONY: vulncheck
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

.PHONY: compile
compile:
	$(GO) run ./cmd/businessidc compile --out $(DIST) --write-docs --source-commit $(SOURCE_COMMIT)

.PHONY: release
release:
	$(GO) run ./cmd/businessidc compile --out $(DIST) --release --write-docs --source-commit $(SOURCE_COMMIT)

.PHONY: verify
verify:
	$(GO) run ./cmd/businessidc verify

.PHONY: check-generated
check-generated:
	$(GO) run ./cmd/businessidc check-generated

.PHONY: sbom
sbom: compile
	@echo "SBOM written to $(DIST)/SBOM.spdx.json"

.PHONY: clean
clean:
	rm -rf $(DIST) $(BIN) coverage.out coverage.html coverage-branch.json

.PHONY: ci
ci: fmt-check proto-lint vet lint test race cover check-generated verify
