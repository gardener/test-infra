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

package util_test

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("testrun util test", func() {

	Context("TestrunStatusPhase", func() {
		It("should return success when the testrun was successfull", func() {
			tr := &v1beta1.Testrun{Status: v1beta1.TestrunStatus{Phase: v1beta1.RunPhaseSuccess}}
			Expect(util.TestrunStatusPhase(tr)).To(Equal(v1beta1.RunPhaseSuccess))
		})

		It("should return success even if a system component failed", func() {
			tr := &v1beta1.Testrun{Status: v1beta1.TestrunStatus{
				Phase: v1beta1.RunPhaseError,
				Steps: []*v1beta1.StepStatus{
					newStepStatus(v1beta1.StepPhaseSuccess, false),
					newStepStatus(v1beta1.StepPhaseError, true),
				},
			}}
			Expect(util.TestrunStatusPhase(tr)).To(Equal(v1beta1.RunPhaseSuccess))
		})

		It("should return error if one non system step fails", func() {
			tr := &v1beta1.Testrun{Status: v1beta1.TestrunStatus{
				Phase: v1beta1.RunPhaseError,
				Steps: []*v1beta1.StepStatus{
					newStepStatus(v1beta1.StepPhaseSuccess, true),
					newStepStatus(v1beta1.StepPhaseError, false),
				},
			}}
			Expect(util.TestrunStatusPhase(tr)).To(Equal(v1beta1.RunPhaseError))
		})

		It("should return the testrun state if all steps are in init state", func() {
			tr := &v1beta1.Testrun{Status: v1beta1.TestrunStatus{
				Phase: v1beta1.RunPhaseError,
				Steps: []*v1beta1.StepStatus{
					newStepStatus(v1beta1.StepPhaseInit, true),
					newStepStatus(v1beta1.StepPhaseInit, false),
				},
			}}
			Expect(util.TestrunStatusPhase(tr)).To(Equal(v1beta1.RunPhaseError))
		})

		It("should return the testrun state if all steps are in skipped state", func() {
			tr := &v1beta1.Testrun{Status: v1beta1.TestrunStatus{
				Phase: v1beta1.RunPhaseError,
				Steps: []*v1beta1.StepStatus{
					newStepStatus(v1beta1.StepPhaseSkipped, true),
					newStepStatus(v1beta1.StepPhaseSkipped, false),
				},
			}}
			Expect(util.TestrunStatusPhase(tr)).To(Equal(v1beta1.RunPhaseError))
		})
	})
})

func newStepStatus(phase v1alpha1.NodePhase, system bool) *v1beta1.StepStatus {
	step := &v1beta1.StepStatus{
		Phase:       phase,
		Annotations: map[string]string{},
	}

	if system {
		step.Annotations[common.AnnotationSystemStep] = "true"
	}

	return step
}
