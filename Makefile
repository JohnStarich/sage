# Force Go modules, even if in GOPATH
GO111MODULE := on
export
SUPPORTED_ARCH := windows/386 windows/amd64 darwin/amd64 linux/386 linux/amd64
SHELL := /usr/bin/env bash

ifdef TRAVIS_TAG
VERSION ?= ${TRAVIS_TAG}
endif
VERSION ?= $(shell git rev-parse --verify HEAD)
VERSION_FLAGS := -ldflags='-s -w -X github.com/johnstarich/sage/consts.Version=${VERSION}'
LINT_VERSION=1.23.3

# Ensure there's at least an empty bindata file when executing a target
ENSURE_STUB := $(shell [[ -f ./server/bindata.go ]] || { mkdir -p web/build && GO111MODULE=on go generate ./server; })

.PHONY: all
all: lint test build

.PHONY: version
version:
	@echo ${VERSION}

.PHONY: lint
lint:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: test
test:
	set -e; \
	tmpfile=$$(mktemp); \
	trap 'rm -f "$$tmpfile"' EXIT; \
	go test ./... -race -cover -coverprofile "$$tmpfile" >&2; \
	coverage=$$(go tool cover -func "$$tmpfile" | tail -1 | awk '{print $$3}'); \
	printf '##########################\n' >&2; \
	printf '### Coverage is %6s ###\n' "$$coverage" >&2; \
	printf '##########################\n' >&2; \
	echo "$$coverage"; \
	if [[ -n "$$COVERALLS_TOKEN" ]]; then \
		go get github.com/mattn/goveralls; \
		goveralls -coverprofile="$$tmpfile" -service=travis-ci -repotoken "$$COVERALLS_TOKEN"; \
	fi

.PHONY: build
build: static
	go build ${VERSION_FLAGS} -o out/sage

.PHONY: docker
docker:
	./goproxy.sh \
		docker build \
			--build-arg VERSION=${VERSION} \
			-t johnstarich/sage:${VERSION} \
			.

.PHONY: clean
clean: cache out
	rm -rf cache/ out/ web/out/

cache:
	mkdir cache

out:
	mkdir out

cache/ofxhome.xml: cache
	# API v1.1.2
	if [[ ! -f cache/ofxhome.xml ]]; then \
		curl -v -o cache/ofxhome.xml http://www.ofxhome.com/api.php?dump=yes; \
	else \
		touch cache/ofxhome.xml; \
	fi

# Try to create easily-scripted file names for download
$(SUPPORTED_ARCH): GOOS = $(@D)
$(SUPPORTED_ARCH): GOARCH = $(@F)
$(SUPPORTED_ARCH): CGO_ENABLED = 0
windows/%: EXT = .exe
%/386: ARCH = i386
%/amd64: ARCH = x86_64
$(SUPPORTED_ARCH): out static
	go build -v ${VERSION_FLAGS} -o out/sage-${VERSION}-${GOOS}-${ARCH}${EXT}

.PHONY: dist
dist: $(SUPPORTED_ARCH)

.PHONY: static-deps
static-deps:
	npm ci --prefix=web

.PHONY: static
static: cache/ofxhome.xml static-deps
	npm run --prefix=web build
	# Unset vars from upcoming targets
	GOOS= GOARCH= go generate ./server ./client/direct/drivers

.PHONY: start
start:
	trap 'jobs -p | xargs kill' EXIT; \
	mkdir -p ./data; \
	npm --prefix=web run start-api & \
	npm --prefix=web start

.PHONY: start-app
start-app:
	@if [[ -e out/sage ]]; then \
		echo "Make sure you ran 'make build' recently!"; \
	else \
		$(MAKE) build; \
	fi
	npm --prefix=web run start-app

.PHONY: start-pass
start-pass:
	trap 'jobs -p | xargs kill' EXIT; \
	mkdir -p ./data; \
	npm --prefix=web run start-api-pass & \
	npm --prefix=web start

.PHONY: poke-travis
poke-travis:
	# Print something out every minute for an hour to keep Travis from terminating the build early.
	(for i in {1..60}; do sleep 60; echo "Keeping Travis CI happy $$i"; done &)

.PHONY: release-servers
release-servers: clean
	$(MAKE) -j4 dist

.PHONY: release-mac
release-mac:
	@if [[ $$(uname) != Darwin ]]; then \
		echo '"release-mac" can only be run on macOS'; \
		exit 1; \
	fi
	$(MAKE) darwin/amd64
	F=web/node_modules/electron-packager/src/mac.js && \
	NEW_F=$$(sed '/if (!notarizeOpts.appleId) {/ { N;N;N;N;N;N;N;N;N; d; }' "$$F") && \
	  cat <<<"$$NEW_F" > "$$F"  # Temporary hack to enable API key notarization
	source .github/ci/utils.sh && \
		retry npm run --prefix=web mac
	mv -f web/out/make/Sage-1.0.0.dmg out/Sage-for-Mac.dmg

.PHONY: release-windows
release-windows: out windows/amd64
	docker run \
		--name sage-windows-builder \
		--rm -it \
		--env DEBUG='electron-windows-installer:*' \
		--env-file <(env | grep -iE 'DEBUG|NODE_|ELECTRON_|YARN_|NPM_|CI') \
		-v "${PWD}:/project:delegated" \
		electronuserland/builder:wine-mono make docker-windows

.PHONY: docker-windows
docker-windows:
	apt update
	apt install -y --no-install-recommends \
		fakeroot \
		p7zip \
		zip
	rm -rf ~/.wine
	fakeroot $(MAKE) static-deps
	# Fix wrong 7-zip architecture for win32 build
	wget -O /tmp/7z.7z https://www.7-zip.org/a/7z1900-extra.7z
	7zr x -o/tmp/7z-files /tmp/7z.7z
	cp /tmp/7z-files/7za.dll ./web/node_modules/electron-winstaller/vendor/7z.dll
	cp /tmp/7z-files/7za.exe ./web/node_modules/electron-winstaller/vendor/7z.exe
	npm config set loglevel verbose
	npm run --prefix=web windows
	chmod -R 777 web/out/make
	mv -f "web/out/make/squirrel.windows/x64/Sage-1.0.0 Setup.exe" out/Sage-for-Windows.exe

.PHONY: release-linux
release-linux: out linux/amd64
	docker run \
		--name sage-linux-builder \
		--rm -it \
		-v "${PWD}:/project:delegated" \
		--workdir /project \
		node:lts-buster make docker-linux

.PHONY: docker-linux
docker-linux:
	apt update
	apt install -y --no-install-recommends \
		fakeroot
	fakeroot $(MAKE) static-deps
	npm run --prefix=web linux
	mv web/out/make/deb/x64/sage_1.0.0_amd64.deb out/Sage-for-Linux.deb
