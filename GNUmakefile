# GNUmakefile — Terraform provider convention (make picks this over Makefile).

BINARY   := terraform-provider-bbox
VERSION  := 0.1.0
OS_ARCH  := $(shell go env GOOS)_$(shell go env GOARCH)
PLUGIN_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/hadamrd/bbox/$(VERSION)/$(OS_ARCH)

ifeq ($(OS),Windows_NT)
BIN_EXT := .exe
else
BIN_EXT :=
endif

.PHONY: build install test testint testacc docs fmt vet lint clean

build:
	go build -o bin/$(BINARY)$(BIN_EXT) .

install: build
	mkdir -p $(PLUGIN_DIR)
	cp bin/$(BINARY)$(BIN_EXT) $(PLUGIN_DIR)/$(BINARY)_v$(VERSION)$(BIN_EXT)
	@echo "Installed to $(PLUGIN_DIR)"

test:
	go test ./... -race -count=1

# Unit-style integration tests: full CRUD driven through terraform, backed by
# an httptest mock router — no live Bbox needed. Runs whenever TF_ACC is unset.
testint:
	go test ./internal/provider/... -run Integration -race -count=1

# Acceptance tests: exercise a REAL Bbox at BBOX_BASE_URL. Refuses to run
# without TF_ACC=1 so `make test` never trips them.
testacc:
ifeq ($(TF_ACC),)
	@echo "Acceptance tests require a live Bbox router. Set TF_ACC=1 and BBOX_PASSWORD_FILE, then run:"
	@echo "  TF_ACC=1 BBOX_PASSWORD_FILE=\$$HOME/.bbox-password make testacc"
else
	go test ./internal/provider/... -run TestAcc -v -timeout 30m
endif

docs:
	@echo "Docs live in docs/ and are hand-maintained; nothing to generate."

fmt:
	gofmt -w .
	terraform fmt -recursive examples/

vet:
	go vet ./...

lint: vet
	@gofmt -l . | tee /dev/stderr | (! read line)

clean:
	rm -rf bin/
