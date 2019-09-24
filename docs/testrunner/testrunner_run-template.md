## testrunner run-template

Run the testrunner with a helm template containing testruns

### Synopsis

Run the testrunner with a helm template containing testruns

```
testrunner run-template [flags]
```

### Options

```
      --all-k8s-versions                   Run the testrun with all available versions specified by the cloudprovider.
      --autoscaler-max string              Max number of worker nodes.
      --autoscaler-min string              Min number of worker nodes.
      --cloudprofile string                Cloudprofile of shoot.
      --cloudprovider string               Cloudprovider where the shoot is created.
      --component-descriptor-path string   Path to the component descriptor (BOM) of the current landscape.
      --concourse-onError-dir string       On error dir which is used by Concourse.
      --enable-telemetry                   Enables the measurements of metrics during execution
      --es-config-name string              The elasticsearch secret-server config name. (default "sap_internal")
      --fail-on-error                      Testrunners exits with 1 if one testruns failed. (default true)
      --floating-pool-name string          Floating pool name where the cluster is created. Only needed for Openstack.
      --gardener-kubeconfig-path string    Path to the gardener kubeconfig.
  -h, --help                               help for run-template
      --interval int                       Poll interval in seconds of the testrunner to poll for the testrun status. (default 20)
      --k8s-version string                 Kubernetes version of the shoot.
      --landscape string                   Current gardener landscape.
      --loadbalancer-provider string       LoadBalancer Provider like haproxy. Only applicable for Openstack.
      --machine-image string               Image of the OS running on the machine
      --machine-image-version string       The version of the machine image
      --machinetype string                 Machinetype of the shoot's worker nodes.
  -n, --namespace string                   Namesapce where the testrun should be deployed. (default "default")
      --output-dir-path string             The filepath where the summary should be written to. (default "./testout")
      --project-name string                Gardener project name of the shoot
      --region string                      Region where the shoot is created.
      --s3-endpoint string                 S3 endpoint of the testmachinery cluster.
      --s3-ssl                             S3 has SSL enabled.
      --secret-binding string              SecretBinding that should be used to create the shoot.
      --set string                         setValues additional helm values
      --shoot-name string                  Shoot name which is used to run tests.
      --testrun-prefix string              Testrun name prefix which is used to generate a unique testrun name. (default "default-")
      --testruns-chart-path string         Path to the testruns chart.
      --timeout int                        Timout in seconds of the testrunner to wait for the complete testrun to finish. (default 3600)
      --tm-kubeconfig-path string          Path to the testmachinery cluster kubeconfig
  -f, --values stringArray                 yaml value files to override template values
      --zone string                        Zone of the shoot worker nodes. Not required for azure shoots.
```

### Options inherited from parent commands

```
      --cli                  logger runs as cli logger. enables cli logging
      --dev                  enable development logging which result in console encoding, enabled stacktrace and enabled caller
      --disable-caller       disable the caller of logs (default true)
      --disable-stacktrace   disable the stacktrace of error logs (default true)
      --disable-timestamp    disable timestamp output (default true)
      --dry-run              Dry run will print the rendered template
  -v, --verbosity int        number for the log level verbosity (default 1)
```

### SEE ALSO

* [testrunner](testrunner.md)	 - Testrunner for Test Machinery

