# E2E Test Runner

The e2e test runner leverages kubetest to execute e2e tests and has a few additional features:

- Define description files consisting of e2e testcases to run
- Annotate testcases to run only for dedicated cloud providers
- Evaluate test results of the kubetest run and provide elastic search documents

## Usage

### Example usage (Conformance tests)
Set the `KUBECONFIG` as path to the kubeconfig file of your newly created cluster (you can find the kubeconfig e.g. in the Gardener dashboard). Follow the instructions below to run the Kubernetes e2e conformance tests. Adjust values for arguments `k8sVersion` and `cloudprovider` respective to your new cluster.

```bash
#first set KUBECONFIG to your cluster
docker run -ti -e --rm -v $KUBECONFIG:/mye2e/shoot.config golang:1.13 bash

# run all commands below within container
go get github.com/gardener/test-infra; cd /go/src/github.com/gardener/test-infra
export GO111MODULE=on; export E2E_EXPORT_PATH=/tmp/export; export KUBECONFIG=/mye2e/shoot.config; export GINKGO_PARALLEL=false
go run -mod=vendor ./integration-tests/e2e --k8sVersion=1.16.2 --cloudprovider=azure --testcasegroup="conformance"
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
