// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package argo

import (
	"context"
	"fmt"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testmachinery"
)

// Scheme defines the kubernetes scheme that contains the argo resources.
var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(argov1.AddToScheme(Scheme))
}

// CreateWorkflow takes a name, templates and volumes to generate an argo workflow object.
func CreateWorkflow(name, namespace, entrypoint, onExitName string, templates []argov1.Template, volumes []corev1.Volume, ttl *int32, pullImageSecretNames []string) (*argov1.Workflow, error) {

	wf := &argov1.Workflow{
		Spec: argov1.WorkflowSpec{
			Affinity:         getWorkflowAffinity(),
			Tolerations:      getWorkflowTolerations(),
			Entrypoint:       entrypoint,
			ImagePullSecrets: getImagePullSecrets(pullImageSecretNames),
			Volumes:          volumes,
			Templates:        append(templates, SuspendTemplate()),
		},
	}

	if ttl != nil {
		wf.Spec.TTLStrategy = &argov1.TTLStrategy{
			SecondsAfterCompletion: ttl,
		}
	}

	if onExitName != "" {
		wf.Spec.OnExit = onExitName
	}

	wf.Name = name
	wf.Namespace = namespace

	return wf, nil
}

// DeployWorkflow creates the given argo workflow object in the given k8s cluster.
func DeployWorkflow(ctx context.Context, wf *argov1.Workflow, masterURL, kubeconfig string) (*argov1.Workflow, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(cfg, client.Options{
		Scheme: Scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client: %w", err)
	}

	if err := kubeClient.Create(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

// CreateTask takes a name, the running phase name, dependencies and artifacts, and return an argo task object.
func CreateTask(taskName, templateName, phase string, continueOnError bool, dependencies []string, artifacts []argov1.Artifact) argov1.DAGTask {
	return argov1.DAGTask{
		Name:         taskName,
		Template:     templateName,
		Dependencies: dependencies,
		ContinueOn: &argov1.ContinueOn{
			Error:  continueOnError,
			Failed: continueOnError,
		},

		Arguments: argov1.Arguments{
			Artifacts: artifacts,
			Parameters: []argov1.Parameter{
				{
					Name:  "phase",
					Value: argov1.AnyStringPtr(phase),
				},
			},
		},
	}
}

// CreateSuspendTask creates a suspend task with a name and dependencies.
// This task is used to pause before a specific step
func CreateSuspendTask(name string, dependencies []string) argov1.DAGTask {
	return argov1.DAGTask{
		Name:         testmachinery.GetPauseTaskName(name),
		Template:     testmachinery.PauseTemplateName,
		Dependencies: dependencies,
	}
}

// SuspendTemplate resturn the shared template for suspended tasks
func SuspendTemplate() argov1.Template {
	return argov1.Template{
		Name:    testmachinery.PauseTemplateName,
		Suspend: &argov1.SuspendTemplate{},
	}
}

// getImagePullSecrets returns a list of LocalObjectReference generated of the provided ImagePullSecretNmes
func getImagePullSecrets(pullSecretNames []string) []corev1.LocalObjectReference {
	secrets := []corev1.LocalObjectReference{}
	for _, name := range pullSecretNames {
		secrets = append(secrets, corev1.LocalObjectReference{Name: name})
	}
	return secrets
}

// getWorkflowAffinity returns the default spec to prefer workflow pods being scheduled on specially labeled nodes
func getWorkflowAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 100,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "purpose",
								Operator: "In",
								Values:   []string{testmachinery.WorkerPoolTaintLabelName},
							},
						},
					},
				},
			},
		},
	}
}

// getWorkflowTolerations returns the default spec to allow workflow pods being scheduled on specially tainted nodes
func getWorkflowTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:      "purpose",
			Operator: "Equal",
			Value:    testmachinery.WorkerPoolTaintLabelName,
			Effect:   "NoSchedule",
		},
	}
}
