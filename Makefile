# Force Go modules, even if in GOPATH
GO111MODULE := on
export
SUPPORTED_ARCH := windows/386 windows/amd64 darwin/amd64 linux/386 linux/amd64
SHELL := /usr/bin/env bash

ifdef TRAVIS_TAG
VERSION ?= ${TRAVIS_TAG}
endif
VERSION ?= $(shell git rev-parse --verify HEAD)

.PHONY: all
all: fmt vet test build

.PHONY: version
version:
	@echo ${VERSION}

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	@diff=$$(gofmt -d .); \
		if [[ -n "$$diff" ]]; then \
			echo "$$diff"; \
			echo 'Formatting error. Run `go fmt ./...` to pass this linter.'; \
			exit 1; \
		fi

.PHONY: test
test:
	./coverage.sh

.PHONY: build
build:
	go build -ldflags='-X github.com/johnstarich/sage/consts.Version=${VERSION}'

.PHONY: docker
docker:
	docker build -t johnstarich/sage:${VERSION} .

.PHONY: clean
clean: out
	rm -rf out/

out:
	mkdir out

# Try to create easily-scripted file names for download
$(SUPPORTED_ARCH): GOOS = $(@D)
$(SUPPORTED_ARCH): GOARCH = $(@F)
windows/%: EXT = .exe
%/386: ARCH = i386
%/amd64: ARCH = x86_64
$(SUPPORTED_ARCH): clean out
	go build -v -o out/sage-${VERSION}-${GOOS}-${ARCH}${EXT}

.PHONY: dist
dist: $(SUPPORTED_ARCH)
