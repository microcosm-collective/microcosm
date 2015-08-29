# Targets:
#
#   all:          Builds the code locally after testing
#
#   fmt:          Formats the source files
#   build:        Builds the code locally
#   vet:          Vets the code
#   lint:         Runs lint over the code (you do not need to fix everything)
#   test:         Runs the tests
#   clean:        Deletes the locally built file (if it exists)
#
#   dep_restore:  Ensures all dependent packages are at the correct version
#   dep_update:   Ensures all dependent packages are at the latest version

.PHONY: all fmt build vet lint test clean dep_restore dep_update

# The first target is always the default action if `make` is called without args
all: clean fmt lint vet test build

fmt:
	@gofmt -w ./$*

build: export GOOS=linux
build: export GOARCH=amd64
build: clean
	@go build

vet:
	@go vet main.go
	@go vet audit/*
	@go vet cache/*
	@go vet config/*
	@go vet controller/*
	@go vet errors/*
	@go vet helpers/*
	@go vet models/*
	@go vet redirector/*
	@go vet resolver/*
	@go vet server/*

lint:
	@golint main.go
	@golint ./...

test:
	@go test -v ./...

clean:
	@find . -name microcosm -delete

dep_restore:
	@sudo apt-get -y install bzr
	@go get -u github.com/tools/godep
	@godep restore

dep_update:
	@rm -rf Godeps/
	@go get -u github.com/bradfitz/gomemcache/memcache
	@go get -u github.com/cloudflare/ahocorasick
	@go get -u github.com/disintegration/imaging
	@go get -u github.com/golang/glog
	@go get -u github.com/gorilla/mux
	@go get -u github.com/lib/pq
	@go get -u github.com/microcosm-cc/bluemonday
	@go get -u github.com/microcosm-cc/goconfig
	@go get -u github.com/microcosm-cc/exifutil
	@go get -u github.com/mitchellh/goamz/aws
	@go get -u github.com/mitchellh/goamz/s3
	@go get -u github.com/nytimes/gziphandler
	@go get -u github.com/robfig/cron
	@go get -u github.com/russross/blackfriday
	@go get -u github.com/rwcarlsen/goexif/exif
	@go get -u github.com/tools/godep
	@go get -u github.com/xtgo/uuid
	@go get -u golang.org/x/net/html
	@go get -u golang.org/x/oauth2
	@godep save ./...
	@make fmt
