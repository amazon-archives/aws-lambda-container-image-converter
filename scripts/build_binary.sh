#!/bin/bash

# Normalize to working directory being build root (up one level from ./scripts)
ROOT=$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )
cd "${ROOT}"

# Builds the ecs-cli binary from source in the specified destination paths.
mkdir -p $1

cd "${ROOT}"

GOOS=$TARGET_GOOS go build -tags="containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_overlay exclude_graphdriver_btrfs containers_image_openpgp" -o $1/img2lambda .
