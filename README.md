# Test Machinery
<img src="docs/images/gardener-test-machinery.svg" width="150px"></img>

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/test-infra)](https://api.reuse.software/info/github.com/gardener/test-infra)

[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/test-infra)](https://goreportcard.com/report/github.com/gardener/test-infra)

![testmachinery diagram overview](docs/testmachinery/test_machinary_overview.png)


The Test Machinery is the test infrastructure for the gardener project.
Gardener uses the Test Machinery to automatically qualify releases and run every sort of integration test.<br>
These tests include periodically executed shoot integration tests and Kubernetes e2e tests ([e2e runner](integration-tests/e2e)) as well as Gardener Lifecycle tests (validate dev versions of gardener).

Gardener offers managed Kubernetes clusters in different flavors including cloud providers (Alicloud, AWS, GCP, Azure, Openstack), K8s versions, Operating Systems and more.
Therefore, the Test Machinery is designed to test and validate this huge variety of clusters in a scalable and efficient way.

The executed tests are uploaded to the Kubernetes testgrid:
* [Conformance](https://testgrid.k8s.io/conformance-gardener)
* [Additional e2e tests](https://testgrid.k8s.io/gardener-all)

Read detailed docs [here](docs/testmachinery/README.md).</br>

See [here](docs/testmachinery/GetStarted.md) how new tests can be easily added.


## Additional Tools in this Repository

- [**Testrunner**](docs/testrunner)<br>
  Testrunner is an additional component of the Test Machinery that abstracts templating, deploying and watching of Testruns and provide additional functionality like storing test results or notifying test-owners.
- [**Kubernetes e2e Testrunner**](integration-tests/e2e)<br>
  Executes the Kubernetes e2e/Conformance tests and uploads them for further analysis to testgrid or elasticsearch.
- [**Kubernetes CNCF Pull Request Creator**](cmd/tm-bot)<br>
  Prepares a pull request for CNCF kubernetes certification
- [**Host Scheduler**](cmd/hostscheduler)<br>
  The hostscheduler selects available cluster from specific providers and locks the selected cluster so that a fresh gardener can be installed.
  When the cluster is not needed anymore, the host scheduler cleans and releases the cluster.
- [**Shoot Telemetry**](cmd/shoot-telemetry)<br>
  A telemetry controller to get granular insights of Shoot apiserver and etcd availability.
- [**Tests**](docs/tests)<br>
  Testruns that are rendered by the testrunner or github app.
- [**TM GitHub Bot**](cmd/tm-bot)<br>
  A GitHub bot to run tests on PullRequests.
