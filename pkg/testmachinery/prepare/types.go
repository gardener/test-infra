package prepare

import "github.com/gardener/test-infra/pkg/testmachinery/testdefinition"

// PrepareDefinition is the TestDefinition of the prepare step to initiliaze the setup.
type Definition struct {
	TestDefinition *testdefinition.TestDefinition
	repositories   []*Repository
}

// PrepareRepository is passed as a json array to the prepare step.
type Repository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Revision string `json:"revision"`
}
