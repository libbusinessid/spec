# LibEntID spec repository.
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
	$(GO) run ./cmd/entidc fmt
	$(BIN)/goimports -w -local github.com/entid-org/spec ./cmd ./internal
	gofmt -w ./cmd ./internal

.PHONY: fmt-check
fmt-check: tools
	@test -z "$$(gofmt -l ./cmd ./internal)" || { echo "gofmt reported files:"; gofmt -l ./cmd ./internal; exit 1; }
	@test -z "$$($(BIN)/goimports -l -local github.com/entid-org/spec ./cmd ./internal)" || \
		{ echo "goimports reported files:"; $(BIN)/goimports -l -local github.com/entid-org/spec ./cmd ./internal; exit 1; }
	$(GO) run ./cmd/entidc fmt --check

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint: vet
	golangci-lint run
	$(GO) run ./cmd/entidc lint

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
# The smoke run is counted in executions, not seconds.
#
# A wall clock budget makes the result depend on how loaded the runner is: the
# fuzzing engine stops at the deadline and reports "context deadline exceeded"
# when a worker has not yet handed back, which looks like a finding and is not
# one. Counting executions asks the question the smoke run actually asks - does
# fuzzing start, and does anything crash - and gives the same answer on every
# machine. The weekly long run in scheduled.yml stays time based, because there
# the point is to spend a budget.
fuzz-smoke:
	$(GO) test -run '^$$' -fuzz FuzzParseHCL -fuzztime 200000x ./internal/hcllang
	$(GO) test -run '^$$' -fuzz FuzzReadJSONL -fuzztime 200000x ./internal/conformance
	$(GO) test -run '^$$' -fuzz FuzzLoadRuleset -fuzztime 200000x ./internal/artifact
	$(GO) test -run '^$$' -fuzz FuzzMutateBundle -fuzztime 200000x ./internal/artifact
	$(GO) test -run '^$$' -fuzz FuzzValidateInput -fuzztime 200000x ./internal/reference
	$(GO) test -run '^$$' -fuzz FuzzCompileUnit -fuzztime 200000x ./internal/typecheck

.PHONY: vulncheck
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

.PHONY: compile
compile:
	$(GO) run ./cmd/entidc compile --out $(DIST) --write-docs --source-commit $(SOURCE_COMMIT)

.PHONY: release
release:
	$(GO) run ./cmd/entidc compile --out $(DIST) --release --write-docs --source-commit $(SOURCE_COMMIT)

.PHONY: verify
verify:
	$(GO) run ./cmd/entidc verify

.PHONY: check-generated
check-generated:
	$(GO) run ./cmd/entidc check-generated

# conformance runs the whole corpus against the reference testee through the
# same protocol an external engine uses. The unit tests only exercise a subset;
# this target is what states conformance.
.PHONY: conformance
conformance: release
	$(GO) build -o $(BIN)/conformance-testee ./cmd/conformance-testee
	$(GO) run ./cmd/conformance-runner \
		--corpus $(DIST)/entid-conformance-$$(cat RULES_VERSION).binpb \
		-- $(BIN)/conformance-testee --bundle $(DIST)/entid-rules-$$(cat RULES_VERSION).binpb

.PHONY: sbom
sbom: compile
	@echo "SBOM written to $(DIST)/SBOM.spdx.json"

.PHONY: clean
clean:
	rm -rf $(DIST) $(BIN) coverage.out coverage.html coverage-branch.json

.PHONY: ci
ci: fmt-check proto-lint vet lint test race cover check-generated verify conformance
