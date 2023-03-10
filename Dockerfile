#
# minigun.dockerfile
#
# minigun Golang app
#
# @authors Minigun Maintainers
# @copyright 2020 Wayfair, LLC -- All rights reserved.

FROM golang:1.20 as test

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOPATH=/go
ENV PATH="$PATH:$GOPATH/bin"
WORKDIR /go/src/github.com/wayfair-incubator/minigun
COPY . ./
RUN make test

FROM test as build

ENV GOPATH=/go
ENV PATH="$PATH:$GOPATH/bin"
WORKDIR /go/src/github.com/wayfair-incubator/minigun
RUN GOOS=linux CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -ldflags '-extldflags "-static"' -o output/minigun .

FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=build /go/src/github.com/wayfair-incubator/minigun/output/minigun /minigun
ENTRYPOINT ["/minigun"]
