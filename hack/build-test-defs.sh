#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

if [[ -z "${REPO_ROOT}" ]]; then
  export REPO_ROOT="$(readlink -f "$(dirname ${0})/..")"
else
  export REPO_ROOT="$(readlink -f "${REPO_ROOT}")"
fi

DOCKERFILE_DIR="$REPO_ROOT/tmp/build-defs-dockerfiles"
mkdir -p $DOCKERFILE_DIR
BASE_IMAGE="golang:1.21"

# build integration-tests/e2e
cat << __EOF > $DOCKERFILE_DIR/e2e
FROM $BASE_IMAGE
WORKDIR /go/src/github.com/gardener/test-infra

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY . .

RUN go install -mod=mod ./integration-tests/e2e
__EOF

docker build -t tm-test-e2e-inst:latest -f $DOCKERFILE_DIR/e2e $REPO_ROOT