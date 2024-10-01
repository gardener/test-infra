# Conformance Test Runner

Run K8s conformance tests using [Hydrophone](https://github.com/kubernetes-sigs/hydrophone).

## Usage

### Run conformance tests via Gardener's test automation
`cd` into your local `gardener/test-infra` folder and target some cluster.  Adjust values for argument `k8sVersion` to match your new cluster's version.

```bash
# first set KUBECONFIG to your cluster
docker run -ti -e --rm -v $KUBECONFIG:/mye2e/shoot.config -v $PWD:/go/src/github.com/gardener/test-infra -e E2E_EXPORT_PATH=/tmp/export -e KUBECONFIG=/mye2e/shoot.config --network=host --workdir /go/src/github.com/gardener/test-infra  --platform linux/amd64 golang:1.23 bash

# run the command below within the container to invoke tests in a parallel way and allow tests to flake
go run ./conformance-tests --k8sVersion=1.30.4 --flakeAttempts=5

# run the command below to invoke tests in a serial way and without any flakes
GINKGO_PARALLEL=false go run ./conformance-tests --k8sVersion=1.30.4

# use the dry-run flag in combination with the hydrophone log level to see what tests to execute
go run ./conformance-tests --k8sVersion=1.30.4 --dryRun --conformanceLogLevel 4
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

Invoke the steps from [Run conformance tests via Gardener's test automation](#run-conformance-tests-via-gardeners-test-automation) and ensure the output has executed some > 400 tests, similar to
```shell
SUCCESS! -- 400 Passed | 0 Failed | 3 Flaked | 0 Pending | 6799 Skipped
```

Tear down the kind cluster with
```shell
kind delete cluster
```

### Environment Prerequisites:

- Go installed
- Git installed
- (only for publishing results) environment variable `GOOGLE_APPLICATION_CREDENTIALS` should point to valid GCP credentials that give permissions to upload files to the specified bucket.

### Parameters:

| Env Var                        | Cmd Line | Default         | Description                                                                                                         |
|--------------------------------|---|-----------------|---------------------------------------------------------------------------------------------------------------------|
| K8S_VERSION                    | k8sVersion |                 | **[Required]** Kubernetes version of the target cluster                                                             |
| HYDROPHONE_VERSION             | hydrophoneVersion | latest          | Hydrophone version used for testing                                                                                 |
| -                              | conformanceLogLevel | 2               | Log level passed to hydrophone                                                                                      |
| GINKGO_PARALLEL                |  | true            | Runs the tests in parallel with 8 ginkgo nodes.                                                                     |
| -                              | flakeAttempts | 1               | Flake attempts define how many times a failed test should be rerun                                                  |
| SKIP_INDIVIDUAL_TEST_CASES     | skipIndividualTestCases |                 | A list of ginkgo.skip patterns (regex based) to skip individual test cases. Use "\|" as delimiter.                  |
| E2E_EXPORT_PATH                |  | /tmp/e2e/export | Location to store test results                                                                                      |
| E2E_KUBECONFIG_PATH            | kubeconfig | $KUBECONFIG     | File path of kubeconfig file. Reverts to $KUBECONFIG and fails if nothing is to be found there.                     |
| PUBLISH_RESULTS_TO_TESTGRID    |  | false           | Publish test results to google cloud storage for testgrid                                                           |
| CLOUDPROVIDER                  | cloudprovider |                 | Cloud provider (supported: aws, gcp, azure, alicloud, openstack) for uploading the results. Required for publishing |
| GCS_PROJECT_ID                 | | gardener | The GCP project hosting the bucket for uploading test results                                                       |
| GCS_BUCKET                     | | k8s-conformance-gardener | The GCS bucket for uploading test results                                                                           |
| GOOGLE_APPLICATION_CREDENTIALS | | | Path to valid GCP credentials for uploading test results                                                            |
| -                              | dryRun | false           | Dry Run mode                                                                                                        |

### Output
You find the results (like e2e.log and junit_*.xml files) in the `/tmp/e2e/artifacts` directory. These artifacts are evaluated and stored in the `EXPORT_PATH` directory.

<!-- @import "[TOC]" {cmd="toc" depthFrom=1 depthTo=6 orderedList=false} -->
