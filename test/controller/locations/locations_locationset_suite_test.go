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
	"github.com/gardener/test-infra/pkg/testmachinery"
	"os"
	"testing"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var (
	maxWaitTime int64 = 300
)

var (
	commitSha string
	namespace string
	tmClient  kubernetes.Interface
)

func TestTestflowWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun locations Integration Test Suite")
}

var _ = Describe("Locations LocationSets tests", func() {

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")

		tmClient, err = kubernetes.NewClientFromFile("", tmKubeconfig, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.WaitForClusterReadiness(tmClient, namespace, maxWaitTime)).ToNot(HaveOccurred())
	})

	Context("LocationSets", func() {
		It("should run a test with one location set and a specific default location", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)

			tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

		})

		It("should use the first location set as default", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: commitSha,
						},
					},
				},
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
			}

			tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should use the second location set as default", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
				{
					Name:    "default",
					Default: true,
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: commitSha,
						},
					},
				},
			}

			tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("flow step", func() {
		It("should use a specific location set", func() {
			ctx := context.Background()
			defer ctx.Done()

			setName := "default"
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
				{
					Name: setName,
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: commitSha,
						},
					},
				},
			}
			tr.Spec.TestFlow[0][0].Location = &setName

			tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
