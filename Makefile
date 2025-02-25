# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

SHELL = /bin/sh
ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR    		:= $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
REPO_ROOT   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(shell dirname $(mkfile_path))
current_sha := $(shell GIT_DIR=${current_dir}/.git git rev-parse @)

REGISTRY            := europe-docker.pkg.dev/gardener-project/releases/testmachinery
HELM_REGISTRY       := europe-docker.pkg.dev/gardener-project/releases/charts/gardener/testmachinery

TM_CONTROLLER_IMAGE := $(REGISTRY)/testmachinery-controller
TM_CONTROLLER_CHART := testmachinery-controller
VERSION             ?= $(shell cat ${REPO_ROOT}/VERSION)
IMAGE_TAG           := ${VERSION}

ENVTEST_K8S_VERSION := 1.31.x

TM_RUN_IMAGE               := $(REGISTRY)/testmachinery-run
TM_BOT_IMAGE               := $(REGISTRY)/bot
PREPARESTEP_IMAGE          := $(REGISTRY)/prepare-step
TM_BASE_IMAGE              := $(REGISTRY)/base

NS ?= default
TESTRUN ?= "examples/int-testrun.yaml"

#########################################
# Tools                                 #
#########################################

TOOLS_DIR := hack/tools
include $(GARDENER_HACK_DIR)/tools.mk

#####################
# Utils             #
#####################

.PHONY: tidy
tidy:
	@go mod tidy
	@GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(REPO_ROOT)/hack/update-github-templates.sh

.PHONY: code-gen
code-gen: $(VGOPATH) $(CONTROLLER_GEN)
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash ./hack/generate-code
	$(MAKE) format

.PHONY: generate
generate: $(VGOPATH) $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM)
	@go build -o $(OPENAPI_GEN) k8s.io/kube-openapi/cmd/openapi-gen
	@go build -o $(MOCKGEN) go.uber.org/mock/mockgen
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./cmd/... ./pkg/... ./test/...
	$(MAKE) format

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test ./conformance-tests

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/... ./conformance-tests/...
	$(MAKE) sast-report

.PHONY: test
test:
	KUBEBUILDER_ASSETS="$(shell setup-envtest use -p path ${ENVTEST_K8S_VERSION})" go test -mod=mod ./cmd/... ./pkg/...

.PHONY: sast
sast: $(GOSEC)
	@bash $(GARDENER_HACK_DIR)/sast.sh

.PHONY: sast-report
sast-report: $(GOSEC)
	@bash $(GARDENER_HACK_DIR)/sast.sh --gosec-report true

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
	@go install github.com/golang/mock/mockgen

.PHONY: gen-certs
gen-certs:
	rm -rf assets
	@mkdir -p assets
	@openssl genrsa -out assets/ca.key 2048
	@openssl req -x509 -new -noenc -key assets/ca.key -sha256 -days 365  -out assets/ca.crt -subj "/CN=SA SE CA/C=DE/ST=Walldorf/L=Walldorf/O=testmachinery"
	@openssl genrsa -out assets/tls.key 2048
	@openssl req -new -sha256 -noenc -extensions v3_req -config charts/testmachinery/controller.cnf -key assets/tls.key -out assets/tls.csr
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
