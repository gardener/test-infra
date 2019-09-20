# Host Scheduler

The host scheduler is a commandline tool to select kubernetes clusters from a pool of clusters of specific providers and lock them for a test execution.

The cluster can be cleaned and released after the tests are finished.
This will be like a factory reset of the cluster to be used again by other tests.

## Build and Run

```
go install ./

hostscheduler <provider> lock/release/ls/clean <args>
```

**ZSH Completion**
```
hostscheduler completion zsh > /somepath/_hostscheduler # e.g hostscheduler completion zsh > ~/.tm/_hostscheduler

# in .zshrc
fpath=( /somepath "${fpath[@]}" ) # e.g fpath=( ~/.tm "${fpath[@]}" )
autoload -Uz compinit
compinit -u

# Optional alias in .zshrc
alias hs='hostscheduler'
compdef hs=hostscheduler
```

## Provider

The scheduler is build to support multiple providers by using plugins that just need implementing the scheduler interface.
```golang
type Interface interface {
	Lock(*flag.FlagSet) (SchedulerFunc, error)
	Release(*flag.FlagSet) (SchedulerFunc, error)
	Cleanup(*flag.FlagSet) (SchedulerFunc, error)

	List(*flag.FlagSet) (SchedulerFunc, error)
}
```
* `lock`: selects an available cluster, locks the cluster to not be used by other tests and writes the cluster's kubeconfigs `$TM_KUBECONFIG_PATH/host.config`
* `release`: releases the cluster, so it can be used by other tests
* `clean`: cleans all k8s resources that are not system components of the corresponding provider
* `list`: only used for cli usage to get an overview of locked clusters

Configuration can be provided via commandline (get help by running `hostscheduler <provider> --help`) or via a config file that should be provided at `$HOME/.tm/hostscheduler.yaml`.
The config file shoudl have this format:
```yml
gardener:
  kubeconfig: <path to service account kubeconfig>
gke:
  project: <gcloud project name>
  zone: <gke regional cluster zone>
  gcloudKeyPath: <path to gcloud authorization json>
```

There are currently 2 providers supported _gardener_ and _gke_.

### Gardener
The gardener provider uses a pool of shoot clusters that are labeled with `"testmachinery.sapcloud.io/host": "true"`.

**Configuration**
| Flag | Configuration | Description |
| ---- | ----- | ---- |
| --kubeconfig | kubeconfig | Path to the service account kubeconfig |
| --cloudprovider | x | Select a specific cloudprovider. One of `all|aws|gcp|azure|alicloud|openstack|packet` |
| --name | x | OPTIONAL: use a specific cluster |

### GKE

The gke provider uses a pool of **regional** gke clusters that have the resource label `tm-host`

**Configuration**
| Flag | Configuration | Description |
| ---- | ----- | ---- |
| --key | gcloudKeyPath | Path to the gcloud json authentication file |
| --project | project | GCloud project name |
| --zone | zone | GCloud location of the cluster |
| --name | x | OPTIONAL: use a specific cluster |
