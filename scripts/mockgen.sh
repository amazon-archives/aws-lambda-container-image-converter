#!/bin/bash

# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

# This script wraps the mockgen tool and inserts licensing information.

set -e
package=${1?Must provide package}
interfaces=${2?Must provide interface names}
outputfile=${3?Must provide an output file}
PROJECT_VENDOR="github.com/awslabs/aws-lambda-container-image-converter/img2lambda/vendor/"

export PATH="${GOPATH//://bin:}/bin:$PATH"

data=$(
cat << EOF
// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
$(mockgen -package mocks "${package}" "${interfaces}")
EOF
)

mkdir -p $(dirname ${outputfile})

echo "$data" | sed -e "s|${PROJECT_VENDOR}||" | goimports > "${outputfile}"
