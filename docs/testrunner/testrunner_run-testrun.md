## testrunner run-testrun

Run the testrunner with a testrun

### Synopsis

Run the testrunner with a testrun

```
testrunner run-testrun [flags]
```

### Options

```
  -f, --file string                  Path to the testrun yaml
  -h, --help                         help for run-testrun
      --interval int                 Poll interval in seconds of the testrunner to poll for the testrun status. (default 20)
      --name-prefix string           Name prefix of the testrun (default "testrunner-")
  -n, --namespace string             Namespace where the testrun should be deployed. (default "default")
      --testrun-flake-attempts int   Max number of testruns until testrun is successful
      --timeout int                  Timout in seconds of the testrunner to wait for the complete testrun to finish. (default 3600)
      --tm-kubeconfig-path string    Path to the testmachinery cluster kubeconfig
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

