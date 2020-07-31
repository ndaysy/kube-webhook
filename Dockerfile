FROM golang:1.15rc1-alpine3.12 AS build

WORKDIR /go/src/github.com/ndaysy/kube-webhook

COPY go.mod /go/src/github.com/ndaysy/kube-webhook
COPY go.sum /go/src/github.com/ndaysy/kube-webhook
COPY src    /go/src/github.com/ndaysy/kube-webhook/src

ENV CGO_ENABLED 0

RUN export GO111MODULE=on && \
    export GOPROXY=https://goproxy.io && \  
    go build -o /_build/kube-webhook src/main.go

FROM alpine:3.9

LABEL maintainer="ikube"

# copy the go binaries from the building stage
COPY --from=build /_build/kube-webhook /go/bin/
#RUN chmod 777 /go/bin/kube-webhook

EXPOSE 443

ENTRYPOINT ["/go/bin/kube-webhook"]
