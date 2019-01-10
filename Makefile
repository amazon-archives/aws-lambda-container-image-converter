ROOT := $(shell pwd)

all: build

SOURCEDIR := ./
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
LOCAL_BINARY := bin/local/img2lambda
LINUX_BINARY := bin/linux-amd64/img2lambda
DARWIN_BINARY := bin/darwin-amd64/img2lambda
WINDOWS_BINARY := bin/windows-amd64/img2lambda.exe
LOCAL_PATH := $(ROOT)/scripts:${PATH}
DEP_RELEASE_TAG := v0.4.1

.PHONY: build
build: $(LOCAL_BINARY)

$(LOCAL_BINARY): $(SOURCES)
	./scripts/build_binary.sh ./bin/local
	@echo "Built img2lambda"

.PHONY: test
test:
	env -i PATH=$$PATH GOPATH=$$GOPATH GOROOT=$$GOROOT go test -timeout=120s -v -cover ./...

.PHONY: generate
generate: $(SOURCES)
	PATH=$(LOCAL_PATH) go generate ./...

.PHONY: generate-deps
generate-deps:
	DEP_RELEASE_TAG=$DEP_RELEASE_TAG
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

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

.PHONY: supported-platforms
supported-platforms: $(LINUX_BINARY) $(DARWIN_BINARY) $(WINDOWS_BINARY)

$(WINDOWS_BINARY): $(SOURCES)
	@mkdir -p ./bin/windows-amd64
	TARGET_GOOS=windows GOARCH=amd64 ./scripts/build_binary.sh ./bin/windows-amd64
	mv ./bin/windows-amd64/img2lambda ./bin/windows-amd64/img2lambda.exe
	@echo "Built img2lambda.exe for windows"

$(LINUX_BINARY): $(SOURCES)
	@mkdir -p ./bin/linux-amd64
	TARGET_GOOS=linux GOARCH=amd64 ./scripts/build_binary.sh ./bin/linux-amd64
	@echo "Built img2lambda for linux"

$(DARWIN_BINARY): $(SOURCES)
	@mkdir -p ./bin/darwin-amd64
	TARGET_GOOS=darwin GOARCH=amd64 ./scripts/build_binary.sh ./bin/darwin-amd64
	@echo "Built img2lambda for darwin"

.PHONY: clean
clean:
rm -rf ./bin/ ||:
