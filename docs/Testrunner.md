# Testrunner

Testrunner is an additional component of the Test Machinery that abstracts templating, deploying and watching of Testruns and provide additional functionality like storing test results or notifying test-owners.

- [Testrunner](#testrunner)
  - [Usage](#usage)
  - [Pipeline Usage](#pipeline-usage)
    - [Templating Configuration](#templating-configuration)
    - [Templating Parameters](#templating-parameters)
    - [Component Descriptor](#component-descriptor)
    - [Helm Template Format](#helm-template-format)

<p align="center">
  <img alt= "testrunner overview" src="https://github.com/gardener/test-infra/raw/master/docs/testrunner_overview.png">
</p>

## Usage

The testrunner is a basic go commandline tool that can be run by either calling it directly via
```
go run cmd/testmachinery-run/main.go [flags]
```
or via the prebuild binary which is updated and kept in sync with each new Test Machinery version (also available in the image `eu.gcr.io/gardener-project/gardener/testmachinery/testmachinery-run`)
```
testrunner [flags]
```

Available commands ([technical detailed docs](testrunner/testrunner.md)):
* [run-template](#pipeline-usage)
* run-testrun

## Pipeline Usage

The default usage of the testrunner is in a CI/CD pipline with the helm templating command.
In addition results of the helm-templated testruns are collected and stored in a database.

The templating can be used via
```
testrunner run|run-tmpl [flags]
```
:warning: As this command is intended to run in a CI/CD pipeline it depends on the gardener [cc-utils](https://github.com/gardener/cc-utils) library to store test results in an elasticsearch database.

### Templating Configuration

| flag | default | description | required
| ---- | ---- | ---- | --- |
| tm-kubeconfig-path | | Path to the kubeconfig of the cluster running the Test Machinery | x |
| testruns-chart-path | | Path to the Testrun helm template that should be deployed. (Additional information about the parameters can be found [here](#helm-template)) | x |
| testrun-prefix | | Prefix of the deployed Testrun. This prefix is used for the `metadata.generateName` of Testruns in the helm template. | x |
| namespace | default | Namespace where the testrun is deployed. |  |
| timeout | 3600 | Max seconds to wait for all Testruns to finish. | |
| version-matrix | false | Run the testrun with all available versions of the specified cloudprovider. :warning: The `k8s-version` is ingored if this parameter is set to true.| |
| output-file-path | "./testout" | The filepath where the test summary and results should be written to. | |
| s3-endpoint | EnvVar ("S3_ENDPOINT") | Accessible S3 endpoint of the s3 storage used by argo. This parameter is needed when tests export test results and the testrunner needs to fetch them and add them to the summary. | |
| s3-ssl | false | Enables ssl support for the corresponding s3 storage. | |
| argoui-endpoint | | Endpoint of the ArgoUI watching the Testmachinery cluster. | |
| es-config-name | | Elasticsearch server config name that is used with the cc-utils cli | |
| concourse-onError-dir | EnvVar ("ON_ERROR_DIR") | Directory where the `notify.cfg` should be written to. | |


### Templating Parameters

| flag | default | description | required
| ---- | ---- | ---- | --- |
| gardener-kubeconfig-path | | Path to a kubeconfig where a gardener is running. The kubeconfig will be base64 encoded and passed to the Testrun. | x |
| shoot-name | | Name of the shoot which is created for testing. | x |
| project-name | | Gardener project name where the shoot should be deployed. | x |
| cloudprovider | | Cloudprovider of the shoot. Can be `aws`, `azure`, `gcp`, `openstack`, `alicloud` | x |
| cloudprofile | | Cloudprofile that should be used to create the shoot. | x |
| secret-binding | | Gardener secret that should be used to create the shoot. <br>:warning: This needs to point to a valid secret for the cloudprofile. | x |
| region | | Region of the shoot | x |
| zone | | Zone of the shoot workers. <br>:warning: Note that currently only 1 workerpool is supported for testing. | x |
| k8s-version | | Kubernetes version of the created shoot. | x |
| machinetype | | Machinetype of the first worker pool. |  |
| autoscaler-min | | Minimum number of worker nodes of the first worker pool. | |
| autoscaler-max | | Maximum number of worker nodes of the first worker pool. | |
| floating-pool-name | | Floating pool name of the created cluster. Needed for Openstack. | Only required for Openstack clusters|
| component-descriptor-path | | Path to a component descriptor. The component descriptor will be automatically parsed and all github modules will be added to the testrun's `testLocations`. For further information see [here](#component-descriptor). | |
| landscape | | Semantic metadata information about the current landscape that is tested. | |

### Component Descriptor

See `cc-utils` repo and [documentation](https://gardener.github.io/cc-utils/traits/component_descriptor.html) for full documentation of how it is calculated in the gardener project.

A component descritor (BOM - Bill of Materials) describes all dependencies like github repositories (modules) and docker images of a project.

The structure of a component descriptor:
```yaml
components:
- name: component_name
  version: component_version
  dependencies:
    components:
    - name: component_dependency_name # e.g. github.com/gardener/test-infra
      version: component_dependency_version # e.g 0.12.2
    container_images:
    - image_reference: "" # eg. k8s.gcr.io/hyperkube:v1.11.6
      name: "" # e.g. hyperkube-1.13.2
      version: "" # e.g. 1.13.2
    generic:
    - name: "" # e.g. CoreOS
      version: "" # 1967.3.0
```

### Helm Template Format

The Testrunner integrates the helm templating engine and uses it to simplify the specification of different Testruns with different purposes.
It it also supported to have multiple testruns in one helm chart which all can be handled by the testrunner.

The helm charts need to have a specific format to be properly handled.

:information_source: The `generateName` attribute is automatically set by the Test Machinery using the commandline provided prefix.

```yaml
apiVersion: testmachinery.sapcloud.io/v1beta1
kind: Testrun
metadata:
  namespace: default
spec:

  ttlSecondsAfterFinished: 172800 # 2 days

  testLocations:
  - type: git
    repo: https://github.com/gardener/test-infra.git
    revision: master
  - type: git
    repo: https://github.com/gardener/gardener.git
    revision: d102d6d088490e5e59d88e4137ee954842be82ac

  kubeconfigs:
    gardener: {{ b64enc .Values.kubeconfigs.gardener }}

  config:
    - name: PROJECT_NAMESPACE
      type: env
      value: {{ .Values.shoot.projectNamespace }}
    - name: SHOOT_NAME
      type: env
      value: {{ .Values.shoot.name }}
    - name: CLOUDPROVIDER
      type: env
      value: {{ .Values.shoot.cloudprovider | default "gcp" }}
    - name: K8S_VERSION
      type: env
      value: {{ .Values.shoot.k8sVersion | default "1.12.4" }}

  testFlow:
  - - name: create-shoot
      config:
      - name: CLOUDPROFILE
        type: env
        value: {{ .Values.shoot.cloudprofile | default "gcp" }}
      - name: SECRET_BINDING
        type: env
        value: {{ .Values.shoot.secretBinding }}
      - name: REGION
        type: env
        value: {{ .Values.shoot.region }}
      - name: ZONE
        type: env
        value: {{ .Values.shoot.zone }}
      {{ if .Values.shoot.machinetype }}
      - name: MACHINE_TYPE
        type: env
        value: {{ .Values.shoot.machinetype }}
      {{ end }}
      {{ if .Values.shoot.autoscalerMin }}
      - name: AUTOSCALER_MIN
        type: env
        value: {{ .Values.shoot.autoscalerMin }}
      {{ end }}
      {{ if .Values.shoot.autoscalerMax }}
      - name: AUTOSCALER_MAX
        type: env
        value: {{ .Values.shoot.autoscalerMax }}
      {{ end }}

  - - label: default
  - - name: delete-shoot

  onExit:
  - - name: delete-shoot
      condition: error

```
