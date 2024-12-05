# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM ghcr.io/open-component-model/ocm/ocm.software/ocmcli/ocmcli-image:0.18.0 AS ocmcli
#############      builder       #############
FROM golang:1.23 AS builder

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
FROM alpine:3.20 AS tm-controller

COPY charts /charts
COPY --from=builder /go/bin/testmachinery-controller /testmachinery-controller

WORKDIR /

ENTRYPOINT ["/testmachinery-controller"]

############# tm-base-step #############
FROM golang:1.23-alpine AS base-step

ENV KUBECTL_VERSION=v1.31.2
ENV HELM_V3_VERSION=v3.16.2

COPY --from=ocmcli /bin/ocm /bin/ocm

RUN  \
  apk update \
  && apk add \
    apache2-utils \
    coreutils \
    cargo \
    bash \
    binutils \
    bind-tools \
    build-base \
    curl \
    file \
    gcc \
    git \
    git-crypt \
    jq \
    libc-dev \
    libev-dev \
    libffi-dev \
    openssh \
    openssl \
    openssl-dev \
    python3 \
    python3-dev \
    py3-pip \
    wget \
    grep \
    findutils \
    rsync \
    bc \
    xz \
    linux-headers \
  && pkgdir=/tmp/packages \
  && ocm_repo="europe-docker.pkg.dev/gardener-project/releases" \
  && cc_utils_version=1.2515.0 \
  && cc_utils_ref="OCIRegistry::${ocm_repo}//github.com/gardener/cc-utils" \
  && mkdir "${pkgdir}" \
  && for resource in gardener-cicd-cli gardener-cicd-libs gardener-oci; do \
    ocm download resources \
      "${cc_utils_ref}:${cc_utils_version}" \
      "${resource}" \
      -O - | tar xJ -C "${pkgdir}"; \
    done \
  && pip install --break-system-packages google-crc32c \
  && pip install --break-system-packages --upgrade --find-links "${pkgdir}" \
    pip \
    "gardener-cicd-cli==${cc_utils_version}" \
    "gardener-cicd-libs==${cc_utils_version}" \
    awscli \
    pytz \
  && rm -rf "${pkgdir}" \
  && mkdir -p /cc/utils && ln -s /usr/bin/cli.py /cc/utils/cli.py \
  && curl -Lo /bin/kubectl \
     https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x /bin/kubectl \
  && curl -L \
    https://get.helm.sh/helm-${HELM_V3_VERSION}-linux-amd64.tar.gz | tar xz -C /tmp --strip=1 \
  && mv /tmp/helm /bin/helm3 \
  && chmod +x /bin/helm3 \
  && ln -s /bin/helm3 /bin/helm \
  && curl -Lo /bin/yaml2json \
    https://github.com/bronze1man/yaml2json/releases/download/v1.2/yaml2json_linux_amd64 \
  && chmod +x /bin/yaml2json \
  && curl -Lo /bin/cfssl \
    https://pkg.cfssl.org/R1.2/cfssl_linux-amd64 \
  && chmod +x /bin/cfssl \
  && curl -Lo /bin/cfssljson \
    https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 \
  && chmod +x /bin/cfssljson \
  &&  curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Root%20CA.crt -o \
    /usr/local/share/ca-certificates/SAP_Global_Root_CA.crt \
  && curl http://aia.pki.co.sap.com/aia/SAPNetCA_G2.crt -o \
      /usr/local/share/ca-certificates/SAPNetCA_G2.crt \
  && curl https://aia.pki.co.sap.com/aia/SAPNetCA_G2_2.crt -o \
    /usr/local/share/ca-certificates/SAPNetCA_G2_2.crt \
  && update-ca-certificates \
  && rm /usr/lib/python3.12/site-packages/certifi/cacert.pem \
  && ln -sf /etc/ssl/certs/ca-certificates.crt "$(python3 -m certifi)"
# SAPNetCA_G2.crt will expire 2025-03-17 -> remove
ENV PATH /cc/utils/bin:$PATH

############# tm-run #############
FROM base-step AS tm-run

COPY --from=builder /go/bin/testrunner /testrunner

WORKDIR /

ENTRYPOINT ["/testrunner"]

############# tm-bot #############
FROM alpine:3.20 AS tm-bot

RUN apk add --update bash curl

COPY ./pkg/tm-bot/ui/static /app/static
COPY ./pkg/tm-bot/ui/templates /app/templates
COPY --from=builder /go/bin/tm-bot /tm-bot

WORKDIR /

ENTRYPOINT ["/tm-bot"]

############# tm-prepare #############
FROM base-step AS tm-prepare

COPY --from=builder /go/bin/prepare /tm/prepare

CMD [ "/tm/prepare" ]
