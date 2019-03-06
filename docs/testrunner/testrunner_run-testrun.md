## testrunner run-testrun

Run the testrunner with a testrun

### Synopsis

Run the testrunner with a testrun

```
testrunner run-testrun [flags]
```

### Options

```
  -f, --file string                 Path to the testrun yaml
  -h, --help                        help for run-testrun
      --interval int                Poll interval in seconds of the testrunner to poll for the testrun status. (default 20)
      --name-prefix string          Name prefix of the testrun (default "testrunner-")
  -n, --namespace string            Namesapce where the testrun should be deployed. (default "default")
      --timeout int                 Timout in seconds of the testrunner to wait for the complete testrun to finish. (default 3600)
      --tm-kubeconfig-path string   Path to the testmachinery cluster kubeconfig
```

### Options inherited from parent commands

```
  -d, --debug   Set debug mode for additional output
```

### SEE ALSO

* [testrunner](testrunner.md)	 - Testrunner for Test Machinery

