# Default Gardener Test

The default gardener tests a new gardener release or specifc commit.

The Testrun is generated in Golang by this [render function](../../pkg/testrunner/renderer/default/default.go).
This function is used by the testrunner and the github bot to create and run the test.

Th default test consists of the following steps:
- select a host cluster
- create a gardener
- test shoots
  - create shoot
  - test shoot
  - delete shoot
- delete gardener
- release host cluster

This tesflow can be configured to test different shoot flavors and test scenarios.
The configuration is wrapped by the testrunner and github bot to ease the configuration and set necessecary defaults for specific instlaltions and use cases.
```golang
// Config is used to render a default gardener test
type Config struct {
	// Namespace of the testrun
	Namespace string

	// Provider where the host clusters are selected from
	HostProvider hostscheduler.Provider

	// CloudProvider of the base cluster (has to be specified to install the correct credentials and cloudprofiles for the soil/seeds)
	BaseClusterCloudprovider gardenv1beta1.CloudProvider

	// Revision for the gardensetup repo that i sused to install gardener
	GardenSetupRevision string

	// List of components (by default read from a component_descriptor) that are added as locations
	Components componentdescriptor.ComponentList

	// Gardener specific configuration
	Gardener templates.GardenerConfig

	// Gardener tests that do not depend on shoots and run after the shoot tests
	Tests renderer.TestsFunc

	// Shoot test flavor configuration
	Shoots ShootsConfig
}

// ShootsConfig describes the flavors of the shoots that are created by the test.
// The resulting shoot test matrix consists of
// - shoot tests for all specified cloudproviders with all specified kubernets with the default test
// - shoot tests for all specified cloudproviders for all specified tests
type ShootsConfig struct {
	// Shoot/Project namespace where the shoots are created
	Namespace          string

	// Default test that is used for all cloudprovider and kubernetes flavors.
	DefaultTest        renderer.TestsFunc

	// Specific tests that get their own shoot per cloudprovider and run in parallel to the default tests
	Tests              []renderer.TestsFunc

	// Kubernetes versions to test
	KubernetesVersions []string

	// Cloiudproviders to test
	CloudProviders     []gardenv1beta1.CloudProvider
}
```

<img src='https://g.gravizo.com/svg?
 digraph G {
    node [shape=record];
    getHost [label="lock host", fillcolor=darkolivegreen1, style=filled];
    releaseHost [label="release host", fillcolor=darkolivegreen1, style=filled];
    createGardener [label="create gardener", fillcolor=darkolivegreen1, style=filled];
    deleteGardener [label="delete gardener", fillcolor=darkolivegreen1, style=filled];
    createShoot [label="create shoot"];
    deleteShoot [label="delete shoot"];
    createHibernatedShoot [label="create shoot"];
    deleteHibernatedShoot [label="delete shoot"];
    wakeup [label="wake up"];
    hibernate1 [label="hibernate"];
    hibernate2 [label="hibernate"];
    gardenerIT [label="gardener integration tests"];
    getHost -> createGardener;
    createGardener -> createShoot -> "test" -> deleteShoot -> gardenerIT;
    createGardener -> createHibernatedShoot -> hibernate1 -> wakeup -> hibernate2 -> deleteHibernatedShoot -> gardenerIT;
    gardenerIT -> deleteGardener -> releaseHost
 }
'/>