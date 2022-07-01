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

.PHONY: all fmt build vet test clean

# The first target is always the default action if `make` is called without args
all: clean vet test build

build: export GOOS=linux
build: export GOARCH=amd64
build: clean
	@go build cmd/microcosm/microcosm.go

vet:
	@go vet $$(go list ./... | grep -v /vendor/)

test:
	@go test $$(go list ./... | grep -v /vendor/)

clean:
	@find . -maxdepth 1 -name microcosm -delete
