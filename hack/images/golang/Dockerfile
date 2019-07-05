FROM golang:1.12.1 AS ginkgo

RUN  \
  apk update \
  && apk add \
    rsync \
  && go get -u github.com/onsi/ginkgo/ginkgo

COPY ./setup /tm/setup