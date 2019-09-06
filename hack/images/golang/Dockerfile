FROM golang:1.13.0 AS ginkgo

RUN  \
  apt-get update \
  && apt-get install rsync -y \
  && go get -u github.com/onsi/ginkgo/ginkgo

COPY ./setup /tm/setup