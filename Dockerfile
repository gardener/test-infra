# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM ghcr.io/open-component-model/ocm/ocm.software/ocmcli/ocmcli-image:0.35.0 AS ocmcli
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

ENV KUBECTL_VERSION v1.35.0
ENV HELM_V3_VERSION v3.20.0

COPY --from=ocmcli /usr/local/bin/ocm /bin/ocm

RUN  \
  apk add --update --no-cache \
    apache2-utils \
    ca-certificates \
    coreutils \
    bash \
    bc \
    binutils \
    bind-tools \
    build-base \
    curl \
    file \
    findutils \
    gcc \
    git \
    git-crypt \
    grep \
    jq \
    yq \
    libc-dev \
    openssl \
    python3 \
    python3-dev \
    py3-pip \
    wget \
    xz \
    linux-headers \
  && mkdir -p $HOME/.config/pip \
  && echo -e "[global]\nbreak-system-packages = true" >> $HOME/.config/pip/pip.conf \
  && pkgdir=/tmp/packages \
  && ocm_repo="europe-docker.pkg.dev/gardener-project/releases" \
  && cc_utils_ref="OCIRegistry::${ocm_repo}//github.com/gardener/cc-utils" \
  && cc_utils_version="$(ocm show versions ${cc_utils_ref} | sort -r | head -1)" \
  && mkdir "${pkgdir}" \
  && for resource in gardener-cicd-cli gardener-cicd-libs gardener-oci gardener-ocm; do \
    ocm download resources \
      "${cc_utils_ref}:${cc_utils_version}" \
      "${resource}" \
      -O - | tar xJ -C "${pkgdir}"; \
    done \
  && CFLAGS='-Wno-int-conversion' \
     pip3 install --upgrade --no-cache-dir --find-links "${pkgdir}" \
      "gardener-cicd-cli==${cc_utils_version}" \
      "gardener-cicd-libs==${cc_utils_version}" \
  && rm -rf "${pkgdir}" \
  && mkdir -p /cc/utils && ln -s /usr/bin/gardener-ci /cc/utils/cli.py \
  && curl -Lo /bin/kubectl \
     https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x /bin/kubectl \
  && curl -L \
    https://get.helm.sh/helm-${HELM_V3_VERSION}-linux-amd64.tar.gz | tar xz -C /tmp --strip=1 \
  && mv /tmp/helm /bin/helm3 \
  && chmod +x /bin/helm3 \
  && ln -s /bin/helm3 /bin/helm \
  && curl https://aia.pki.co.sap.com/aia/SAP%20Global%20Root%20CA.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Root_CA.crt \
  && curl https://aia.pki.co.sap.com/aia/SAPNetCA_G2.crt -o \
      /usr/local/share/ca-certificates/SAPNetCA_G2.crt \
  && curl https://aia.pki.co.sap.com/aia/SAPNetCA_G2_2.crt -o \
      /usr/local/share/ca-certificates/SAPNetCA_G2_2.crt \
  && update-ca-certificates \
  && rm /usr/lib/python3.12/site-packages/certifi/cacert.pem \
  && ln -sf /etc/ssl/certs/ca-certificates.crt "$(python3 -m certifi)"
# SAPNetCA_G2.crt will expire 2025-03-17 -> remove
# TODO: remove after migrating scripts to gardener-ci
ENV PATH /cc/utils:$PATH

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
