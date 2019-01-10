FROM golang:1.11 AS builder

# Install dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR $GOPATH/src/github.com/awslabs/img2lambda

# Ensure deps, optimized for build caching
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only

# Copy and build the code
COPY . ./
RUN go build -o /bin/img2lambda -tags="containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_overlay exclude_graphdriver_btrfs containers_image_openpgp" .

FROM busybox:glibc
COPY --from=builder /bin/img2lambda /bin/img2lambda
RUN /bin/img2lambda --help
CMD [ "/bin/img2lambda" ]
