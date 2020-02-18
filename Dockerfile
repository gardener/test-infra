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
FROM golang:1.13.0 AS builder

WORKDIR /go/src/github.com/gardener/test-infra
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -mod=vendor \
   -ldflags "-X github.com/gardener/test-infra/pkg/version.gitVersion=$(cat VERSION) \
               -X github.com/gardener/test-infra/pkg/version.gitTreeState=$([ -z git status --porcelain 2>/dev/null ] && echo clean || echo dirty) \
               -X github.com/gardener/test-infra/pkg/version.gitCommit=$(git rev-parse --verify HEAD) \
               -X github.com/gardener/test-infra/pkg/version.buildDate=$(date --rfc-3339=seconds | sed 's/ /T/')" \
  ./cmd/...

############# tm-controller #############
FROM alpine:3.10 AS tm-controller

RUN apk add --update bash curl

COPY --from=builder /go/bin/testmachinery-controller /testmachinery-controller
COPY ./.env /

WORKDIR /

ENTRYPOINT ["/testmachinery-controller"]

############# telemetry-controller #############
FROM alpine:3.10 AS telemetry-controller

RUN apk add --update bash curl

COPY --from=builder /go/bin/telemetry-controller /telemetry-controller
COPY ./.env /

WORKDIR /

ENTRYPOINT ["/telemetry-controller"]

############# tm-run #############
FROM eu.gcr.io/gardener-project/gardener/testmachinery/base-step:latest AS tm-run

COPY --from=builder /go/bin/testrunner /testrunner

WORKDIR /

ENTRYPOINT ["/testrunner"]

############# tm-bot #############
FROM alpine:3.10 AS tm-bot

RUN apk add --update bash curl

COPY ./pkg/tm-bot/ui/static /app/static
COPY ./pkg/tm-bot/ui/templates /app/templates
COPY --from=builder /go/bin/tm-bot /tm-bot

WORKDIR /

ENTRYPOINT ["/tm-bot"]

############# tm-prepare #############
FROM eu.gcr.io/gardener-project/gardener/testmachinery/base-step:latest AS tm-prepare

COPY --from=builder /go/bin/prepare /tm/prepare

CMD [ "/tm/prepare" ]