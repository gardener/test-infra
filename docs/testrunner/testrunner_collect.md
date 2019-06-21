## testrunner collect

Collects results from a completed testrun.

### Synopsis

Collects results from a completed testrun.

```
testrunner collect [flags]
```

### Options

```
      --concourse-onError-dir string   On error dir which is used by Concourse.
      --es-config-name string          The elasticsearch secret-server config name.
  -h, --help                           help for collect
  -n, --namespace string               Namespace where the testrun should be deployed. (default "default")
  -o, --output-dir-path string         The filepath where the summary should be written to. (default "./testout")
      --s3-endpoint string             S3 endpoint of the testmachinery cluster.
      --s3-ssl                         S3 has SSL enabled.
      --tm-kubeconfig-path string      Path to the testmachinery cluster kubeconfig
  -t, --tr-name string                 Name of the testrun to collect results.
```

### Options inherited from parent commands

```
  -d, --debug     Set debug mode for additional output
      --dry-run   Dry run will print the rendered template
```

### SEE ALSO

* [testrunner](testrunner.md)	 - Testrunner for Test Machinery

