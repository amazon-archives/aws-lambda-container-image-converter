#!/bin/bash

set -e

# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

# Normalize to working directory being build root (up one level from ./scripts)
ROOT=$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )
cd "${ROOT}"

cd "${ROOT}"/example

# Build the example and parse the layers
docker build -t lambda-php .

docker run lambda-php hello '{"name": "World"}'

docker run lambda-php goodbye '{"name": "World"}'

../bin/local/img2lambda -i lambda-php:latest --dry-run

# Look for the 2 layers that contain files in opt/
ls output/layer-1.zip
ls output/layer-2.zip
