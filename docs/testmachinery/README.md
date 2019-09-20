# Testmachinery <!-- omit in toc -->

- [Concept](#concept)
  - [Goals](#goals)
  - [Test scenarios to be supported](#test-scenarios-to-be-supported)
  - [Design](#design)
- [Developer](#developer)
  - [TestMachinery Deployment](#testmachinery-deployment)
  - [Developing tests locally](#developing-tests-locally)
  - [Use private images](#use-private-images)
  - [Use with GitHub Authentication](#use-with-github-authentication)
  - [Testrunner](#testrunner)
- [Local TestMachinery development](#local-testmachinery-development)

![testmachinery diagram overview](test_machinary_overview.png)


## Concept

The TestMachinery is a k8s controller that watches a k8s cluster for `Testrun`s and executes the tests in the specified order as an argo workflow.

### Goals

- provide a robust test environment that takes care to spin up a gardener cluster together with a first shoot from any gardener commit, execute tests and collect their results
- provide as much convenience as possible to the individual tests: e.g. kubeconfig for the garden, seed and shoot cluster
- allow to specify test dimensions:
  - cloud providers
  - k8s versions
  - test
- don't enforce tests to some programming language, rather specify a contract between TM and actual tests
- export test results for release notes, statistics, etc

### Test scenarios to be supported

The following tests should be supported by TM

- Gardener integration tests
- Conformance tests
- MCM
- etcd backup/restore
- Performance tests
- Security tests
- Stakeholder tests
- See e.g. Gardener K8S upgrade tests in [issue #400](https://github.com/gardener/gardener/issues/400)
- Gardener Lifecycle tests (Upgrade to new gardener version)

### Design

Tests are defined in `TestDefinition.yaml` files, a bunch of tests is executed by defining a `TestRun` CRD. The TestMachinery watches a k8s cluster for `TestRun` CRDs and executes them by translating into an argo workflow.


- **TestDefinition**: Yaml file defining the test as such: [example](../examples/GuestbookTestDef.yaml).

    They are described as a Kubernetes resource but are just used as configuration by the testmachinery.<br>
    If a `TestRun` enumerates tests by a label, all `TestDefinition`s that are found in the given locations and match that label are executed in parallel.
    TestDefinition resources must reside in a folder named `/.test-defs` in the repository of the respective component.
- **TestRun**: CRD to schedule tests and record result: [example](../examples/int-testrun.yaml).

  Can be created by anyone (e.g. CI/CD pipeline or manually using kubectl).
  Could have different states (init, running, finished), references the tests that will be executed as part of this TestRun instance and finally records test durations and results.
  In case of eventual error during execution, exit handlers can be defined to clean up leftover resources.

  `kubectl get testruns` serves as a simple reporting means of currently stored TestRuns.
  More details can be obtained by inspecting the respective argo workflow.
  Of course this needs to be cleaned up eventually, reported somewhere else for longer archiving, etc.

The TestMachinery searches the locations in `testrun.spec.locationSets` for the `TestDefinition` (specified by name and label in the testFlow and onExit), executes them with the provided global and local config and mounts the files of the location to the container where the TestDefinition is found.
Accordingly, TestDefinitions do not need to be deployed to the k8s cluster.
They are automatically picked up and parsed by the testmachinery.

Currently there are 2 location types available:

- `git`: searches a remote git respository for `TestDefinition`s (Note: The k8s cluster hosting the TestMachinery has to run in corporate network if git repositories from the internal github are specified)
- `local`: searches a local file path for TestDefinitions

The TestMachinery parses the `TestRun` definition and generates an argo workflow that executes the test flow.
After the argo workflow has finished (either successful or with failure), the TestMachinery collects the results (phase, duration, ...) and updates the status of the `TestRun`.

Furthermore, generated artifacts that are stored in the s3 storage are deleted after the `TestRun` is deleted from the cluster.


## Developer
### TestMachinery Deployment

1. Setup a k8s cluster (min. Version 1.10.x, preferred Version: 1.12.x, minikube is also suitable)
2. Install prerequisites ([argo](https://github.com/argoproj/argo), [minio](https://www.minio.io/) and the Testrun CRD) with `make install`
    * The default namespace is `default`; another namespace can be defined with `make install NS=namespace-name`
3. Install the TestMachinery with `make install-controller`. Then the controller alongside to a service, validation webhooks and needed rbac permissions is installed.
4. `TestRun`s can be executed by creating them with `kubectl create -f path/to/testrun.yaml` (examples can be found in the [examples folder](examples))

**Prerequisite**: the TestMachinery and the `TestRun`s have to reside in the same namespace due to cross-namespace issues of the argo workflow engine.

### Developing tests locally

`TestRun`s and `TestDefinition`s can be developed locally in a minikube cluster so that no remote installation is needed.
To develop a `TestRun` locally the TestMachinery has to be installed as described in [TestMachinery Deployment](#TestMachinery-Deployment).

If a local `TestDefinition` is developed, the TestMachinery has to be started in _insecure_ mode to mount hostPaths:

- the `TestDefinition` root folder has to be mounted to the minikube cluster with `make mount-local path=path/to/folder`
- the `TestDefinition` itself has to be in the directory `path/to/folder/.test-defs`.
- the TestMachinery itself has to be installed with `make install-controller-local` which starts the controller in _insecure_ mode and mounts the previously specified folder to the controller pod.

### Use private images

Images from private repositories can be used by

1. adding corresponding pull-secrets to the kubernetes cluster (see https://cloud.google.com/container-registry/docs/advanced-authentication and https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
2. add the name of the created secret to the Testmachinery ConfigMap `tm-config` --> `.data.secrets.PullSecrets`

These secrets can be added during runtime as the testmachinery fetches these secrets for every new run.

### Use with GitHub Authentication

The Testmachinery uses no authentication for GitHub by default.
To enable private repositories or increase the rate limit of GitHub; a GitHub config with an user needs to be added for authentication.
Adding a new GitHub config requires the following steps:

1. Create a GitHub config file in the format:
```
secrets:
    - sshUrl: ssh://git@github.com         // change if you want to add configs for different github enterprise instances
      httpUrl: https://github.com
      apiUrl: https://api.github.com
      disable_tls_validation: true
      webhook_token:
      technicalUser:
        username:
        password:
        emailAddress:
        authToken:
    - ...
```
2. Encode the config file in base64 and the encoded data to config.yaml key in `examples/gh-secrets.yaml`.
3. Deploy the secret into the same namespace as the controller.

Another GitHub instance can be added editing the exiting secret and change the base64 encoded data.

### Testrunner
See testrunner [docs](docs/Testrunner.md)

## Local TestMachinery development

For local development of the TestMachinery itself, the prerequisites from [TestMachinery Deployment](#TestMachinery-Deployment) from step 1 to step 3 have to be performed (skip step 4, we don't want to deploy the TestMachinery controller into the cluster as it should run locally).
Afterwards the controller can be started locally with `make run-local KUBECONFIG=/path/to/.kube/config`.
It is then automatically compiled, started in insecure mode and watches for `TestRun`s in the specified cluster.