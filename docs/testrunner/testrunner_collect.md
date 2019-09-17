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
      --tm-kubeconfig-path string      Path to the testmachinery cluster kubeconfig (default "/Users/d064999/.kubeconfigs/dev/.virtual")
  -t, --tr-name string                 Name of the testrun to collect results.
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

