# Conformance Test Runner

Run K8s conformance tests using [Hydrophone](https://github.com/kubernetes-sigs/hydrophone).

## Usage

### Run conformance tests via Gardener's test automation
`cd` into your local `gardener/test-infra` folder and target some cluster.  Adjust values for argument `k8sVersion` to match your new cluster's version.

```bash
# first set KUBECONFIG to your cluster
docker run -ti -e --rm -v $KUBECONFIG:/mye2e/shoot.config -v $PWD:/go/src/github.com/gardener/test-infra -e E2E_EXPORT_PATH=/tmp/export -e KUBECONFIG=/mye2e/shoot.config --network=host --workdir /go/src/github.com/gardener/test-infra  --platform linux/amd64 golang:1.23 bash

# run command below within container to invoke tests in a parallelized way (keep --cloudprovider=dummy, it means that the tests won't utilize any cloud provider specifics)
go run conformance-tests --k8sVersion=1.30.4 --gardenKubeconfig=$KUBECONFIG --cloudprovider=dummy --flakeAttempts=5

# run the command below to invoke tests in a serial way and without any flakes
GINKGO_PARALLEL=false go run conformance-tests --k8sVersion=1.30.4 --gardenKubeconfig=$KUBECONFIG --cloudprovider=dummy

# use the dry-run flag in combination with the hydrophone log level to see what tests to execute
go run conformance-tests --k8sVersion=1.30.4 --gardenKubeconfig=$KUBECONFIG --cloudprovider=dummy --dryRun
```

### Run conformance tests (or single tests) directly (without Gardener's test automation)
```shell
# target some cluster
# cd into your k/k folder
# (if not done already: go install github.com/onsi/ginkgo/v2/ginkgo)
# all conformance tests:
ginkgo --focus "\[Conformance\]" -p ./test/e2e
# single test:
ginkgo --focus "should detect duplicates in a CR when preserving unknown fields" -p ./test/e2e
``````

### Run conformance tests against new K8s versions
To test whether the conformance testing machinery will work with a new, not-yet-Gardener-supported K8s version, you can create a kind cluster and invoke the machinery against it.

Create a kind cluster with two worker nodes and the desired version (see https://github.com/kubernetes-sigs/kind/releases for image links to other K8s versions)
```bash
cat > kind.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

kind create cluster  --config kind.yaml --image kindest/node:v1.27.1@sha256:b7d12ed662b873bd8510879c1846e87c7e676a79fefc93e17b2a52989d3ff42b
```

Invoke the steps from [Run conformance tests via Gardener's test automation](#run-conformance-tests-via-gardeners-test-automation) and ensure the output has executed some > 300 tests, similar to
```shell
INFO[2133] test suite summary: {ExecutedTestcases:371 SuccessfulTestcases:371 FailedTestcases:0...
```

Tear down the kind cluster with
```shell
kind delete cluster
```

### Environment Prerequisites:

- Go installed
- Git installed
- (only for publishing results) environment variable `GCLOUD_ACCOUNT_SECRET` should point to a google cloud storage secret file

### Parameters:

| Env Var                     | Cmd Line | Default                       | Description                                                                                               |
|-----------------------------|---|-------------------------------|-----------------------------------------------------------------------------------------------------------|
| K8S_VERSION                 | k8sVersion |                               | **[Required]** Kubernetes cluster version                                                                 |
| HYDROPHONE_VERSION          | hydrophoneVersion | latest                        | Hydrophone version used for testing                                                                       |
| -                           | conformanceLogLevel | 2                             | Log level passed to hydrophone                                                                            |
| CLOUDPROVIDER               | cloudprovider |                               | **[Required]** Cloud provider (supported: aws, gcp, azure, alicloud, openstack) for uploading the results |
| E2E_EXPORT_PATH             |  | /tmp/e2e/export               | Location to store test results                                                                            |
| GINKGO_PARALLEL             |  | true                          | Runs the tests in parallel with 8 ginkgo nodes.                                                           |
| FLAKE_ATTEMPTS              | flakeAttempts | 1                             | Flake attempts define how many times a failed test should be rerun                                        |
| PUBLISH_RESULTS_TO_TESTGRID |  | false                         | Whether to push test results to google cloud storage, for testgrid                                        |
| E2E_KUBECONFIG_PATH         | kubeconfig | $E2E_EXPORT_PATH/shoot.config | File path of kubeconfig file                                                                              |
| SKIP_INDIVIDUAL_TEST_CASES  | skipIndividualTestCases |                               | A list of ginkgo.skip patterns (regex based) to skip individual test cases. Use "\|" as delimiter.        |
| -                           | dryRun | false                         | Dry Run mode, get all test cases and save them to a file, then print the filename path.                   |

### Output
You find the results (like e2e.log and junit_*.xml files) in the `/tmp/e2e/artifacts` directory. These artifacts are evaluated and stored in the `EXPORT_PATH` directory.

<!-- @import "[TOC]" {cmd="toc" depthFrom=1 depthTo=6 orderedList=false} -->
