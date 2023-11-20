FROM golang:1.21 AS ginkgo

RUN  \
  apt-get update \
  && apt-get install rsync -y \
  && go install github.com/onsi/ginkgo/ginkgo@latest

COPY ./setup /tm/setup