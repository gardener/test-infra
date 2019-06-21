## testrunner run-gardener-template

Run the testrunner with a helm template containing testruns

### Synopsis

Run the testrunner with a helm template containing testruns

```
testrunner run-gardener-template [flags]
```

### Options

```
      --component-descriptor-path string            Path to the component descriptor (BOM) of the current landscape.
      --concourse-onError-dir string                On error dir which is used by Concourse.
      --es-config-name string                       The elasticsearch secret-server config name. (default "sap_internal")
      --gardener-current-revision string            Set current revision of gardener. This will result in the helm value {{ .Values.gardener.current.revision }}
      --gardener-current-version string             Set current version of gardener. This will result in the helm value {{ .Values.gardener.current.version }}
      --gardener-upgraded-revision string           Set current revision of gardener. This will result in the helm value {{ .Values.gardener.upgraded.revision }}
      --gardener-upgraded-version string            Set current version of gardener. This will result in the helm value {{ .Values.gardener.upgraded.version }}
  -h, --help                                        help for run-gardener-template
      --interval int                                Poll interval in seconds of the testrunner to poll for the testrun status. (default 20)
  -n, --namespace string                            Namesapce where the testrun should be deployed. (default "default")
      --output-dir-path string                      The filepath where the summary should be written to. (default "./testout")
      --s3-endpoint string                          S3 endpoint of the testmachinery cluster.
      --s3-ssl                                      S3 has SSL enabled.
      --testrun-prefix string                       Testrun name prefix which is used to generate a unique testrun name. (default "default-")
      --testruns-chart-path string                  Path to the testruns chart.
      --timeout int                                 Timout in seconds of the testrunner to wait for the complete testrun to finish. (default 3600)
      --tm-kubeconfig-path string                   Path to the testmachinery cluster kubeconfig
      --upgraded-component-descriptor-path string   Path to the component descriptor (BOM) of the new landscape.
```

### Options inherited from parent commands

```
  -d, --debug     Set debug mode for additional output
      --dry-run   Dry run will print the rendered template
```

### SEE ALSO

* [testrunner](testrunner.md)	 - Testrunner for Test Machinery

