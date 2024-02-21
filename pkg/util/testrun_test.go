// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
