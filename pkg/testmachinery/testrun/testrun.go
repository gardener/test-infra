// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrun

import (
	"context"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/prepare"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
)

// New takes a testrun crd and creates a new Testrun representation.
// It fetches testruns from specified testdeflocations and generates a testflow object.
func New(ctx context.Context, log logr.Logger, tr *tmv1beta1.Testrun, reader client.Reader) (*Testrun, error) {

	kubeconfigs, secrets, projectedTokenMounts, err := ParseKubeconfigs(ctx, reader, tr)
	if err != nil {
		return nil, err
	}

	locs, err := locations.NewLocations(log, tr.Spec)
	if err != nil {
		return nil, err
	}

	globalConfig := config.New(tr.Spec.Config, config.LevelGlobal)
	globalConfig = append(globalConfig, config.NewElement(createTestrunIDConfig(tr.Name), config.LevelGlobal))

	// create initial prepare step
	prepareDef, err := prepare.New("prepare", false, true)
	if err != nil {
		return nil, err
	}
	prepareDef.TestDefinition.AddConfig(kubeconfigs)
	tf, err := testflow.New(testflow.FlowIDTest, tr.Spec.TestFlow, locs, globalConfig, prepareDef)
	if err != nil {
		return nil, err
	}

	postPrepareDef, err := prepare.New("post-prepare", true, false)
	if err != nil {
		return nil, err
	}
	postPrepareDef.TestDefinition.AddConfig(kubeconfigs)
	onExitFlow, err := testflow.New(testflow.FlowIDExit, tr.Spec.OnExit, locs, globalConfig, postPrepareDef)
	if err != nil {
		return nil, err
	}

	return &Testrun{
		Info:            tr,
		Testflow:        tf,
		OnExitTestflow:  onExitFlow,
		HelperResources: secrets,
		ProjectedTokens: projectedTokenMounts,
	}, nil
}

// GetWorkflow returns the argo workflow object of this testrun.
func (tr *Testrun) GetWorkflow(name, namespace string, pullImageSecretNames []string) (*argov1.Workflow, error) {
	testrunName := "testrun"
	onExitName := "exit-handler"

	trustedTokenMounts := make([]node.ProjectedTokenMount, 0)
	untrustedTokenMounts := make([]node.ProjectedTokenMount, 0)

	for name, tokenMount := range tr.ProjectedTokens {
		switch name {
		case shootKubeconfig:
			untrustedTokenMounts = append(untrustedTokenMounts, *tokenMount)
			trustedTokenMounts = append(trustedTokenMounts, *tokenMount)
		default:
			trustedTokenMounts = append(trustedTokenMounts, *tokenMount)
		}
	}

	templates, err := tr.Testflow.GetTemplates(testrunName, testmachinery.PhaseRunning, trustedTokenMounts, untrustedTokenMounts)
	if err != nil {
		return nil, err
	}
	onExitTemplates, err := tr.OnExitTestflow.GetTemplates(onExitName, testmachinery.PhaseExit, trustedTokenMounts, untrustedTokenMounts)
	if err != nil {
		return nil, err
	}

	volumes := tr.Testflow.GetLocalVolumes()
	onExitVolumes := tr.OnExitTestflow.GetLocalVolumes()

	return argo.CreateWorkflow(name, namespace, testrunName, onExitName, append(templates, onExitTemplates...), append(volumes, onExitVolumes...), tr.Info.Spec.TTLSecondsAfterFinished, pullImageSecretNames)
}

func createTestrunIDConfig(id string) *tmv1beta1.ConfigElement {
	return &tmv1beta1.ConfigElement{
		Type:  tmv1beta1.ConfigTypeEnv,
		Name:  testmachinery.TM_TESTRUN_ID_NAME,
		Value: id,
	}
}
