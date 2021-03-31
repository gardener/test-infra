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
	argov1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
)

// ResumeWorkflow resumes the suspend step that currently blocks the execution.
func ResumeWorkflow(wf *argov1.Workflow) {
	for id, status := range wf.Status.Nodes {
		if status.Type == argov1.NodeTypeSuspend && status.Phase == argov1.NodeRunning {
			status.Phase = argov1.NodeSucceeded
			wf.Status.Nodes[id] = status
			return
		}
	}
}

// ResumeWorkflowStep resumes a specific step of a workflow.
// If the step is already resumed false is returned.
func ResumeWorkflowStep(wf *argov1.Workflow, stepName string) bool {
	for id, status := range wf.Status.Nodes {
		if status.Name == stepName {
			if status.Type == argov1.NodeTypeSuspend && status.Phase == argov1.NodeRunning {
				status.Phase = argov1.NodeSucceeded
				wf.Status.Nodes[id] = status
				return true
			}
			return false
		}
	}
	return false
}

func GetRunningSteps(wf *argov1.Workflow) []argov1.NodeStatus {
	nodes := make([]argov1.NodeStatus, 0)
	for _, status := range wf.Status.Nodes {
		if status.Phase == argov1.NodeRunning {
			nodes = append(nodes, status)
		}
	}

	return nodes
}
