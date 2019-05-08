package node

import (
	"fmt"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

func GetUniqueName(td *testdefinition.TestDefinition, step *tmv1beta1.DAGStep, flow string) string {
	name := td.Info.Metadata.Name
	if step != nil {
		name = fmt.Sprintf("%s-%s", name, step.Name)
	}

	return fmt.Sprintf("%s-%s", name, flow)
}
