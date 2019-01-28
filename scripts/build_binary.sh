#!/bin/bash

set -e

# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

# Normalize to working directory being build root (up one level from ./scripts)
ROOT=$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )
cd "${ROOT}"

# Builds the binary from source in the specified destination paths.
mkdir -p $1

cd "${ROOT}"

PACKAGE_ROOT="github.com/awslabs/aws-lambda-container-image-converter/img2lambda"

BUILDTAGS="containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_overlay exclude_graphdriver_btrfs containers_image_openpgp"

VERSION_LDFLAGS=""
if [[ -n "${3}" ]]; then
  VERSION_LDFLAGS="-X ${PACKAGE_ROOT}/version.Version=${3}"
fi

if [[ -n "${4}" ]]; then
  VERSION_LDFLAGS="$VERSION_LDFLAGS -X ${PACKAGE_ROOT}/version.GitCommitSHA=${4}"
fi

GOOS=$TARGET_GOOS go build -a -tags="${BUILDTAGS}" -ldflags "-s ${VERSION_LDFLAGS}" -o $1/$2 ./img2lambda/cli

go test -v -tags="${BUILDTAGS}" -timeout 30s -short -cover $(go list ./img2lambda/... | grep -v /vendor/ | grep -v /internal/)

cd $1
md5sum $2 > $2.md5
sha256sum $2 > $2.sha256
