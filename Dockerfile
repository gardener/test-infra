# Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#############      builder       #############
FROM golang:1.19 AS builder

WORKDIR /go/src/github.com/gardener/test-infra
COPY . .

RUN make install

############# tm-controller #############
FROM alpine:3.17 AS tm-controller

COPY charts /charts
COPY --from=builder /go/bin/testmachinery-controller /testmachinery-controller

WORKDIR /

ENTRYPOINT ["/testmachinery-controller"]

############# tm-base-step #############
FROM golang:1.19-alpine AS base-step

ENV HELM_TILLER_VERSION=v2.16.12
ENV KUBECTL_VERSION=v1.26.3
ENV HELM_V3_VERSION=v3.11.3

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
    linux-headers \
  && pip install google-crc32c \
  && pip install --upgrade pip \
    "gardener-cicd-cli>=1.1437.0" \
    "gardener-cicd-libs>=1.1437.0" \
    awscli \
  && mkdir -p /cc/utils && ln -s /usr/bin/cli.py /cc/utils/cli.py \
  && curl -Lo /bin/kubectl \
    https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x /bin/kubectl \
  && curl -L \
    https://get.helm.sh/helm-${HELM_TILLER_VERSION}-linux-amd64.tar.gz \
    | tar xz -C /bin --strip=1 \
  && chmod +x /bin/helm \
  && curl -L \
    https://get.helm.sh/helm-${HELM_V3_VERSION}-linux-amd64.tar.gz | tar xz -C /tmp --strip=1 \
  && mv /tmp/helm /bin/helm3 \
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
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2002.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_02.crt \
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2004.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_04.crt \
  && curl http://aia.pki.co.sap.com/aia/SAP%20Global%20Sub%20CA%2005.crt -o \
      /usr/local/share/ca-certificates/SAP_Global_Sub_CA_05.crt \
  && update-ca-certificates

ENV PATH /cc/utils/bin:$PATH

############# tm-run #############
FROM base-step AS tm-run

COPY --from=builder /go/bin/testrunner /testrunner

WORKDIR /

ENTRYPOINT ["/testrunner"]

############# tm-bot #############
FROM alpine:3.17 AS tm-bot

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