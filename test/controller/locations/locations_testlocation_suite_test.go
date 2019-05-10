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

package locations_test

import (
	"context"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Locations TestLocations tests", func() {

	It("should run a test with a TestLocations", func() {
		ctx := context.Background()
		defer ctx.Done()
		tr := resources.GetBasicTestrun(namespace, commitSha)
		tr.Spec.LocationSets = nil
		tr.Spec.TestLocations = []tmv1beta1.TestLocation{
			{
				Type:     tmv1beta1.LocationTypeGit,
				Repo:     "https://github.com/gardener/test-infra.git",
				Revision: commitSha,
			},
		}

		tr, _, err := utils.RunTestrun(ctx, tmClient, tr, tmv1beta1.PhaseStatusSuccess, namespace, maxWaitTime)
		defer utils.DeleteTestrun(tmClient, tr)
		Expect(err).ToNot(HaveOccurred())
	})
})
