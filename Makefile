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

SHELL = /bin/sh
REPO_ROOT   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(shell dirname $(mkfile_path))
current_sha := $(shell GIT_DIR=${current_dir}/.git git rev-parse @)

REGISTRY            := eu.gcr.io/gardener-project/gardener/testmachinery
HELM_REGISTRY       := eu.gcr.io/gardener-project/charts/gardener/testmachinery

TM_CONTROLLER_IMAGE := $(REGISTRY)/testmachinery-controller
TM_CONTROLLER_CHART := testmachinery-controller
VERSION             ?= $(shell cat ${REPO_ROOT}/VERSION)
IMAGE_TAG           := ${VERSION}

ENVTEST_K8S_VERSION := 1.27.x

TELEMETRY_CONTROLLER_IMAGE := $(REGISTRY)/telemetry-controller
TM_RUN_IMAGE               := $(REGISTRY)/testmachinery-run
TM_BOT_IMAGE               := $(REGISTRY)/bot
PREPARESTEP_IMAGE          := $(REGISTRY)/prepare-step
TM_BASE_IMAGE              := $(REGISTRY)/base
TM_GOLANG_BASE_IMAGE       := $(REGISTRY)/golang

NS ?= default
TESTRUN ?= "examples/int-testrun.yaml"

#########################################
# Tools                                 #
#########################################

TOOLS_DIR := hack/tools
include vendor/github.com/gardener/gardener/hack/tools.mk

#####################
# Utils             #
#####################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/k8s.io/code-generator/generate-internal-groups.sh
	@$(REPO_ROOT)/hack/update-github-templates.sh

.PHONY: code-gen
code-gen:
	@./hack/generate-code

.PHONY: generate
generate: $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(MOCKGEN) $(OPENAPI_GEN)
	@$(REPO_ROOT)/hack/generate.sh ./cmd/... ./pkg/... ./test/...

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg ./test ./integration-tests

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...

.PHONY: test
test:
	KUBEBUILDER_ASSETS="$(shell setup-envtest use -p path ${K8S_VERSION})" go test -mod=vendor $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...


.PHONY: install
install:
	@./hack/install

.PHONY: verify
verify: check

.PHONY: build-test-defs
build-test-defs:
	@./hack/build-test-defs.sh

.PHONY: all
all: generate format verify install

.PHONY: install-requirements
install-requirements:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.1
	@go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/golang/mock/mockgen

.PHONY: gen-certs
gen-certs:
	rm -rf assets
	@mkdir -p assets
	@openssl genrsa -out assets/ca.key 2048
	@openssl req -config charts/testmachinery/ca.cnf -new -key assets/ca.key -out assets/ca.csr \
		-subj "/C=DE/O=SAP SE/OU=testmachinery"
	@openssl x509 -req -sha256 -days 365 -in assets/ca.csr -signkey assets/ca.key -out assets/ca.crt
	@openssl genrsa -out assets/tls.key 2048
	@openssl req -new -sha256 -nodes -extensions v3_req -config charts/testmachinery/controller.cnf -key assets/tls.key -out assets/tls.csr
	@openssl x509 -req -sha256 -days 365 -extensions v3_req -extfile charts/testmachinery/controller.cnf -in assets/tls.csr -CA assets/ca.crt -CAkey assets/ca.key -CAcreateserial -out assets/tls.crt


################################
# Prerequisistes, Installation #
################################

.PHONY: deploy-controller
deploy-controller:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml --set "controller.tag=${VERSION}" ./charts/testmachinery | kubectl apply -f -

.PHONY: remove-controller
remove-controller:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml ./charts/testmachinery | kubectl delete -f -

.PHONY: remove-controller-local
remove-controller-local:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml  --set "testmachinery.local=true,testmachinery.insecure=true,controller.hostPath=/tmp/tm" \
		./charts/testmachinery | kubectl delete -f -

#####################
# Local development #
#####################
.PHONY: mount-local
mount-local:
	@echo "$(realpath ${path})"
	@minikube mount "$(realpath ${path})":"/tmp/tm"

.PHONY: run-local
run-local:
	@TM_NAMESPACE=${NS} go run cmd/testmachinery-controller/main.go --insecure=true --local=true --dev -v=3

.PHONY: run-controller
run-controller:
	@TM_NAMESPACE=${NS} TESTDEF_PATH=test/.test-defs go run cmd/testmachinery-controller/main.go --insecure=true --dev -v=3

.PHONY: install-controller-local
install-controller-local:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml --set "testmachinery.local=true,testmachinery.insecure=true,controller.hostPath=/tmp/tm" \
		./charts/testmachinery | kubectl create -f -

.PHONY: run-it-tests
run-it-tests:
	GIT_COMMIT_SHA=${current_sha} ginkgo ./test/... -v -progress -- \
		--tm-namespace=${NS} --namespace="" --git-commit-sha=master --s3-endpoint=""

.PHONY: validate
validate:
	@go run cmd/local-validator/main.go --testrun=${TESTRUN}


##################################
# Binary build and docker image  #
##################################

.PHONY: testrunner
testrunner:
	@go install -v \
		-ldflags "$(LD_FLAGS)" \
		./cmd/testrunner

.PHONY: docker-images
docker-images: docker-image-prepare docker-image-base docker-image-golang docker-image-run docker-image-controller

.PHONY: docker-push
docker-push: docker-push-prepare docker-push-controller docker-push-run


.PHONY: docker-push-controller
docker-push-controller:
	@docker push $(TM_CONTROLLER_IMAGE):$(IMAGE_TAG)
	@docker push $(TM_CONTROLLER_IMAGE):latest

.PHONY: docker-push-run
docker-push-run:
	@docker push $(TM_RUN_IMAGE):$(IMAGE_TAG)
	@docker push $(TM_RUN_IMAGE):latest

.PHONY: docker-push-prepare
docker-push-prepare:
	@docker push $(PREPARESTEP_IMAGE):$(IMAGE_TAG)
	@docker push $(PREPARESTEP_IMAGE):latest

.PHONY: docker-image-controller
docker-image-controller:
	@docker build -t $(TM_CONTROLLER_IMAGE):$(IMAGE_TAG) -t $(TM_CONTROLLER_IMAGE):latest --target tm-controller .

.PHONY: docker-image-run
docker-image-run:
	@docker build -t $(TM_RUN_IMAGE):$(IMAGE_TAG) -t $(TM_RUN_IMAGE):latest --target tm-run .

.PHONY: docker-image-bot
docker-image-bot:
	@docker build -t $(TM_BOT_IMAGE):$(IMAGE_TAG) -t $(TM_BOT_IMAGE):latest --target tm-bot .

.PHONY: docker-image-prepare
docker-image-prepare:
	@docker build -t $(PREPARESTEP_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest --target tm-prepare .

.PHONY: docker-image-base
docker-image-base:
	@docker build -t $(TM_BASE_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest --target base-step .

.PHONY: docker-image-golang
docker-image-golang:
	@docker build -t $(TM_GOLANG_BASE_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest -f ./hack/images/golang/Dockerfile ./hack/images/golang

##################################
# Helm charts                    #
##################################

.PHONY: build-tm-chart
build-tm-chart:
	@helm package $(REPO_ROOT)/charts/testmachinery --version $(VERSION) -d $(REPO_ROOT)/charts

.PHONY: publish-tm-chart
publish-tm-chart: build-tm-chart
	@helm push $(REPO_ROOT)/charts/$(TM_CONTROLLER_CHART)-$(VERSION).tgz oci://$(HELM_REGISTRY)