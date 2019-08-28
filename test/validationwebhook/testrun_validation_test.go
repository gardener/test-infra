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

package validationwebhook_test

import (
	"context"
	"github.com/gardener/test-infra/pkg/util/strconf"
	"github.com/gardener/test-infra/test/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/test-infra/test/resources"
)

var _ = Describe("Testrun validation tests", func() {

	Context("Metadata", func() {
		It("should reject when name contains '.'", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "integration.testdef",
					},
				},
			}

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("name must not contain '.'"))
		})
	})

	Context("TestLocations", func() {
		It("should reject when no locations are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}
			tr.Spec.LocationSets = nil

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("no location for TestDefinitions defined"))
		})

		It("should reject when a local location is defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.LocationSets = nil
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}
			tr.Spec.TestLocations = append(tr.Spec.TestLocations, tmv1beta1.TestLocation{
				Type: tmv1beta1.LocationTypeLocal,
			})

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("Local testDefinition locations are only available in insecure mode"))
		})
	})

	Context("Testflow", func() {
		It("should reject when no locations can be found", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{}

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("No testdefinitions found"))
		})

		It("should reject when a no locations for only one label in the testrun can be found", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "NoTestDefsFoundLabel",
					},
				},
			}

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("No testdefinitions found"))
		})
	})

	Context("Kubeconfigs", func() {
		It("should reject when a invalid kubeconfig is provided", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.Kubeconfigs.Gardener = strconf.FromString("dGVzdGluZwo=")

			err := operation.Client().Client().Create(ctx, tr)
			if err == nil {
				defer utils.DeleteTestrun(operation.Client(), tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("Cannot build config"))
		})
	})

	Context("OnExit", func() {
		It("should accept when no steps are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

			err := operation.Client().Client().Create(ctx, tr)
			defer utils.DeleteTestrun(operation.Client(), tr)

			Expect(err).ToNot(HaveOccurred())

			err = operation.Client().Client().Get(ctx, client.ObjectKey{Namespace: operation.TestNamespace(), Name: tr.Name}, tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
