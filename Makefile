# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

ROOT := $(shell pwd)

all: build

SOURCEDIR := ./img2lambda
BINARY_NAME=img2lambda
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
LOCAL_BINARY := bin/local/img2lambda
LINUX_BINARY := bin/linux-amd64/img2lambda
DARWIN_BINARY := bin/darwin-amd64/img2lambda
WINDOWS_BINARY := bin/windows-amd64/img2lambda.exe
LOCAL_PATH := $(ROOT)/scripts:${PATH}
VERSION := $(shell cat VERSION)
GITFILES := $(shell find ".git/")

.PHONY: build
build: $(LOCAL_BINARY)

$(LOCAL_BINARY): $(SOURCES) GITCOMMIT_SHA
	./scripts/build_binary.sh ./bin/local $(VERSION) $(shell cat GITCOMMIT_SHA)
	@echo "Built img2lambda"

.PHONY: test
test:
	go test -v -timeout 30s -short -cover $(shell go list ./img2lambda/... | grep -v /vendor/)

.PHONY: generate
generate: $(SOURCES)
	PATH=$(LOCAL_PATH) go generate -x $(shell go list ./img2lambda/... | grep -v '/vendor/')

.PHONY: install-deps
install-deps:
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	go get golang.org/x/tools/cmd/cover
	go get github.com/golang/mock/mockgen
	go get golang.org/x/tools/cmd/goimports

.PHONY: windows-build
windows-build: $(WINDOWS_BINARY)

.PHONY: docker-build
docker-build:
	docker run -v $(shell pwd):/usr/src/app/src/github.com/awslabs/img2lambda \
		--workdir=/usr/src/app/src/github.com/awslabs/img2lambda \
		--env GOPATH=/usr/src/app \
		golang:1.11 make $(LINUX_BINARY)
	docker run -v $(shell pwd):/usr/src/app/src/github.com/awslabs/img2lambda \
		--workdir=/usr/src/app/src/github.com/awslabs/img2lambda \
		--env GOPATH=/usr/src/app \
		golang:1.11 make $(DARWIN_BINARY)
	docker run -v $(shell pwd):/usr/src/app/src/github.com/awslabs/img2lambda \
		--workdir=/usr/src/app/src/github.com/awslabs/img2lambda \
		--env GOPATH=/usr/src/app \
		golang:1.11 make $(WINDOWS_BINARY)

.PHONY: docker-test
docker-test:
	docker run -v $(shell pwd):/usr/src/app/src/github.com/awslabs/img2lambda \
		--workdir=/usr/src/app/src/github.com/awslabs/img2lambda \
		--env GOPATH=/usr/src/app \
		--env IMG_TOOL_RELEASE=$(IMG_TOOL_RELEASE) \
		golang:1.11 make test

.PHONY: all-platforms
all-platforms: $(LINUX_BINARY) $(DARWIN_BINARY) $(WINDOWS_BINARY)

$(WINDOWS_BINARY): $(SOURCES) GITCOMMIT_SHA
	@mkdir -p ./bin/windows-amd64
	TARGET_GOOS=windows GOARCH=amd64 ./scripts/build_binary.sh ./bin/windows-amd64 $(VERSION) $(shell cat GITCOMMIT_SHA)
	mv ./bin/windows-amd64/img2lambda ./bin/windows-amd64/img2lambda.exe
	@echo "Built img2lambda.exe for windows"

$(LINUX_BINARY): $(SOURCES) GITCOMMIT_SHA
	@mkdir -p ./bin/linux-amd64
	TARGET_GOOS=linux GOARCH=amd64 ./scripts/build_binary.sh ./bin/linux-amd64 $(VERSION) $(shell cat GITCOMMIT_SHA)
	@echo "Built img2lambda for linux"

$(DARWIN_BINARY): $(SOURCES) GITCOMMIT_SHA
	@mkdir -p ./bin/darwin-amd64
	TARGET_GOOS=darwin GOARCH=amd64 ./scripts/build_binary.sh ./bin/darwin-amd64 $(VERSION) $(shell cat GITCOMMIT_SHA)
	@echo "Built img2lambda for darwin"

GITCOMMIT_SHA: $(GITFILES)
	git rev-parse --short=7 HEAD > GITCOMMIT_SHA

.PHONY: clean
clean:
	- rm -rf ./bin
	- rm -rf ./output
	- rm -f GITCOMMIT_SHA
