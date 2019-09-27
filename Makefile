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
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(shell dirname $(mkfile_path))
current_sha := $(shell GIT_DIR=${current_dir}/.git git rev-parse @)

REGISTRY            := eu.gcr.io/gardener-project/gardener/testmachinery

TM_CONTROLLER_IMAGE := $(REGISTRY)/testmachinery-controller
VERSION             := $(shell cat VERSION)
IMAGE_TAG           := ${VERSION}

TM_RUN_IMAGE := $(REGISTRY)/testmachinery-run
TM_BOT_IMAGE := $(REGISTRY)/bot
PREPARESTEP_IMAGE := $(REGISTRY)/testmachinery-prepare

NS ?= default
KUBECONFIG ?= "~/.kube/config"
TESTRUN ?= "examples/int-testrun.yaml"
LD_FLAGS := $(shell ./hack/get-build-ld-flags)


################################
# Prerequisistes, Installation #
################################

.PHONY: install
install: create-ns install-prerequisites

.PHONY: clean
clean: remove-prerequisites delete-ns

.PHONY: install-controller
install-controller:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml --set "controller.tag=${VERSION}" ./charts/testmachinery | kubectl create -f -

.PHONY: remove-controller
remove-controller:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml ./charts/testmachinery | kubectl delete -f -

.PHONY: install-prerequisites
install-prerequisites:
	helm template --namespace ${NS} ./charts/bootstrap_tm_prerequisites --set="argo.containerRuntimeExecutor=pns" | kubectl apply -f -

.PHONY: remove-prerequisites
remove-prerequisites:
	helm template --namespace ${NS} ./charts/bootstrap_tm_prerequisites | kubectl delete -f -

.PHONY: gen-certs
gen-certs:
	rm -rf assets
	@mkdir -p assets
	@openssl genrsa -out assets/ca.key 2048
	@openssl req -config charts/testmachinery/ca.cnf -new -key assets/ca.key -out assets/ca.csr \
		-subj "/C=DE/O=SAP SE/OU=testmachinery"
	@openssl x509 -req -sha256 -days 365 -in assets/ca.csr -signkey assets/ca.key -out assets/ca.crt
	@openssl genrsa -out assets/tls.key 2048
	@openssl req -config charts/testmachinery/controller.cnf  -new -key assets/tls.key -out assets/tls.csr \
		-subj "/C=DE/O=SAP SE/OU=testmachinery/CN=testmachinery-controller.default.svc"
	@openssl x509 -req -sha256 -days 365 -in assets/tls.csr -CA assets/ca.crt -CAkey assets/ca.key -CAcreateserial -out assets/tls.crt

.PHONY: create-ns
create-ns:
	@ if [ ${NS} != "default" ] && [ ! kubectl get ns ${NS} &> /dev/null ]; then \
		kubectl create ns ${NS}; \
	fi

.PHONY: delete-ns
delete-ns:
	@ if [ ${NS} != "default" ] && [ kubectl get ns ${NS} ]; then \
		kubectl delete ns ${NS}; \
	fi

#####################
# Local development #
#####################
.PHONY: mount-local
mount-local:
	@echo "$(realpath ${path})"
	@minikube mount "$(realpath ${path})":"/tmp/tm"

.PHONY: run-local
run-local:
	@TM_NAMESPACE=${NS} go run cmd/testmachinery-controller/main.go --kubeconfig=${KUBECONFIG} --insecure=true --local=true --dev -v=3

.PHONY: run-controller
run-controller:
	@TM_NAMESPACE=${NS} TESTDEF_PATH=test/.test-defs go run cmd/testmachinery-controller/main.go --kubeconfig=${KUBECONFIG} --insecure=true --dev -v=3

.PHONY: install-controller-local
install-controller-local:
	helm template --namespace ${NS} -f ./charts/testmachinery/local-values.yaml --set "local.enabled=true,local.hostPath=/tmp/tm" \
		./charts/testmachinery | kubectl create -f -

.PHONY: run-it-tests
run-it-tests:
	GIT_COMMIT_SHA=${current_sha} ginkgo ./test/... -v -progress -- \
		--kubeconfig=${KUBECONFIG} --tm-namespace=${NS} --namespace="" --git-commit-sha=master --s3-endpoint=""

.PHONY: code-gen
code-gen:
	@./hack/generate-code

.PHONY: validate
validate:
	@go run cmd/local-validator/main.go -testrun=${TESTRUN}


##################################
# Binary build and docker image  #
##################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy

.PHONY: testrunner
testrunner:
	@go install -v \
		-ldflags "$(LD_FLAGS)" \
		./cmd/testrunner

.PHONY: docker-images
docker-images: docker-image-prepare docker-image-base docker-image-golang docker-image-run docker-image-controller

.PHONY: docker-push
docker-push: docker-push-prepare docker-push-controller


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
	@docker build -t $(PREPARESTEP_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest -f ./hack/images/prepare/Dockerfile ./hack/images/prepare

.PHONY: docker-image-base
docker-image-base:
	@docker build -t $(PREPARESTEP_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest -f ./hack/images/base/Dockerfile ./hack/images/base

.PHONY: docker-image-golang
docker-image-golang:
	@docker build -t $(PREPARESTEP_IMAGE):$(IMAGE_TAG) -t $(PREPARESTEP_IMAGE):latest -f ./hack/images/golang/Dockerfile ./hack/images/golang
