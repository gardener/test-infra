// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrun

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
)

// Testrun is the internal representation of a testrun crd
type Testrun struct {
	Info            *tmv1beta1.Testrun
	Testflow        *testflow.Testflow
	OnExitTestflow  *testflow.Testflow
	HelperResources []client.Object
	ProjectedTokens map[string]*node.ProjectedTokenMount
}
