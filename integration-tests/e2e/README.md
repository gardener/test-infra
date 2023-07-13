# E2E Test Runner

The e2e test runner leverages kubetest to execute e2e tests and has a few additional features:

- Define description files consisting of e2e testcases to run
- Annotate testcases to run only for dedicated cloud providers
- Evaluate test results of the kubetest run and provide elastic search documents

## Usage

### Run conformance tests via Gardener's test automation
`cd` into your local `gardener/test-infra` folder and target some cluster.  Adjust values for argument `k8sVersion` to match your new cluster's version.

```bash
# first set KUBECONFIG to your cluster
docker run -ti -e --rm -v $KUBECONFIG:/mye2e/shoot.config -v $PWD:/go/src/github.com/gardener/test-infra -e E2E_EXPORT_PATH=/tmp/export -e KUBECONFIG=/mye2e/shoot.config --network=host --workdir /go/src/github.com/gardener/test-infra golang:1.20 bash

# run command below within container to invoke tests in a parallelized way (keep --cloudprovider=skeleton, it means that the tests won't utilize any cloud provider specifics but only resort to kube-apiserver access to the cluster, most likely this is anyway not relevant for the conformance tests, but only for other e2e tests)
GINKGO_PARALLEL=true go run -mod=vendor ./integration-tests/e2e --k8sVersion=1.27.1 --cloudprovider=skeleton --testcasegroup="conformance"
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

| Env Var  | Cmd Line | Default | Description  |
|---|---|---|---|
| K8S_VERSION | k8sVersion |  | **[Required]** Kubernetes cluster version |
| TESTCASE_GROUPS | testcasegroup |  | **[Required]** testcases groups to run (comma separated). E.g. `fast,slow` |
| CLOUDPROVIDER | cloudprovider |  | **[Required]** Cloud provider (supported: aws, gcp, azure, alicloud, openstack) |
| DESCRIPTION_FILE |  | working.json | Path to description json file, which lists the testcases to run |
| E2E_EXPORT_PATH  |  | /tmp/e2e/export  | Location of `shoot.config` file and test results |
| GINKGO_PARALLEL |  | true | Whether to run kubetest in parallel way. Testcases that consist of the `[Serial] tag are executed serially. |
| IGNORE_FALSE_POSITIVE_LIST |  | false | Ignores exclusion of testcases that are listed in `false_positive.json` |
| IGNORE_SKIP_LIST |  | false | Ignores exclusion of testcases that are listed in `skip.json`  |
| INCLUDE_UNTRACKED_TESTS |  | false | Executes testcases that are not mentioned in description files for given provider and kubernetes release version |
| FLAKE_ATTEMPTS | flakeAttempts | 2 | Flake attempts for kubetest: how many time a failed test should be rerun |
| PUBLISH_RESULTS_TO_TESTGRID |  | false | Whether to push test results to google cloud storage, for testgrid |
| RETEST_FLAGGED_ONLY |  | false | Runs testcases with retest flag only. Value of `DESCRIPTION_FILE` is ignored |
| E2E_KUBECONFIG_PATH | kubeconfig | $E2E_EXPORT_PATH/shoot.config | File path of kubeconfig file |
| - | debug | false | Runs application in debug mode |
| - | testcase |  | List of explicit testcases to test. If used, `TESTCASE_GROUPS` and `TESTCASE_GROUPS` are ignored.  |
| - | cleanUpAfterwards | false | Removes downloaded or existings kubernetes files to reduce memory footprint. |
| - | dryRun | false | Dry Run mode, get all test cases and save them to a file, then print the filename path. |
### Description Files
Example:
```json
[
  { "testcase": "[k8s.io] Sysctls [NodeFeature:Sysctls] should reject invalid sysctls", "groups": ["slow", "conformance"], "only": ["aws", "gcp"], "retest": ["aws"], "comment": "Some comment"},
  { "testcase": "[k8s.io] Sysctls [NodeFeature:Sysctls] should support sysctls", "groups": ["slow"], "exclude": ["aws"]}
]
```
| Field  | Description  |
|---|---|
| testcase | testcase name. Can be a substring. All testcases that has this as substring will be executed |
| groups | assigns the testcase to testcase groups |
| only | will consider the testcase only for given cloud provider |
| exclude | will not consider the tetscase for given cloud provider |
| comment | is not evaluated in any way in code. Use only for additional information |
| retest | testcase will be excluded from all general test runs for given providers. Testcases with retest flag can be executed by setting `RETEST_ONLY=true`  |

Existing description files:
- `working.json` consists of all working e2e testcases separated in different groups
- `skip.json` consists of testcases that are always skipped by kubetest due to reasons like: driver not supported, requires >1 nodes, etc.
- `false_positive.json` consists of testcases that are failing because of different reasons like bad code, which makes sense to test with next kubernetes release version

### Output
You find the kubetest dump results (like e2e.log and junit_*.xml files) in the `/tmp/e2e/artifacts` directory. These artifacts are evaluated and stored as *.json files in the `EXPORT_PATH` directory.

<!-- @import "[TOC]" {cmd="toc" depthFrom=1 depthTo=6 orderedList=false} -->
