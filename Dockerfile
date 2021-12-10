#
# minigun.dockerfile
#
# minigun Golang app
#
# @authors Minigun Maintainers
# @copyright 2020 Wayfair, LLC -- All rights reserved.

FROM golang:1.17 as test
ENV GOPATH=/go
ENV PATH="$PATH:$GOPATH/bin"
WORKDIR /go/src/github.com/wayfair-incubator/minigun
COPY . ./
RUN make test

FROM test as build
ENV GOPATH=/go
ENV PATH="$PATH:$GOPATH/bin"
WORKDIR /go/src/github.com/wayfair-incubator/minigun
RUN make build

FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=build /go/src/github.com/wayfair-incubator/minigun/minigun /minigun
ENTRYPOINT ["/minigun"]
