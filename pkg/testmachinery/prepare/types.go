package prepare

import "github.com/gardener/test-infra/pkg/testmachinery/testdefinition"

const (
	PrepareConfigPath = "/tm/config.json"
)

// PrepareDefinition is the TestDefinition of the prepare step to initialiaze the setup.
type Definition struct {
	TestDefinition *testdefinition.TestDefinition
	GlobalInput    bool
	config         Config
}

// Config represents the configuration for the prepare step.
// It defined which repos should be cloned and which folder have to be created
type Config struct {
	Directories  []string               `json:"directories"`
	Repositories map[string]*Repository `json:"repositories"`
}

// PrepareRepository is passed as a json array to the prepare step.
type Repository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Revision string `json:"revision"`
}
