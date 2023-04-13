#!/usr/bin/env bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
set -e

if [[ -z "${REPO_ROOT}" ]]; then
  export REPO_ROOT="$(readlink -f "$(dirname ${0})/..")"
else
  export REPO_ROOT="$(readlink -f "${REPO_ROOT}")"
fi

DOCKERFILE_DIR="$REPO_ROOT/tmp/build-defs-dockerfiles"
mkdir -p $DOCKERFILE_DIR
BASE_IMAGE="golang:1.20"

# build integration-tests/e2e
cat << __EOF > $DOCKERFILE_DIR/e2e
FROM $BASE_IMAGE
WORKDIR /go/src/github.com/gardener/test-infra
COPY . .

RUN go install -mod=vendor ./integration-tests/e2e
__EOF

docker build -t tm-test-e2e-inst:latest -f $DOCKERFILE_DIR/e2e $REPO_ROOT