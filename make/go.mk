# By default the project should be build under GOPATH/src/github.com/<orgname>/<reponame>
GO_PACKAGE_ORG_NAME ?= $(shell basename $$(dirname $$PWD))
GO_PACKAGE_REPO_NAME ?= $(shell basename $$PWD)
GO_PACKAGE_PATH ?= github.com/${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}

# enable Go modules
GO111MODULE?=on
export GO111MODULE

.PHONY: build
## runs go build
build:
	$(Q)CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
	    go build ./...

.PHONY: verify-dependencies
## Runs commands to verify after the updated dependecies of toolchain-common/API(go mod replace), if the repo needs any changes to be made
verify-dependencies: tidy vet build test lint-go-code

.PHONY: tidy
## runs go mod tidy
tidy: 
	go mod tidy

.PHONY: vet
## runs go mod vet ./...
vet:
	go vet ./...

.PHONY: verify-replace-run
## downloads all the repos that depend on toolchain-common, installs the current version of the library and runs all the verifications in order to check for compatibility and breaking changes
verify-replace-run:
	./scripts/verify-replace.sh; 