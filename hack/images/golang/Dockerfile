FROM golang:1.12.9 AS ginkgo

RUN  \
  apt-get update \
  && apt-get install rsync -y \
  && go get -u github.com/onsi/ginkgo/ginkgo

COPY ./setup /tm/setup