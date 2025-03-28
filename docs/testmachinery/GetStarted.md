# Getting started

- [Getting started](#getting-started)
  - [Create a Test](#create-a-test)
    - [Input Contract](#input-contract)
      - [Default](#default)
      - [Shoot tests](#shoot-tests)
    - [Export Contract](#export-contract)
    - [Shared Folder](#shared-folder)
    - [Images](#images)
    - [Test](#test)
  - [Create a Testrun](#create-a-testrun)
  - [Configuration](#configuration)
    - [Types](#types)
    - [Sources](#sources)
    - [Location](#location)
    - [Kubeconfigs](#kubeconfigs)

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
  # By specifying "serial" behavior, tests can be forced to be executed in serial.
  # By specifying "disruptive" behavior, tests are executed in serial and are forced to run with continueOnError=false
  behavior: ["serial", "disruptive"]

  # required; Entrypoint array. Not executed within a shell.
  # The docker image's ENTRYPOINT is used if this is not provided.
  command: [bash, -c]
  # Arguments to the entrypoint. The docker image's CMD is used if this is not provided.
  args: ["test.sh"]

  image: golang:1.22 # optional, default image is "europe-docker.pkg.dev/gardener-project/releases/testmachinery/base-step:latest"

  # optional; Configuration of a TestDefinition.
  # Environment Variables can be configured per TestDefinition
  # by specifying the varibale's name and a value, secret or configmap.
  # Files can be mounted into the test by specifying a base64 encoded value, secret or configmap.
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
  - type: file
    name: file1 # name for description
    path: /tmp/tm/file1.txt
    value: "aGVsbG8gd29ybGQK" # base64 encoded file content: "hello world"
  - type: file
    name: file2
    path: /tmp/tm/file2.txt
    valueFrom:
      configMapKeyRef:
        name: configmapName
        key: configmapKey
```
> Note that the working directory is set to the root of your repository.

### Input Contract

#### Default
| Environment Variable Name        | Description           |
| ------------- |-------------|
| TM_KUBECONFIG_PATH      | points to a directory containing all kubeconfig files (defaults to `/tmp/env/kubeconfig`). </br> The files contained in this dir depend on the concrete TestRun and can contain up to 5 files: <ul><li>_testmachinery.config_: the kubeconfig pointing to the testmachinery cluster</li><li>_host.config_: the kubeconfig pointing to the cluster hosting the gardener (not available for untrusted steps)</li><li>_gardener.config_: the kubeconfig pointing to the gardener cluster created by TM (or predefined/given by a TestRun, not available for untrusted steps)</li><li>_seed.config_: the kubeconfig pointing to the seed cluster configured by TM (or predefined/given by a TestRun, not available for untrusted steps)</li><li>_shoot.config_: the kubeconfig pointing to the shoot cluster created by TM (or predefined/given by a TestRun). It expires in 6h.</li></ul>|
| TM_EXPORT_PATH | points to a directory where the test can place arbitrary test-run data which will be archived at the end. Useful if some postprocessing needs to be done on that data. Further information can be found [here](#export-contract) |
| TM_TESTRUN_ID | Name of the testrun |
| TM_GIT_SHA | The commit SHA of the used git location |
| TM_GIT_REF | The ref of the used git location (branch or tag name). Will be empty if only a commit sha is provided (e.g. when testing a Pull Request via the TM bot) |

#### Shoot tests
When your test is running as part of the gardener test suite to test a shoot, there are some more available context variables.

| Environment Variable Name        | Description           |
| ------------- |-------------|
| GARDENER_VERSION      | The current version of the gardener installation. |
| SHOOT_NAME | Name of the shoot to test. |
| PROJECT_NAMESPACE | Project namespace where the current shoot was created. |
| CLOUDPROVIDER | Cloudprovider of the shoot. |
| K8S_VERSION | Kubernetes version of the shoot. |

### Export Contract

Tests are executed by argo. It expects a test to exit with RC=0 in case of success and RC!=0 in case of failures at which point the testrun execution is stopped and considered failed.

> (There is no distinction between a failed test (i.e. asserts are violated) and a test that crashed / exited unexpectedly as in both cases the test subject could contain a regression and a developer needs to asses the reason for RC!=0)

Some installations of the TestMachinery contain a connection to an elasticsearch installation for persistence and evaluation of test results.

The TestMachinery writes some metadata into elasticsearch upon each TestRun completion. It consists of the following attributes:
```
tm_meta.landscape: Gardener Landscape of th shoot, e.g. dev, staging,... .
tm_meta.cloudprovider: Cloudprovider of the shoot, e.g. gcp, aws, azure or openstack.
tm_meta.kubernetes_version: Kubernetes version of the shoot.
tm_meta.testrun_id: ID of the overall testrun.
```

TestDefinition can additionally place json files in `TM_EXPORT_PATH` to have them picked up by the TestMachinery and forwarded into elasticsearch.

Such additional data (written by a single test) has to be in one of the 3 formats below. The TestMachinery automatically derives which of the 3 formats is used.
It then automatically uploads these documents to an index named like the TestDefinition. A TestDefinition called `CreateShoot` will be uploaded to the index `createshoot`.

1. Valid JSON document
2. Newline-delimited JSON (multiple json documents in one file, separated by newlines)
    ```
      { "key": "value" }
      { "key2": true }
    ```
3. ElasticSearch bulk format with a specific index (see https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html).
    The documents are then uploaded to the specified index prefixed by `tm-`.
    ```
      { "index": { "_index": "mySpecificIndex" } }
      { "key": "value" }
      { "index": { "_index": "mySpecificIndex" } }
      { "key2": true }
      { "index": { "_index": "mySecondSpecificIndex" } }
      { "key3": 5 }
    ```

### Shared Folder

Data that is stored in `TM_SHARED_PATH` location, can be accessed from within any testflow step of a the workflow. This is essential if e.g. a test flow step needs to evaluate the output of the previously finished test flow step. This folder is also available as an artifact in the Argo UI.

### Images

The Testmachinery provides some images to run your Integration Tests (Dockerfiles can be found in hack/images).

#### Base image (europe-docker.pkg.dev/gardener-project/releases/testmachinery/base-step:latest) <!-- omit in toc -->
- Kubectl
- Helm
- coreutils
- python3
- [cc-utils](https://github.com/gardener/cc-utils) at `/cc/utils` and cli.py added to $PATH
- SAP Root CA

#### Golang image (europe-docker.pkg.dev/gardener-project/releases/testmachinery/golang:latest) <!-- omit in toc -->
- FROM base image
- Golang v1.22
- ginkgo test suite at $GOPATH/src/github.com/onsi/ginkgo
- Go project setup script
  - automatically setup test repository at the provided gopath and cd's into it
  - RUN ```/tm/setup github.com/org repo``` e.g. ``` /tm/setup github.com/gardener test-infra ```

### Test
A TestDefinition can be easily tested with the Testmachinery with the following steps:

- Upload your Test and TestDefinition e.g. `my-test` to your repo e.g. `github.com/my-repo/it.git`
- Copy the example Testrun for a single test from `/examples/single-testrun.yaml`
- Add your test repository to the `locationSets` of the copied Testrun:
  ```yaml
  spec:
    locationSets:
    - name: default
      default: true
      locations:
      - type: git
        repo: github.com/my-repo/it.git
        revision: master
  ```
- Add your TestDefinition name to `testflow[0]definition.name` of the copied Testrun
  ```yaml
  spec:
    testflow:
    - name: test
      definition:
        name: my-test
  ```
- Add your dependent kubeconfigs to `kubeconfigs` of the Testrun.
  For example if you just want to test an already existing shoot, then just base64 encode the kubeconfig `cat $KUBECONFIG | base64 -w0` and copy the string to `kubeconfigs.shoot`.
  The same procedure applies for other kubeconfigs (but all kubeconfigs are optional).
- Run the Testrun with `kubectl create -f single-testrun.yaml` (Note: your current kubecontext need to point to the cluster where the Testmachinery is installed).

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
  # Deprecated
  # Note: it is only possible to describe testLocations or locationSets.
  testLocations:
  - type: git
    repo: https://github.com/gardener/test-infra.git
    revision: master

  # LocationSets defines multiple TestLocations which can be referenced from steps
  locationSets:
  - name: other
    # optional; defines the default location set which is used if no specific location is defined for a step.
    default: true
    locations:
    - type: git
      repo: https://github.com/gardener/test-infra.git
      revision: 0.20.0

  # Specific kubeconfigs can be defined for the garden, seed or shoot cluster.
  # These kubeconfigs are available in "TM_KUBECONFIG_PATH/xxx.config" inside every TestDefinition container.
  # Serial steps can also generate new or update current kubeconfigs in this path.
  # Usefull for testing a TestDefinition with a specific shoot.
  kubeconfigs:
    gardener: # base64 encoded gardener cluster kubeconfig or ref to secret
    seed: # base64 encoded seed cluster kubeconfig or ref to secret
    shoot: # base64 encoded shoot cluster kubeconfig or ref to secret


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
  testflow:
    - name: create-shoot
      definition:
        name: create-shoot
    - name: test1
      definition:
        label: default
        config: ...
        location: ...
        # Specify if if tests are running trusted content.
        # If the test is not trusted only the minimal configuration is mounted into test.
        # Minimal configuration does currently only content the shoot kubeconfig.
        untrusted: false 
      dependsOn: [ create-shoot ]
      useGlobalArtifacts: true # optional, default get from last "serial" step
      artifactsFrom: create-shoot # optional, default get from last "serial" step

  # OnExit specifies the same execution flow as the testFlow.
  # This flow is run after the testFlow and every step can specify the condition
  # under which it should run depending on the outcome of the testFlow.
  onExit:
  - - name: delete-shoot
      condition: error|success|always # optional; default is always;
```

 ### Locations
 Locations are references to a local directory or a github repository where the TestDefinition reside.
 These 2 location types are used by the TestMachinery to search for all TestDefinitions.

 Git Location:
 ```yaml
type: git
repo: https://github.com/gardener/test-infra.git # http link to the repository
revision: master # tag, commit or branch
 ```
  Local Location (only for local development):
  ```yaml
 type: local
 hostPath: /tmp/tm # hostpath to a directory containing TestDefinition
  ```

 Multiple of these locations can be defined in the testrun in 2 different ways:

 #### TestLocations (Deprecated)
 :warning: Deprecated old way to define locations
 ```yaml
 apiVersion: testmachinery.sapcloud.io/v1beta1
 kind: Testrun
 metadata:
   generateName: integration-
   namespace: default
 spec:
   # TestLocations define where to search for TestDefinitions.
   # Note: it is only possible to describe testLocations or locationSets.
   testLocations:
   - type: git
     repo: https://github.com/gardener/test-infra.git
     revision: master
   - type: git
     repo: https://github.com/gardener/gardener.git
     revision: 0.20.0
```
 #### LocationSets
 Location sets define multiple sets of TestLocations with a unique name.
 These sets can be referenced with their unique name in the testflow steps which enables the following feature in the TestMachinery:
 - Use multiple versions of the same TestDefinition in one Testrun
 - Restrict TestDefinition of steps with labels
  ```yaml
  apiVersion: testmachinery.sapcloud.io/v1beta1
  kind: Testrun
  metadata:
    generateName: integration-
    namespace: default
  spec:
    locationSets:
    - name: first
      default: true
      locations:
      - type: git
        repo: https://github.com/gardener/gardener.git
        revision: 0.21.0
    - name: second
      locations:
      - type: git
        repo: https://github.com/gardener/gardener.git
        revision: 0.20.0
      - type: git
        repo: https://github.com/gardener/test-infra.git
        revision: 0.4.0
 ```

### Flow
A testflow can be specified for the the real tests `spec.testflow` or exit handlers `.spec.onExit`.
The exit handler flow is called when the testflow finished (success or unsuccessful).

A flow is a DAG (Directed Acyclic graph) that can be described by using the `dependsOn` to set a dependency to another step.
The `dependsOn` attribute is a list of step names :warning: do not use `definition.name` as dependency.

```yaml
apiVersion: testmachinery.sapcloud.io/v1beta1
kind: Testrun
metadata:
  generateName: integration-
  namespace: default`

spec:
  testflow:
  - name: create-shoot
    definition:
      name: create-shoot
  - name: test1
    definition:
      label: default
      config: ...
      location: ...
    dependsOn: [ create-shoot ]
    artifactsFrom: create-shoot # optional, default get from last "serial" step
  - name: test2
    definition:
      label: "default,!beta" # matches all testdefintions with label default and not labeled beta
    dependsOn: [ create-shoot ]
    artifactsFrom: # automatically resolved to create-shoot
  - name: delete-shoot
    definition:
      name: delete-shoot
    dependsOn: [ test1, test2 ]
    artifactsFrom: # automatically resolved to create-shoot
```

#### Step Definition
Step definitions define the testdefinitions with their configuration that should be executed within this step.

Test Definitions can be defined by specifying a Testdefinition `name` or `label`.
##### Name
If a definition is specified by a name, then only one TestDefinition with the specified name is searched in the locationSet and executed.

#### Label
If a definition is specified by a labelSelector, then the Test Machinery searches for all TestDefinitions in a locationSet where the specified label Selector is true.
The labelSelector consists of a list of comma separated label names where labels can be included or excluded by adding a `!` at the beginning of the label.

Examples:

TestDefinitions:
```yaml
kind: TestDefinition
metadata:
  name: def1
spec:
  labels: "default_1"
```
```yaml
kind: TestDefinition
metadata:
  name: def2
spec:
  labels: "default_2"
```
```yaml
kind: TestDefinition
metadata:
  name: def3
spec:
  labels: "default_1,default_2"
```

`definition.label: "default_1"`: matches `def1` and `def3` <br>
`definition.label: "default_2"`: matches `def2` and `def3` <br>
`definition.label: "default_1,default_2"`: matches `def3` <br>
`definition.label: "default_1,!default_2"`: matches `def1`

## Configuration

Test can be configured by passing environment variables to the test or mounting files.
The testmachinery offers 2 types of configuration (Environment Variable and File) and 3 value sources (raw value, secret, configmap).

Configuration elements of a step can come from 4 different sources:
1. TestDefinition: Configuration that is directly defined in the testdefinition.
2. Step: Configuration that is defined in a testflow step in the testrun.
3. Shared: Configuration from previous steps that is shared with following steps.
4. Global: Configuration that is defined in the spec of the testrun.

As a configuration element needs to be unique, configurations with the same name are overwritten by the more specific level.
This means that configuration defined by the TestDefinition overwrites anything coming from other configuration sources.

Shared configuration will stack from the root of the dag to it's subtree's. There also the more specific (which is the nearest node) will win.

### Types
Test configuration can be of type "env" and of type "file".

"env" configuration is available as environment variable with the specified `name` to the test.
  ```yaml
  config:
  - type: env
    name: ENV_NAME
  ```

"file" configration is available as mounted file at the specified `path`.
The path to the mounted file is exposed through a Environment Variable with the configname pointing to the file. (e.g `export file="/file/path"`)
  ```yaml
  config:
  - type: file
    name: file # only for description; no effect on the file itself.
    path: /file/path
  ```

### Sources
The value of a configuration type can be defined by 3 different sources

1. *Value*:<br> The value is directly defined in the yaml. :warning: Vale has to be base64 encoded for config type "file"
  ```yaml
  config:
  - type: env | file
    name: config
    value: "Env content" # or base64 encoded content for files
  ```
2. *Secret*:<br> Value from a secret that is available on the cluster.
  ```yaml
  config:
  - type: env | file
    name: config
    valueFrom:
      configMapKeyRef:
        name: configmapName
        key: configmapKey
  ```
3. *ConfigMap*: <br> Value from a configmap that is available on the cluster
  ```yaml
  config:
  - type: env | file
    name: config
    valueFrom:
      configMapKeyRef:
        name: configmapName
        key: configmapKey
  ```

### Location
This configuration can be defined in 3 possible section:

1. *TestDefinition:* <br>Configurations are testdefinition scoped which means that all configuration is only available for these testdefinitions.
  ```yaml
  kind: TestDefinition
  metadata:
    name: TestDefName
  spec:
    config: # Specify configuration here
  ```
2. *Testrun Step:*<br> Configuration will be available to all tests defined in the specific step.
  ```yaml
  apiVersion: testmachinery.sapcloud.io/v1beta1
  kind: Testrun
  metadata:
    generateName: integration-
    namespace: default
  spec:
    testFlow:
     - name: step
       definition:
         label: default
         config: # Specify configuration here
  ```
3. *Testrun Global:*<br> Configuration will be available to all tests.
  ```yaml
  apiVersion: testmachinery.sapcloud.io/v1beta1
  kind: Testrun
  metadata:
    generateName: integration-
    namespace: default
  spec:
    config: # Specify configuration here
  ```

### Kubeconfigs
Specific kubeconfigs can be defined in the testrun and automatically mounted into all teststeps.

Kubeconfigs can be defined by specifying a base64 encoded kubeconfig or a refrerence to a configmap or secret.
Note that if a step is untrusted, only the shoot kubeconfig is mounted.<br>
:warning: The kubeconfigs can be overwritten by serial steps.
  ```yaml
  spec:
    kubeconfigs:
      host: "abc" # base64 encoded kubeconfig
      gardener:  # define kubeconfig from secret
        secretKeyRef:
          name: "mygardenersecret"
          key: "kubeconfig"
      seed: # define kubeconfig from configmap
        configMapKeyRef:
          name: "mygardenerconfigmap"
          key: "kubeconfig"
      shoot: "abc"
  ```

If [OpenID Connect Webhook Authenticator](https://github.com/gardener/oidc-webhook-authenticator) is used to establish trust with another cluster, a kubeconfig may specify the usage of a `tokenFile` instead of a static `token`. 
To make the referenced token available, an additional volume/mount has to be created for each relevant template of the workflow. 
The details for the volume, like `audience` or `expirationSecconds` are read from the `landscapeMappings` as defined in the central testmachinery configuration.

A `landscapeMapping` could look like this:
```yaml
testmachinery:
  landscapeMappings:
    - allowUntrustedUsage: false
      apiServerUrl: https://api.server.com
      audience: dev
      expirationSeconds: 7200
      namespace: default
```

It would create a volume and mount for the specified `tokenFile`, if the kubeconfig's API Server URL matches the address specified by the `landscapeMapping` and the testrun is deployed to namespace `default`.

```yaml
volumes:
- name: token-0
  projected:
    sources:
      - serviceAccountToken:
          audience: dev
          expirationSeconds: 7200
          path: dev-token

...

container:
    volumeMounts:
    - mountPath: /var/run/secrets/gardener/serviceaccount/
      name: token-0
      readOnly: true
```

Unless `allowUntrustedUsage` is set to `true` shoot kubeconfigs are not entitled to make use of a `tokenFile`.