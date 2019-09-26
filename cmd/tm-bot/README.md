# Test Machinery GitHub Bot

The Testmachinery GitHub bot is a [GitHub app](https://developer.github.com/apps/about-apps/) that listens on Pull Request events/webhooks and reacts on them.

It also checks if a user is authorized (currently is member of the organization) to perform the command.

The bot is mainly build to run integration tests in PR's and report back the correct status.

## Plugins

Currently there are the following plugins available.

#### echo
Writes the value as PR comment
```
Command: echo
Flags: --value string
```
#### xkcd
Prints a random or flag specified xkcd
```
Command: xkcd
Flags: --num int
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
  --github-app-id=<appid>
  --github-key-file=<path to github app private key>
```

### plugins

Plugins are the commands that are executed by github when someone is writing a `/<cmd>` as Pull Request comment.
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

	// Description returns a short description of the plugin
	Description() string

	// Example returns an example for the command
	Example() string
}
```

