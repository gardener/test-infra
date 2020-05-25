# Test Machinery GitHub Bot

The Testmachinery GitHub bot is a [GitHub app](https://developer.github.com/apps/about-apps/) that listens on Pull Request events/webhooks and reacts on them.

It also checks if a user is authorized (currently is member of the organization) to perform teh command.

The bot is mainly build to run integration tests in PR's and report back the correct status.

## Plugins

[Command help](https://tm.gardener.cloud/command-help)

Plugins and their default values can be configured by creating a values file in `.ci/tm-bot` in the default branch of the repository.
This file will be parsed by the bot and the plugins will automatically be able to access the config for a certain call.
For detailed information about the plugins configuration see the respective plugin.

The configuration file has the following format:
```yaml
<command name>:
  custom_value1: xxx
  custom_value2: xxx

# Example

echo:
  value: hello

test:
  hostprovider: gardener
  baseClusterCloudprovider: gcp

  gardener:
    version:
      path: GARDENER_VERSION

  kubernetes:
    versions:
    - 1.14.4
    - 1.13.10

```

```yaml
# StringOrGitHubConfig
parameter: "string"

parameter:
  value: "string" # raw string value. Same as defining only a string
  path: test/path # read the file in the default branch of the repo (repo root will used to define the path) and return its content as a string
  prHead: true # use the commit sha of the current PR's head
```

## Development

### Run and install

#### Install
The tm bot is using github app authentication to access github's api and github webhooks to receive events and act accordingly.<br>
Therefore, you need to setup a github app ([create a github app](https://developer.github.com/apps/building-github-apps/creating-a-github-app/)) with the following permissions:
- Repository
  - Checks: read&write
  - Issues: read&write
  - PullRequests: read&write
  - Commit statuses: read&write
- Organization
  - Members: read
  - Blocking users: read

and events:
- Issues
- Issue comment
- Pull request
- Status

The github bot exposes its webhook handler at `/event/handler`
```
go run ./cmd/tm-bot
  --config=/path/to/config
  --kubeconfig=/path/to/config # optional
```

See the full Bot Configuration [here](/examples/50-bot-configuration.yaml)

### Plugins

Plugins are the commands that are executed by the bot when someone is writing a `/<cmd>` as Pull Request comment.
Therefore plugins consist of the following interface and can be added to the bot by registering it [here](../../pkg/tm-bot/hok/handler.go).
```golang
// Plugin specifies a tm github bot plugin/command that can be triggered by a user
type Plugin interface {
	// Command returns the unique matching command for the plugin
	Command() string

	// Flags return command line style flags for the command
	Flags() *pflag.FlagSet

	// Run runs the command with the parsed flags (flag.Parse()) and the event that triggered the command
	Run(fs *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error

	// resume the plugin execution from a persisted state
	ResumeFromState(client github.Client, event *github.GenericRequestEvent, state string) error

	// Description returns a short description of the plugin
	Description() string

	// Example returns an example for the command
	Example() string

	// Create a deep copy of the plugin
	New(runID string) Plugin
}
```

#### State
Every plugin call will get its own state in the plugins during their execution.
This state is persisted in a ConfigMap the cluster (could be changed int the future) and is used to resume plugin executions after the bot restarted or was updated, etc. .
When the plugin is finished this state gets removed automatically.

If a Plugin needs their own state to resume after the bot has restarted or gets updated, it can add their state to the plugins by just calling `UpdateState` on the plugins and the state will be automatically persisted in the configured persistance.

When the bot get started all saved states are read from the persistance and the bot will automatically resume their execution.

