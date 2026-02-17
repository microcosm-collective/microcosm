# Targets:
#
#   all:          Builds the code locally after testing
#
#   fmt:          Formats the source files
#   build:        Builds the code locally
#   vet:          Vets the code
#   test:         Runs the tests
#   clean:        Deletes the locally built file (if it exists)
#
#   dep_restore:  Ensures all dependent packages are at the correct version
#   dep_update:   Ensures all dependent packages are at the latest version
GOCMD := go
date=$(shell date "+%Y-%m-%d_%H%M")
version=$(shell git log --format="%H" -n 1)

.PHONY: all fmt build vet test clean

# The first target is always the default action if `make` is called without args
all: clean vet test build

build: export GOOS=linux
build: export GOARCH=amd64
build: export CGO_ENABLED=0
build: clean
	@$(GOCMD) build -ldflags "-w -X 'main.version=$(version)' -X 'main.date=$(date)'" cmd/microcosm/microcosm.go

vet:
	@$(GOCMD) vet $$($(GOCMD) list ./... | grep -v /vendor/)

test:
	@$(GOCMD) test $$($(GOCMD) list ./... | grep -v /vendor/)

clean:
	@find . -maxdepth 1 -name microcosm -delete

.PHONY: deps
deps:
	@$(GOCMD) list -m -u -mod=mod all
	@$(GOCMD) mod tidy
	@$(GOCMD) get -d -u ./...
	@$(GOCMD) mod vendor
