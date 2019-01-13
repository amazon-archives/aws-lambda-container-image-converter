FROM golang:1.11 AS builder

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/awslabs/img2lambda

COPY . ./
RUN make

FROM busybox:glibc
COPY --from=builder /go/src/github.com/awslabs/img2lambda/bin/local/img2lambda /bin/img2lambda
CMD [ "/bin/img2lambda" ]
