// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argo

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateWorkflow takes a name, templates and volumes to generate an argo workflow object.
func CreateWorkflow(name, namespace, entrypoint, onExitName string, templates []argov1.Template, volumes []corev1.Volume, ttl *int32, pullImageSecretNames []string) (*argov1.Workflow, error) {

	wf := &argov1.Workflow{
		Spec: argov1.WorkflowSpec{
			Entrypoint:              entrypoint,
			ImagePullSecrets:        getImagePullSecrets(pullImageSecretNames),
			Volumes:                 volumes,
			Templates:               templates,
			TTLSecondsAfterFinished: ttl,
		},
	}

	if onExitName != "" {
		wf.Spec.OnExit = onExitName
	}

	wf.Name = name
	wf.Namespace = namespace

	return wf, nil
}

// DeployWorkflow creates the given argo workflow object in the given k8s cluster.
func DeployWorkflow(wf *argov1.Workflow, masterURL, kubeconfig string) (*argov1.Workflow, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return nil, err
	}
	argoclient, err := argoclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	wf, err = argoclient.ArgoprojV1alpha1().Workflows("default").Create(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

// CreateTask takes a name, the running phase name, dependencies and artifacts, and return an argo task object.
func CreateTask(taskName, templateName, phaseRunning string, dependencies []string, artifacts []argov1.Artifact) *argov1.DAGTask {
	return &argov1.DAGTask{
		Name:         taskName,
		Template:     templateName,
		Dependencies: dependencies,
		Arguments: argov1.Arguments{
			Parameters: []argov1.Parameter{
				{
					Name:  "phase",
					Value: &phaseRunning,
				},
			},
			Artifacts: artifacts,
		},
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
