## testrunner

Testrunner for Test Machinery

### Synopsis

Testrunner for Test Machinery

### Options

```
      --cli                  logger runs as cli logger. enables cli logging
      --dev                  enable development logging which result in console encoding, enabled stacktrace and enabled caller
      --disable-caller       disable the caller of logs (default true)
      --disable-stacktrace   disable the stacktrace of error logs (default true)
      --disable-timestamp    disable timestamp output (default true)
      --dry-run              Dry run will print the rendered template
  -h, --help                 help for testrunner
  -v, --verbosity int        number for the log level verbosity (default 1)
```

### SEE ALSO

* [testrunner collect](testrunner_collect.md)	 - Collects results from a completed testrun.
* [testrunner docs](testrunner_docs.md)	 - Generate docs for the testrunner
* [testrunner gardener-telemetry](testrunner_gardener-telemetry.md)	 - Collects metrics during gardener updates until gardener is updated and all shoots are successfully reconciled
* [testrunner run-gardener](testrunner_run-gardener.md)	 - Run the testrunner with the default gardener test
* [testrunner run-template](testrunner_run-template.md)	 - Run the testrunner with a helm template containing testruns
* [testrunner run-testrun](testrunner_run-testrun.md)	 - Run the testrunner with a testrun
* [testrunner version](testrunner_version.md)	 - GetInterface testrunner version

