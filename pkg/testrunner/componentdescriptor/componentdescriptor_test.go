// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package componentdescriptor

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentDescriptor Suite")
}

var _ = Describe("componentdescriptor test", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		ctx.Done()
	})

	DescribeTable("list the transitive closure of the component", func(cdPath, repoRef, configPath string) {
		opts := func(opts *Options) {
			opts.CfgPath = configPath
		}
		components, err := GetComponents(ctx, logr.Discard(), cdPath, repoRef, opts)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(components)).To(Equal(7))
	},
		Entry("repository from argument", "./testdata/root-component.yaml", "CommonTransportFormat::testdata/repositories/ocm-repo-ctf", ""),
		Entry("repository from ocm config file", "./testdata/root-component.yaml", "", "./testdata/ocmconfigs/.ocmconfig-single-repo"),
		Entry("mulitple repositories from ocm config file", "./testdata/root-component.yaml", "", "./testdata/ocmconfigs/.ocmconfig-multi-repo"),
	)

	DescribeTable("fail if referenced component cannot be found", func(cdPath, repoRef, configPath string) {
		opts := func(opts *Options) {
			opts.CfgPath = configPath
		}
		components, err := GetComponents(ctx, logr.Discard(), cdPath, repoRef, opts)
		Expect(err).To(HaveOccurred())
		Expect(components).To(BeNil())
	},
		Entry("referenced component does not exist in registry", "./testdata/root-component-with-invalid-ref.yaml", "./testdata/repositories/ocm-repo-ctf", ""),
		Entry("repository specified in argument cannot be found", "./testdata/root-component.yaml", "./testdata/ocmconfigs/non-existing-repo", ""),
		Entry("repository specified in ocm config file cannot be found", "./testdata/root-component.yaml", "", "./testdata/ocmconfigs/.ocmconfig-non-existing-repo"),
	)
})
