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
package componentdescriptor

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	mock_ociclient "github.com/gardener/component-cli/ociclient/mock"
	"github.com/gardener/component-cli/pkg/commands/constants"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentDescriptor Suite")
}

var _ = Describe("componentdescriptor test", func() {

	var (
		ctx           context.Context
		mockCtrl      *gomock.Controller
		mockOCIClient *mock_ociclient.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, "./testdata")).To(Succeed())
		mockCtrl = gomock.NewController(GinkgoT())
		mockOCIClient = mock_ociclient.NewMockClient(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	It("Should parse a component descriptor and return 2 dependencies", func() {
		input, err := ioutil.ReadFile("./testdata/component_descriptor_1")
		Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/component_descriptor_1")

		dependencies, err := GetComponents(ctx, log.NullLogger{}, mockOCIClient, input)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(dependencies)).To(Equal(2))
	})

	It("Should parse a component descriptor and ignore duplicates", func() {
		input, err := ioutil.ReadFile("./testdata/registry.example/example.com/repo1-0.17.0")
		Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/component_descriptor_2")

		result := []*Component{
			{
				Name:    "example.com/repo1",
				Version: "0.17.0",
			},
			{
				Name:    "example.com/repo2",
				Version: "1.27.0",
			},
			{
				Name:    "example.com/repo3",
				Version: "1.30.0",
			},
		}

		dependencies, err := GetComponents(ctx, log.NullLogger{}, mockOCIClient, input)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(dependencies)).To(Equal(3), "There should be 3 dependencies")
		Expect(dependencies).To(Equal(ComponentList(result)))
	})
})
