## testrunner run-template

Run the testrunner with a helm template containing testruns

### Synopsis

Run the testrunner with a helm template containing testruns

```
testrunner run-template [flags]
```

### Options

```
      --asset-component stringArray           The github components to which the testrun status shall be attached as an asset.
      --asset-prefix string                   Prefix of the asset name.
      --component-descriptor-path string      Path to the component descriptor (BOM) of the current landscape.
      --concourse-onError-dir string          On error dir which is used by Concourse.
      --enable-telemetry                      Enables the measurements of metrics during execution
      --es-config-name string                 The elasticsearch secret-server config name. (default "sap_internal")
      --fail-on-error                         Testrunners exits with 1 if one testruns failed. (default true)
      --filter-patch-versions                 Filters patch versions so that only the latest patch versions per minor versions is used.
      --flavor-config string                  Path to shoot test configuration.
      --flavored-testruns-chart-path string   Path to the testruns chart to test shoots.
      --gardener-kubeconfig-path string       Path to the gardener kubeconfig.
      --github-password string                Github password.
      --github-user string                    On error dir which is used by Concourse.
  -h, --help                                  help for run-template
      --interval int                          Poll interval in seconds of the testrunner to poll for the testrun status. (default 20)
      --landscape string                      Current gardener landscape.
  -n, --namespace string                      Namesapce where the testrun should be deployed. (default "default")
      --output-dir-path string                The filepath where the summary should be written to. (default "./testout")
      --s3-endpoint string                    S3 endpoint of the testmachinery cluster.
      --s3-ssl                                S3 has SSL enabled.
      --set string                            setValues additional helm values
      --shoot-name string                     Shoot name which is used to run tests.
      --testrun-prefix string                 Testrun name prefix which is used to generate a unique testrun name. (default "default-")
      --testruns-chart-path string            Path to the default testruns chart.
      --timeout int                           Timout in seconds of the testrunner to wait for the complete testrun to finish. (default 3600)
      --tm-kubeconfig-path string             Path to the testmachinery cluster kubeconfig
      --upload-status-asset                   Upload testrun status as a github release asset.
  -f, --values stringArray                    yaml value files to override template values
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

