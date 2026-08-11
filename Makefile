.PHONY: build test verify notices install assets package clean \
	dashboard-install dashboard-typecheck dashboard-test dashboard-build dashboard-verify

VERSION ?= dev
PREFIX ?= $(HOME)/.local
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ASSET_NAME = cc-dialect_$(VERSION)_$(GOOS)_$(GOARCH)
DASHBOARD_DIR = internal/app/dashboard
DASHBOARD_DIST = $(DASHBOARD_DIR)/dist

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dist/cc-dialect .

test:
	go test ./...

dashboard-install:
	npm --prefix "$(DASHBOARD_DIR)" ci

dashboard-typecheck: dashboard-install
	npm --prefix "$(DASHBOARD_DIR)" run typecheck

dashboard-test: dashboard-install
	npm --prefix "$(DASHBOARD_DIR)" test

dashboard-build: dashboard-install
	npm --prefix "$(DASHBOARD_DIR)" run build

dashboard-verify: dashboard-typecheck dashboard-test dashboard-build
	git ls-files --error-unmatch -- "$(DASHBOARD_DIST)/index.html" >/dev/null
	test -z "$$(git ls-files --others --exclude-standard -- "$(DASHBOARD_DIST)")"
	git diff --exit-code -- "$(DASHBOARD_DIST)"

verify: dashboard-verify
	test -z "$$(gofmt -l .)"
	go mod verify
	go test ./...
	go vet ./...
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...

notices: dashboard-install
	./scripts/generate-third-party-notices.sh

install: build
	mkdir -p "$(PREFIX)/bin"
	tmp="$(PREFIX)/bin/.cc-dialect.tmp.$$$$"; \
	cp dist/cc-dialect "$$tmp"; \
	chmod 755 "$$tmp"; \
	mv -f "$$tmp" "$(PREFIX)/bin/cc-dialect"; \
	rm -f "$(PREFIX)/bin/dialect"

assets: build
	rm -rf "artifacts/$(ASSET_NAME)"
	mkdir -p "artifacts/$(ASSET_NAME)"
	cp dist/cc-dialect LICENSE README.md THIRD_PARTY_NOTICES.md "artifacts/$(ASSET_NAME)/"
	cd artifacts && zip -qr "$(ASSET_NAME).zip" "$(ASSET_NAME)"
	cd artifacts && { command -v sha256sum >/dev/null 2>&1 && sha256sum "$(ASSET_NAME).zip" || shasum -a 256 "$(ASSET_NAME).zip"; } > SHA256SUMS

package: assets

clean:
	rm -rf dist artifacts
