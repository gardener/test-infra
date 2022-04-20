// Copyright 2022 Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package result

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("summary-poster", func() {

	It("should render summary with single result", func() {
		run := &testrunner.Run{
			Info: nil,
			Testrun: &tmv1beta1.Testrun{
				Status: tmv1beta1.TestrunStatus{
					Phase: tmv1beta1.RunPhaseSuccess,
				},
			},
			Metadata: &metadata.Metadata{
				CloudProvider: "foo",
			},
			Error:      nil,
			Rerenderer: nil,
		}
		var runs testrunner.RunList
		runs = append(runs, run)
		items := parseTestrunsToTableItems(runs)
		Expect(items).To(HaveLen(1))
		slack, err := util.RenderTableForSlack(nil, items)
		Expect(err).ToNot(HaveOccurred())
		Expect(slack).To(ContainSubstring(string(util.StatusSymbolSuccess)))
		Expect(slack).ToNot(ContainSubstring(string(util.StatusSymbolNA)))
	})

	It("should render summary with two results for the same line", func() {
		var runs testrunner.RunList
		run := &testrunner.Run{
			Info: nil,
			Testrun: &tmv1beta1.Testrun{
				Status: tmv1beta1.TestrunStatus{
					Phase: tmv1beta1.RunPhaseSuccess,
				},
			},
			Metadata: &metadata.Metadata{
				CloudProvider: "foo",
			},
			Error:      nil,
			Rerenderer: nil,
		}
		runs = append(runs, run)
		run = &testrunner.Run{
			Info: nil,
			Testrun: &tmv1beta1.Testrun{
				Status: tmv1beta1.TestrunStatus{
					Phase: tmv1beta1.RunPhaseFailed,
				},
			},
			Metadata: &metadata.Metadata{
				CloudProvider: "foo",
			},
			Error:      nil,
			Rerenderer: nil,
		}
		runs = append(runs, run)
		items := parseTestrunsToTableItems(runs)
		Expect(items).To(HaveLen(2))
		slack, err := util.RenderTableForSlack(nil, items)
		Expect(err).ToNot(HaveOccurred())
		Expect(slack).To(ContainSubstring(string(util.StatusSymbolSuccess) + util.SymbolOffset + string(util.StatusSymbolFailure)))
		Expect(slack).ToNot(ContainSubstring(string(util.StatusSymbolNA)))
	})
})
