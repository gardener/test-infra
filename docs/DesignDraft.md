# Test-Machinery (TM) Design

- [Goals](#goals)
- [Test scenarios to be supported](#test-scenarios-to-be-supported)
- [Design](#design)
  - [TestDefinitions and TestRuns](#testdefinitions-and-testruns)
  - [Test Input](#test-input)
  - [Test Output](#test-output)
  - [Deployment](#deployment)
    - [Cloud](#cloud)
    - [Local](#local)
  - [Responsibilities/Tasks](#responsibilitiestasks)
    - [Boostrap TestDefinitions](#boostrap-testdefinitions)
    - [Ensure cloned git repo is available to argo tasks](#ensure-cloned-git-repo-is-available-to-argo-tasks)
    - [Watch on TestRun CRDs](#watch-on-testrun-crds)
    - [Garbage Collection](#garbage-collection)
- [Open points / issues](#open-points--issues)

# Goals

- provide a robust test environment that takes care to spin up a gardener cluster together with a first shoot from any gardener commit, execute tests and collect their results
- provide as much convenience as possible to the individual tests: e.g. kubeconfig for the garden, seed and shoot cluster
- allow to specify test dimensions:
  - cloud providers
  - k8s versions
  - test
- don't enforce tests to some programming language, rather specify a contract between TM and actual tests
- export test results for release notes, statistics, etc

# Test scenarios to be supported

The following tests should be supported by TM

- Gardener integration tests
- Conformance tests
- MCM
- etcd backup/restore
- Performance tests
- Security tests
- Stakeholder tests
- See e.g. Gardener K8S upgrade tests in [issue #400](https://github.com/gardener/gardener/issues/400)

# Design

Tests are defined in `TestDefinition.yaml` files, a bunch of tests is executed by defining a `TestRun` CRD. The TestMachinery watches a k8s cluster for `TestRun` CRDs and executes them by translating into an argo workflow.

## TestDefinitions and TestRuns

- **TestDefinition**: Yaml file defining the test as such: [example](../examples/GuestbookTestDef.yaml). TestDefinition resources must reside in a folder named `/.test-defs` in the repository of the respective component.
- **TestRun**: CRD to schedule tests and record result: [example](../examples/int-testrun.yaml).

  Can be created by anyone (e.g. CI/CD pipeline or manually using kubectl).
  Could have different states (init, running, finished), references the tests that will be executed as part of this TestRun instance and finally records test durations and results.

  `kubectl get testruns` serves as a simple reporting means of currently stored TestRuns.
  More details can be obtained by inspecting the respective argo workflow.
  Of course this needs to be cleaned up eventually, reported somewhere else for longer archiving, etc.

- **TestProfile**: probably not needed for first phase... Could serve as a higher-level kind of grouping of TestDefs, e.g. "dev" excluding [Slow],[Flaky] tests, "release" adding tests like [Slow],[Conformance] tests. Instead of enumerating every TestDef in a TestRun, it could as well point to such a predefined TestProfile.
  On the other side, test categories could be enough, see e.g. https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-tests.md#kinds-of-tests

## Test Input

TM schedules tests as argo workflows and provides necessary input as environment variables prefixed with `TM_` plus a bunch of files.

- **Configuration provided by TM**:
  - `TM_KUBECONFIG_PATH`: points to a directory containing all kubeconfig files (defaults to `/tmp/env/kubeconfig`). The files contained in this dir depend on the concrete TestRun and can contain up to 3 files:
    - _gardener.config_: the kubeconfig pointing to the gardener cluster created by TM (or predefined/given by a TestRun)
    - _seed.config_: the kubeconfig pointing to the seed cluster configured by TM (or predefined/given by a TestRun)
    - _shoot.config_: the kubeconfig pointing to the shoot cluster created by TM (or predefined/given by a TestRun)
  - not needed `TM_SHARED_CONTEXT_PATH`: points to a directory where a test can place arbitrary files, it will be stashed as an output artifact and can be used by other tests as an input artifact. Read-only for parallel tests. (Usecase: CreateShootTestDef writes the shootname and UpgradeShootTestDef consumes it)
  - `TM_EXPORT_PATH`: points to a directory where the test can place arbitrary test-run data which will be archived at the end. Useful if some postprocessing needs to be done on that data.
- **Configuration provided by a TestRun**:
  - A TestRun can specify additional configuration in `.spec.config`, with either `.spec.config.type` being `env` or `file`.
    - `env` configuration is injected as an environment variable with name `.spec.config.name` and value `.spec.config.value`
    - Not needed in the beginning: `file`: a file located at `TM_TESTRUN_CONFIG`+`.spec.config.name` containing `.spec.config.value`
  - Probably we need some special/more advanced configuration (e.g. pointing to some configmap) instead of only supporting heredocs. E.g. others might need some portion of `landscape.yaml` and a config could look like
      ```
      config:
      - name: landscape-config
          value: ref-to-configmap
          type: configmap
          testDefinition: EPSTestDef
      ```

## Test Output

- Tests are executed by argo. It expects a test to exit with RC=0 in case of success and RC!=0 in case of failures at which point the testrun execution is stopped and considered failed.

  (There is no distinction between a failed test (i.e. asserts are violated) and a test that crashed / exited unexpectedly as in both cases the test subject could contain a regression and a developer needs to asses the reason for RC!=0)
- Not needed for first phase: TM collects the logs from `stdout` and `stderr` and archives them for later processing.
- Not needed for first phase: All files from directory `TM_EXPORT_PATH` are collected and archived for later processing.

## Deployment

### Cloud

TM can be deployed to some cluster where CI/CD will create TestRun CRDs.
Deployment via make scripts, see [main README](../README.md)

### Local

TMC can be run locally in which case it needs a local volume mapped into e.g. the minikube cluster, see [main README](../README.md)

## Responsibilities/Tasks

### Boostrap TestDefinitions

TestRun CRD contains `.spec.testDefLocations` supporting 2 types: _git_ and _local_:

- `git`
  - TM checks the mentioned git repository `.spec.testDefLocations.repo` at revision `.spec.testDefLocations.revision` for test definitions contained in the `/testdefs` folder
  - TM additionally checks if the repo contains a valid CI component descriptor (by using cc-utils) in which case TM will follow dependencies transitively to other components and lookup testdefs in their respective repositories as well
- `local`: for local development of tests; prerequisite is to mount the desired volume with [`minikube mount`](https://github.com/kubernetes/minikube/blob/master/docs/host_folder_mount.md)
  - TM checks the defined folder at `.spec.testDefLocations.filepath` and looks for a `/testdef` folder for contained test definitions (`*.yaml`)

### Ensure cloned git repo is available to argo tasks

Git respositories contain the actual test code, therefore the repo has to be also available in the container of the test tasks.

#### A few possible solutions:

1. Clone all repos in the _prepare_ task. Then generate one artifact and pass the artifact to all following tasks (avoids cloning too often from github, still requires roundtripping to minio).
2. Use argo's `git` artifacts to clone all repos to every task (probably too slow)
3. Use _ReadOnlyMany_ volumes (drawback: probably not available on all clouds, argo bug)

#### Special case for local runs:

- everything is already mounted at `.spec.testDefLocations.filepath`

### Watch on TestRun CRDs

- compute the effective Test list
- translate TestRun & TestDef into an argo DAG workflow:
  - use artifact passing for `TM_KUBECONFIG_PATH` to allow tasks to modify/ammend them (i.e. input `TM_KUBECONFIG_PATH` and output it at task end)
  - serial tasks are able to change artifacts; they are then transferred to inputs of the following tasks
  - parallel tasks are not able to modify artifacts, i.e. potential modifications are not transferred to later tasks
  - inputs of all tasks are the output artifacts of the previous **serial** task (expect for the initial _prepare_ task)
  - submit workflow
- watch workflow progress and update the TestRun resource correspondingly
- tear down test environment
- persist/consolidate/report test results (e.g. so that they can be used on dashboards, Release Notes, whatever post-processing)

### Garbage Collection

- Either add an ownership reference pointing from the workflow to the TestRun CRD or vice versa.
- For cron based TestRuns/workflows: they should be purged automatically after some time [workflow ttl](https://github.com/argoproj/argo/blob/master/examples/gc-ttl.yaml)
- For PR triggered TestRuns/workflows: add some hook to delete the respective TestRun/workflow on PR close
- Minio artifcats: register a test-machinery finalizer on the argo workflow CRD (or the TestRun CRD), watch them for deletion and delete all minio artifacts named in the .status field of the workflow, remove the finalizer and retrigger the workflow deletion

# Open points / issues

- The workflows and argo itself must run in the same namespace for convenience reasons (otherwise we would need to recreate the minio secret in every desired namespace and apply additional role bindings as the workflow sidecar currently cannot locate secrets outside its namespace and we don't want to double/triple the minio secret)
- TestDefs need to define timeouts (map to argo `activeDeadlineSeconds`)
- Should tests be able to influence control flow (i.e. parallel/serial/disruptive)
- If possible (but doesn't look like argo supports this), we want to specify whether a failing task should stop the workflow execution or not
- How to trigger future virtual/nodeless gardener? E.g. by just omitting a `.kubeconfigs.gardener` or better have some `CreateVirtualGardenerTestDef`?
- <a id="minimal-minio"></a>minimal minio installation:
  - `k apply -f minio.yaml`
  - `k apply -f minio-secret.yaml`
  - Patch argo configmap with content from `minio-configchange.yaml` via `k -n argo edit cm workflow-controller-configmap`
- <a id="github_api"></a>GitHub API endpoint to retrieve files (content base64 encoded) of a specific repo path [Content API Doc (only 1000 files)](https://developer.github.com/v3/repos/contents/) or [Tree API Doc](https://developer.github.com/v3/git/trees/#get-a-tree).
- argo and pvcs: https://github.com/kubernetes/kubernetes/issues/67342
