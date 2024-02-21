// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var _ = Describe("workflow", func() {
	It("should mark the testrun as success if all nodes are successfull", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.WorkflowSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.RunPhaseSuccess))
	})

	It("should mark the testrun as running if it is not completed yet", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.WorkflowRunning,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeFailed,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.RunPhaseRunning))
	})

	It("should mark the testrun as failure if a node are failed", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.WorkflowSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeFailed,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.RunPhaseFailed))
	})

	It("should mark the testrun as error if a node is erroneous", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.WorkflowSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeError,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.RunPhaseError))
	})
})
