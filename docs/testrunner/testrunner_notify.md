## testrunner notify

Posts a result table of a previous run as table to slack.

### Synopsis

Posts a result table of a previous run as table to slack.

```
testrunner notify [flags]
```

### Options

```
      --concourse-url string         Concourse job URL.
      --github-password string       Github password.
      --github-repo string           Specify the Github repository that should be used to get the test results
      --github-repo-version string   Specify the version fot the Github repository that should be used to get the test results
      --github-user string           On error dir which is used by Concourse.
  -h, --help                         help for notify
      --overview string              Name of the overview asset file in the release.
      --slack-channel string         Client channel id to send the message to.
      --slack-token string           Client token to authenticate
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

