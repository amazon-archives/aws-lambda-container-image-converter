# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/awslabs/aws-lambda-container-image-converter

COPY . ./

RUN make install-tools && make

FROM busybox:glibc
COPY --from=builder /go/src/github.com/awslabs/aws-lambda-container-image-converter/bin/local/img2lambda /bin/img2lambda
CMD [ "/bin/img2lambda" ]
