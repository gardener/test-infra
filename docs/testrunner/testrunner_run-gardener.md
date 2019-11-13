## testrunner run-gardener

Run the testrunner with the default gardener test

### Synopsis

Run the testrunner with the default gardener test

```
testrunner run-gardener [flags]
```

### Options

```
  -p, --cloudprovider CloudProviderArray   Specify the cloudproviders to test.
      --component-descriptor-path string   Path to the component descriptor (BOM) of the current landscape.
      --concourse-onError-dir string       On error dir which is used by Concourse.
      --es-config-name string              The elasticsearch secret-server config name. (default "sap_internal")
      --fail-on-error                      Testrunners exits with 1 if one testruns failed. (default true)
      --garden-setup-version string        Specify the garden setup version to setup gardener (default "master")
      --gardener-commit string             Specify the gardener commit that is deployed by garden setup
      --gardener-extensions string         Specify the gardener extensions versions to be deployed by garden setup (default "provider-gcp=github.com/gardener/gardener-extensions.git:master")
      --gardener-image string              Specify the gardener image tag to be deployed by garden setup
      --gardener-version string            Specify the gardener version to be deployed by garden setup
  -h, --help                               help for run-gardener
      --hibernation                        test hibernation
      --host-cloudprovider CloudProvider   Specify the cloudprovider of the host cluster. Optional and only affect gardener base cluster (default gcp)
      --hostprovider HostProvider          Specify the provider for selecting the base cluster (default gardener)
      --interval duration                  Poll interval of the testrunner to poll for the testrun status. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'. (default 20s)
      --kubeconfig string                  Path to the testmachinery cluster kubeconfig
      --kubernetes-version stringArray     Specify the kubernetes version to test
  -l, --label string                       Specify test label that should be fetched by the testmachinery (default "default")
  -n, --namespace string                   Namespace where the testrun should be deployed. (default "default")
      --output-dir-path string             The filepath where the summary should be written to. (default "./testout")
      --project-namespace string           Specify the shoot namespace where the shoots should be created (default "garden-core")
      --s3-endpoint string                 S3 endpoint of the testmachinery cluster.
      --s3-ssl                             S3 has SSL enabled.
      --testrun-prefix string              Testrun name prefix which is used to generate a unique testrun name. (default "default-")
      --timeout duration                   Timout the testrunner to wait for the complete testrun to finish. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'. (default 1h0m0s)
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

