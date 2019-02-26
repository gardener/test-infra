# Getting started

- [Getting started](#getting-started)
  - [Create a Test](#create-a-test)
    - [Input Contract](#input-contract)
    - [Export Contract](#export-contract)
    - [Images](#images)
      - [Base image (eu.gcr.io/gardener-project/gardener/testmachinery/base-step:0.28.0)](#base-image-eugcriogardener-projectgardenertestmachinerybase-step0280)
      - [Golang image (eu.gcr.io/gardener-project/gardener/testmachinery/golang:0.28.0)](#golang-image-eugcriogardener-projectgardenertestmachinerygolang0280)
  - [Create a Testrun](#create-a-testrun)

## Create a Test

A prerequisites for a test to be executed by the TestMachinery is to be available as a GitHub repository.
The repository does not necessarily need to be public.
But if it is private you need to grant the TestMachinery read access to this repository.

Tests in the TestMachinery are configuration files called TestDefinitions which are located in the `repoRoot/.test-defs` folder.
The TestMachinery automatically searches for TestDefinitions files in all specified testlocations (see [testrun description](#create-a-testrun) for more information).
A TestDefinition consists of a name and a description of how the test should be executed by the TestMachinery.
Therefore, a basic test consists of a command which will be executed inside the specified container image.

```yaml
kind: TestDefinition
metadata:
  name: TestDefName
spec:
  owner: gardener@example.com # test owner and contact person in case of a test failure
  recipientsOnFailure: developer1@example.com, developer2@example.com # optional, list of emails to be notified if a step fails
  description: test # optional; description of the test.

  activeDeadlineSeconds: 600 # optional; maximum seconds to wait for the test to finish.
  labels: ["default"] # optional;

  # optional, specify specific behavior of a test.
  # By default steps are executed in parallel.
  # By specifying "serial behavior", tests can be forced to be executed in serial.
  behavior: ["serial"]

  # required; Entrypoint array. Not executed within a shell.
  # The docker image's ENTRYPOINT is used if this is not provided.
  command: [bash, -c]
  # Arguments to the entrypoint. The docker image's CMD is used if this is not provided.
  args: ["test.sh"]

  image: golang:1.11.2 # optional, default image is "eu.gcr.io/gardener-project/gardener/testmachinery/base-step:0.27.0"

  # optional; Configuration of a TestDefinition.
  # Environment Variables can be configured per TestDefinition
  # by specifying the varibale's name and a value, secret or configmap.
  config:
  - type: env
    name: TESTENV1
    value: "Env content"
  - type: env
    name: TESTENV2
    valueFrom:
      secretKeyRef:
        name: secretName
        key: secretKey
```
> Note that the working directory is set to the root of your repository.

### Input Contract

| Environment Variable Name        | Description           |
| ------------- |-------------|
| TM_KUBECONFIG_PATH      | points to a directory containing all kubeconfig files (defaults to `/tmp/env/kubeconfig`). </br> The files contained in this dir depend on the concrete TestRun and can contain up to 3 files: <ul><li>_gardener.config_: the kubeconfig pointing to the gardener cluster created by TM (or predefined/given by a TestRun)</li><li>_seed.config_: the kubeconfig pointing to the seed cluster configured by TM (or predefined/given by a TestRun)</li><li>_shoot.config_: the kubeconfig pointing to the shoot cluster created by TM (or predefined/given by a TestRun)</li></ul>|
| TM_EXPORT_PATH | points to a directory where the test can place arbitrary test-run data which will be archived at the end. Useful if some postprocessing needs to be done on that data. Further information can be found [here](#export-contract) |


### Export Contract

Some installations of the TestMachinery contain a connection to an elasticsearch installation for persistence and evaluation of test results.

The TestMachinery writes some metadata into elasticsearch upon each TestRun completion. It conconsists of the following attributes:
```
tm_meta.landscape: Gardener Landscape of th shoot, e.g. dev, staging,... .
tm_meta.cloudprovider: Cloudprovider of the shoot, e.g. gcp, aws, azure or openstack.
tm_meta.kubernetes_version: Kubernetes version of the shoot.
tm_meta.testrun_id: ID of the overall testrun.
```

TestDefinition can additionally place json files in `TM_EXPORT_PATH` to have them picked up by the TestMachinery and forwarded into elasticsearch.

Such additional data (written by a single test) has to be in one the 3 formats below.
The TestMachinery automatically uploads these documents to an index named like the TestDefinition. A TestDefinition called `CreateShoot` will be uploaded to the index `createshoot`.

- Valid JSON document
- Newline-delimited JSON (multiple json documents in one file, separated by newlines)
  ```
    { "key": "value" }
    { "key2": true }
  ```
- ElasticSearch bulk format with a specific index (see https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html)
  - The documents are then uploaded to the specified index prefixed by `tm-`.
  ```
    { "index": { "_index": "mySpecificIndex", "_type": "_doc" } }
    { "key": "value" }
    { "index": { "_index": "mySpecificIndex", "_type": "_doc" } }
    { "key2": true }
    { "index": { "_index": "mySecondSpecificIndex", "_type": "_doc" } }
    { "key3": 5 }
  ```

### Images

The Testmachinery provides some images to run your Integration Tests (Dockerfiles can be found in hack/images).

#### Base image (eu.gcr.io/gardener-project/gardener/testmachinery/base-step:0.28.0)
- Kubectl
- Helm
- coreutils
- python3
- [cc-utils](https://github.com/gardener/cc-utils) at `/cc/utils` and cli.py added to $PATH
- SAP Root CA

#### Golang image (eu.gcr.io/gardener-project/gardener/testmachinery/golang:0.28.0)
- FROM base image
- Golang v1.11.5
- ginkgo test suite at $GOPATH/src/github.com/onsi/ginkgo
- Go project setup script
  - automatically setup test repository at the provided gopath and cd's into it
  - RUN ```/tm/setup github.com/org repo``` e.g. ``` /tm/setup github.com/gardener/test-infra ```

## Create a Testrun
Before a TestDefinition is executed by the TestMachinery, it must be added to a Testrun.
The Testrun can then be deployed to the Kubernetes cluster running a TestMachinery.
It is picked up and the Testflow with the TestDefinitions is executed.

```yaml
apiVersion: testmachinery.sapcloud.io/v1beta1
kind: Testrun
metadata:
  generateName: integration-
  namespace: default
spec:
  ttlSecondsAfterFinished: # optional; define when the Testrun should cleanup itself.

  # TestLocations define where to search for TestDefinitions.
  testLocations:
  - type: git
    repo: https://github.com/gardener/test-infra.git
    revision: master

  # Specific kubeconfigs can be defined for the garden, seed or shoot cluster.
  # These kubeconfigs are available in "TM_KUBECONFIG_PATH/xxx.config" inside every TestDefinition container.
  # Serial steps can also generate new or update current kubeconfigs in this path.
  # Usefull for testing a TestDefinition with a specific shoot.
  kubeconfigs:
    gardener: # base64 encoded gardener cluster kubeconfig
    seed: # base64 encoded seed cluster kubeconfig
    shoot: # base64 encoded shoot cluster kubeconfig


  # Global config available to every test task in all phases (testFlow and onExit)
  config:
    - name: PROJECT_NAMESPACE
      type: env
      value: "garden-it"
    - name: SHOOT_NAME
      type: env
      value: "xxx"

  # The testFlow describes the execution order of the Testrun.
  # It defines which TestDefinition (that can be found in the TestLocations)
  # are executed in which specified order.
  # If a label is specified then all TestDefinitions labeled with the specific key are executed.
  testFlow:
  - - name: CreateShoot
  - - label: default
  - - name: DeleteShoot

  # OnExit specifies the same execution flow as the testFlow.
  # This flow is run after the testFlow and every step can specify the condition
  # under which it should run depending on the outcome of the testFlow.
  onExit:
  - - name: DeleteShoot
      condition: error|success|always # optional; default is always;
```
