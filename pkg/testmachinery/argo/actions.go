// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package argo

import (
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
