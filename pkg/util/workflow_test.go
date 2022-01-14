// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package util

import (
	argov1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var _ = Describe("workflow", func() {
	It("should mark the testrun as success if all nodes are successfull", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.NodeSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.PhaseStatusSuccess))
	})

	It("should mark the testrun as running if it is not completed yet", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.NodeRunning,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeFailed,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.PhaseStatusRunning))
	})

	It("should mark the testrun as failure if a node are failed", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.NodeSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeFailed,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.PhaseStatusFailed))
	})

	It("should mark the testrun as error if a node is erroneous", func() {
		wf := &argov1.Workflow{Status: argov1.WorkflowStatus{
			Phase: argov1.NodeSucceeded,
			Nodes: argov1.Nodes{
				"n1": argov1.NodeStatus{
					Phase: argov1.NodeSucceeded,
				},
				"n2": argov1.NodeStatus{
					Phase: argov1.NodeError,
				},
			},
		}}
		Expect(WorkflowPhase(wf)).To(Equal(tmv1beta1.PhaseStatusError))
	})
})
