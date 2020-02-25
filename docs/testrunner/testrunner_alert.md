## testrunner alert

Evaluates recently completed testruns and sends alerts for failed  testruns if conditions are met.

### Synopsis

Evaluates recently completed testruns and sends alerts for failed  testruns if conditions are met.

```
testrunner alert [flags]
```

### Options

```
      --elasticsearch-endpoint string   Elasticsearch endpoint URL
      --elasticsearch-pass string       Elasticsearch password
      --elasticsearch-user string       Elasticsearch username
      --eval-time-days int              if test fails >=n times send alert (default 3)
      --focus stringArray               regexp to keep context test names e.g. 'e2e-untracked.*aws. Is executed after skip filter.'
  -h, --help                            help for alert
      --min-continuous-failures int     if test fails >=n times send alert (default 3)
      --min-success-rate int            if test success rate % falls below threshold, then post an alert (default 50)
      --skip stringArray                regexp to filter context test names e.g. 'e2e-untracked.*aws'
      --slack-channel string            Client channel id to send the message to.
      --slack-token string              Client token to authenticate
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

