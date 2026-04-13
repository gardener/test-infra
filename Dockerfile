# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#############      builder       #############
FROM golang:1.26 AS builder

WORKDIR /go/src/github.com/gardener/test-infra

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY . .

RUN make install

############# tm-controller #############
FROM alpine:3.23 AS tm-controller

COPY charts /charts
COPY --from=builder /go/bin/testmachinery-controller /testmachinery-controller

WORKDIR /

ENTRYPOINT ["/testmachinery-controller"]

############# tm-base-step #############
FROM golang:1.26-alpine AS base-step

RUN  \
  apk add --update --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
  && curl https://aia.pki.co.sap.com/aia/SAP%20Global%20Root%20CA.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Root_CA.crt \
  && curl https://aia.pki.co.sap.com/aia/SAPNetCA_G2_2.crt -o \
      /usr/local/share/ca-certificates/SAPNetCA_G2_2.crt \
  && update-ca-certificates

############# tm-run #############
FROM base-step AS tm-run

COPY --from=builder /go/bin/testrunner /testrunner

WORKDIR /

ENTRYPOINT ["/testrunner"]

############# tm-bot #############
FROM alpine:3.23 AS tm-bot

RUN apk add --update --no-cache bash curl

COPY ./pkg/tm-bot/ui/static /app/static
COPY ./pkg/tm-bot/ui/templates /app/templates
COPY --from=builder /go/bin/tm-bot /tm-bot

WORKDIR /

ENTRYPOINT ["/tm-bot"]

############# tm-prepare #############
FROM base-step AS tm-prepare

COPY --from=builder /go/bin/prepare /tm/prepare

CMD [ "/tm/prepare" ]
