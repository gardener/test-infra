// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
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
		for _, comp := range components {
			if comp.Name == "github.com/component-1-1" {
				Expect(comp.SourceRepoURL).To(Equal("github.com/does-not-exist/does-not-exist"))
				Expect(comp.SourceRevision).To(Equal("v1.0.0"))
			}
		}
	},
		Entry("repository from argument", "./testdata/root-component.yaml", "./testdata/repositories/ocm-repo-ctf", ""),
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
