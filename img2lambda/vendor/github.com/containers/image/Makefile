.PHONY: all tools test validate lint .gitvalidation fmt

export GOPROXY=https://proxy.golang.org

# Which github repository and branch to use for testing with skopeo
SKOPEO_REPO = containers/skopeo
SKOPEO_BRANCH = master
# Set SUDO=sudo to run container integration tests using sudo.
SUDO =

# when cross compiling _for_ a Darwin or windows host, then we must use openpgp
BUILD_TAGS_WINDOWS_CROSS = containers_image_openpgp
BUILD_TAGS_DARWIN_CROSS = containers_image_openpgp

BUILDTAGS = btrfs_noversion libdm_no_deferred_remove
BUILDFLAGS := -tags "$(BUILDTAGS)"

PACKAGES := $(shell go list $(BUILDFLAGS) ./... | grep -v github.com/containers/image/vendor)
SOURCE_DIRS = $(shell echo $(PACKAGES) | awk 'BEGIN{FS="/"; RS=" "}{print $$4}' | uniq)

PREFIX ?= ${DESTDIR}/usr
MANINSTALLDIR=${PREFIX}/share/man
GOMD2MAN ?= $(shell command -v go-md2man || echo '$(GOBIN)/go-md2man')
MANPAGES_MD = $(wildcard docs/*.5.md)
MANPAGES ?= $(MANPAGES_MD:%.md=%)

# On macOS, (brew install gpgme) installs it within /usr/local, but /usr/local/include is not in the default search path.
# Rather than hard-code this directory, use gpgme-config. Sadly that must be done at the top-level user
# instead of locally in the gpgme subpackage, because cgo supports only pkg-config, not general shell scripts,
# and gpgme does not install a pkg-config file.
# If gpgme is not installed or gpgme-config can’t be found for other reasons, the error is silently ignored
# (and the user will probably find out because the cgo compilation will fail).
GPGME_ENV = CGO_CFLAGS="$(shell gpgme-config --cflags 2>/dev/null)" CGO_LDFLAGS="$(shell gpgme-config --libs 2>/dev/null)"

all: tools test validate .gitvalidation

build:
	$(GPGME_ENV) GO111MODULE="on" go build $(BUILDFLAGS) $(PACKAGES)

$(MANPAGES): %: %.md
	$(GOMD2MAN) -in $< -out $@

docs: $(MANPAGES)

install-docs: docs
	install -d -m 755 ${MANINSTALLDIR}/man5
	install -m 644 docs/*.5 ${MANINSTALLDIR}/man5/

install: install-docs

cross:
	GOOS=windows $(MAKE) build BUILDTAGS="$(BUILDTAGS) $(BUILD_TAGS_WINDOWS_CROSS)"
	GOOS=darwin $(MAKE) build BUILDTAGS="$(BUILDTAGS) $(BUILD_TAGS_DARWIN_CROSS)"

tools: tools.timestamp

tools.timestamp: Makefile
	@GO111MODULE="off" go get -u $(BUILDFLAGS) golang.org/x/lint/golint
	@GO111MODULE="off" go get $(BUILDFLAGS) github.com/vbatts/git-validation
	@touch tools.timestamp

clean:
	rm -rf tools.timestamp $(MANPAGES)

test:
	@$(GPGME_ENV) GO111MODULE="on" go test $(BUILDFLAGS) -cover $(PACKAGES)

# This is not run as part of (make all), but Travis CI does run this.
# Demonstrating a working version of skopeo (possibly with modified SKOPEO_REPO/SKOPEO_BRANCH, e.g.
#    make test-skopeo SKOPEO_REPO=runcom/skopeo-1 SKOPEO_BRANCH=oci-3 SUDO=sudo
# ) is a requirement before merging; note that Travis will only test
# the master branch of the upstream repo.
test-skopeo:
	@echo === Testing skopeo build
	@project_path=$$(pwd) && export GOPATH=$$(mktemp -d) && \
		skopeo_path=$${GOPATH}/src/github.com/containers/skopeo && \
		vendor_path=$${skopeo_path}/vendor/github.com/containers/image && \
		git clone -b $(SKOPEO_BRANCH) https://github.com/$(SKOPEO_REPO) $${skopeo_path} && \
		cd $${skopeo_path} && \
		GO111MODULE="on" go mod edit -replace github.com/containers/image=$${project_path} && \
		make vendor && \
		make BUILDTAGS="$(BUILDTAGS)" binary-local test-all-local && \
		$(SUDO) make BUILDTAGS="$(BUILDTAGS)" check && \
		rm -rf $${skopeo_path}

fmt:
	@gofmt -l -s -w $(SOURCE_DIRS)

validate: lint
	@GO111MODULE="on" go vet $(PACKAGES)
	@test -z "$$(gofmt -s -l . | grep -ve '^vendor' | tee /dev/stderr)"

lint:
	@out="$$(golint $(PACKAGES))"; \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		exit 1; \
	fi

# When this is running in travis, it will only check the travis commit range
.gitvalidation:
	@which git-validation > /dev/null 2>/dev/null || (echo "ERROR: git-validation not found. Consider 'make clean && make tools'" && false)
ifeq ($(TRAVIS),true)
	git-validation -q -run DCO,short-subject,dangling-whitespace
else
	git fetch -q "https://github.com/containers/image.git" "refs/heads/master"
	upstream="$$(git rev-parse --verify FETCH_HEAD)" ; \
		git-validation -q -run DCO,short-subject,dangling-whitespace -range $$upstream..HEAD
endif
