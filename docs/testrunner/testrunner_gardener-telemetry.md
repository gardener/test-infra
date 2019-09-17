## testrunner gardener-telemetry

Collects metrics during gardener updates until gardener is updated and all shoots are successfully reconciled

### Synopsis

Collects metrics during gardener updates until gardener is updated and all shoots are successfully reconciled

```
testrunner gardener-telemetry [flags]
```

### Options

```
      --component-descriptor string   Path to component descriptor
  -h, --help                          help for gardener-telemetry
      --kubeconfig string             Path to the gardener kubeconfig (default "/Users/d064999/.kubeconfigs/dev/.virtual")
      --result-dir string             Path to write the metricss (default "/tmp/res")
      --timeout string                Initial timeout to wait for the update to start. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'. (default "1m")
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

