.PHONY: help build clean build-all package release
.DEFAULT_GOAL := help

pkg_root = .

### Calculate a few variables for use in building
VERSION = $(shell git describe --tags --abbrev=0 --always)
COMMIT = $(shell git log --pretty='format:%h' -n 1)
BUILDDATE = $(shell date +"%Y-%m-%dT%H:%M:%S")
# ldflags inject new values into variables at compilation time
# this is how we dynamically set the version/etc of the application
ldflags = "-X 'main.appVersion=$(VERSION)' \
	-X 'main.appCommit=$(COMMIT)' \
	-X 'main.buildTime=$(BUILDDATE)' \
	-w -s"

##@ Help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo
	@echo "Variable pkg_root is set to . by default. This should be the directory of spsrv source code."

##@ Utilities
init: ## Install utils
	go mod download

dep-tidy: ## Remove unused dependencies
	go mod tidy

dep-upgrade: ## Upgrade versions of dependencies
	go get -u

##@ Build
build: clean ## Build spsrv for your native architecture
	go build -o ./bin/spsrv -ldflags=$(ldflags) $(pkg_root)

build-all: clean ## Build spsrv for linux and mac
	GOOS=darwin GOARCH=amd64 go build -o ./bin/spsrv-darwin-amd64/spsrv -ldflags=$(ldflags) $(pkg_root)
	GOOS=darwin GOARCH=arm64 go build -o ./bin/spsrv-darwin-arm64/spsrv -ldflags=$(ldflags) $(pkg_root)
	GOOS=linux GOARCH=amd64 go build -o ./bin/spsrv-linux-amd64/spsrv -ldflags=$(ldflags) $(pkg_root)
	GOOS=linux GOARCH=arm64 go build -o ./bin/spsrv-linux-arm64/spsrv -ldflags=$(ldflags) $(pkg_root)

clean: ## Delete any compiled artifacts
	rm -rf ./bin

##@ Release
package: build-all ## Build everything and package up arch-specific tarballs
	tar czvf ./bin/spsrv-darwin-amd64.tar.gz ./bin/spsrv-darwin-amd64
	tar czvf ./bin/spsrv-darwin-arm64.tar.gz ./bin/spsrv-darwin-arm64
	tar czvf ./bin/spsrv-linux-amd64.tar.gz ./bin/spsrv-linux-amd64
	tar czvf ./bin/spsrv-linux-arm64.tar.gz ./bin/spsrv-linux-arm64

release: package ## Attach packages to sr.ht ref for current tag
	./_scripts/release.sh

##@ Test
test: ## Run tests
	go test $(shell go list ./...) -coverprofile=coverage.out
	# go tool cover -func=coverage.out
