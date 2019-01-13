#!/bin/bash

# Normalize to working directory being build root (up one level from ./scripts)
ROOT=$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )
cd "${ROOT}"

# Builds the binary from source in the specified destination paths.
mkdir -p $1

cd "${ROOT}"

BUILDTAGS="containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_overlay exclude_graphdriver_btrfs containers_image_openpgp"

VERSION_LDFLAGS=""
if [[ -n "${2}" ]]; then
  VERSION_LDFLAGS="-X main.Version=${2}"
fi

if [[ -n "${3}" ]]; then
  VERSION_LDFLAGS="$VERSION_LDFLAGS -X main.GitCommitSHA=${3}"
fi

GOOS=$TARGET_GOOS go build -a -tags="${BUILDTAGS}" -ldflags "-s ${VERSION_LDFLAGS}" -o $1/img2lambda .
