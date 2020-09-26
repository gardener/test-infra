# Elasticsearch data manipulation tool

```
Elasticsearch tool for TestMachinery

Usage:
  elasticsearch [command]

Available Commands:
  help        Help about any command
  precompute  Reads existing teststep metadata, re-computes the current precomputed values and optionally updates the respective elasticsearch document.

Flags:
      --cli                  logger runs as cli logger. enables cli logging
      --dev                  enable development logging which result in console encoding, enabled stacktrace and enabled caller
      --disable-caller       disable the caller of logs (default true)
      --disable-stacktrace   disable the stacktrace of error logs (default true)
      --disable-timestamp    disable timestamp output (default true)
      --endpoint string      Elasticsearch endpoint, e.g. https://example.com:9200
  -h, --help                 help for elasticsearch
      --password string      Elasticsearch basic auth password
      --user string          Elasticsearch basic auth username
  -v, --verbosity int        number for the log level verbosity (default 1)

Use "elasticsearch [command] --help" for more information about a command.
subcommand is requiredexit status 1
```

# Commands
## `precompute`
This command reads all existing teststep metadata from elasticsearch, then re-computes all the precomputed values like phaseNum and clusterDomain and if changes to the existing `.pre` field are detected, updates them in elasticsearch.
This is useful if you want to modify or amend the precomputed values and fields in the testmachinery code and then want to update all existing data.  