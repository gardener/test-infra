# Elasticsearch data manipulation tool

```
Elasticsearch tool for TestMachinery

Usage:
  elasticsearch [command]

Available Commands:
  help        Help about any command
  precompute  Reads existing teststep metadata, re-computes the current precomputed values and optionally updates the respective elasticsearch document.
  ingest      Verifies that ingestion of testrun metadata into elasticsearch/opensearch works.

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

#### Note: read-only indexes
If you use curator to roll-over indexes, they will be made read-only-allow-delete, hence before attempting to change data, make them read-write again:
```shell
# check which indexes are read-only-allow-delete
curl -s 'localhost:9200/testmachinery-*' | jq '.[].settings.index | {index: .provided_name, readOnly: .blocks.read_only_allow_delete}' -c

# remove read-only-allow-delete for one index
curl -X PUT "localhost/testmachinery-000017/_settings" -H 'Content-Type: application/json' -d'{ "index.blocks.read_only_allow_delete" : null } }'
```