## testrunner run-gardener

Run the testrunner with the default gardener test

### Synopsis

Run the testrunner with the default gardener test

```
testrunner run-gardener [flags]
```

### Options

```
      --asset-component stringArray        The github components to which the testrun status shall be attached as an asset.
      --asset-prefix string                Prefix of the asset name.
  -p, --cloudprovider CloudProviderArray   Specify the cloudproviders to test.
      --component-descriptor-path string   Path to the component descriptor (BOM) of the current landscape.
      --concourse-onError-dir string       On error dir which is used by Concourse.
      --fail-on-error                      Testrunners exits with 1 if one testruns failed. (default true)
      --garden-setup-version string        Specify the garden setup version to setup gardener (default "master")
      --gardener-commit string             Specify the gardener commit that is deployed by garden setup
      --gardener-extensions string         Specify the gardener extensions versions to be deployed by garden setup (default "provider-gcp=github.com/gardener/gardener-extensions.git::master")
      --gardener-image string              Specify the gardener image tag to be deployed by garden setup
      --gardener-version string            Specify the gardener version to be deployed by garden setup
      --github-password string             Github password.
      --github-user string                 GitHub username.
  -h, --help                               help for run-gardener
      --hibernation                        test hibernation
      --host-cloudprovider CloudProvider   Specify the cloudprovider of the host cluster. Optional and only affect gardener base clusters (default gcp)
      --hostprovider HostProvider          Specify the provider for selecting the base cluster (default gardener)
      --interval string                    [DEPRECTAED] Value has no effect on the testrunner (default "20s")
      --kubeconfig string                  Path to the testmachinery cluster kubeconfig (default "/Users/d064999/.kubeconfigs/office/garden-core/tm-stg.shoot")
      --kubernetes-version stringArray     Specify the kubernetes version to test
  -l, --label string                       Specify test label that should be fetched by the testmachinery (default "default")
      --landscape string                   gardener landscape name
  -n, --namespace string                   Namespace where the testrun should be deployed. (default "default")
      --project-namespace string           Specify the shoot namespace where the shoots should be created (default "garden-core")
      --testrun-prefix string              Testrun name prefix which is used to generate a unique testrun name. (default "default-")
      --timeout duration                   Timout the testrunner to wait for the complete testrun to finish. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'. (default 1h0m0s)
      --upload-status-asset                Upload testrun status as a github release asset.
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

