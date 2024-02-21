// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testdefinition

import (
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
)

func NewEmpty() *TestDefinition {
	td := TestDefinition{
		Info:            &v1beta1.TestDefinition{},
		Template:        &argov1.Template{},
		inputArtifacts:  make(ArtifactSet),
		outputArtifacts: make(ArtifactSet),
		config:          make(config.Set),
	}

	return &td
}
