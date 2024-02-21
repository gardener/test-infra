// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("util test", func() {

	Context("url parsing", func() {
		It("github url should return the repo and owner of a repository", func() {
			ghUrl := "https://github.com/gardener/gardener-extensions.git"

			owner, repo, err := ParseRepoURLFromString(ghUrl)
			Expect(err).ToNot(HaveOccurred())
			Expect(repo).To(Equal("gardener-extensions"))
			Expect(owner).To(Equal("gardener"))
		})
		It("urls without owner/repo scheme should not panic", func() {
			ghUrl := "https://example.com/example"

			_, _, err := ParseRepoURLFromString(ghUrl)
			Expect(err).To(HaveOccurred())
		})
		It("domain should parse from grafana url", func() {
			hostname := "grafana.ingress.tm.core.shoot.live.k8s-hana.ondemand.com"
			domain, err := parseDomain(hostname)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).To(Equal("tm.core.shoot.live.k8s-hana.ondemand.com"))
		})
	})

	DescribeTable("IsLastElementOfBucket",
		func(value int, expected bool) {
			Expect(IsLastElementOfBucket(value, 3)).To(Equal(expected))
		},
		Entry("0", 0, false),
		Entry("0", 1, false),
		Entry("2", 2, true),
		Entry("3", 3, false),
		Entry("4", 4, false),
		Entry("5", 5, true),
	)

	DescribeTable("DomainMatches",
		func(expected bool, value string, domains ...string) {
			Expect(DomainMatches(value, domains...)).To(Equal(expected))
		},
		Entry("match exactly", true, "example.com", "example.com"),
		Entry("not match", false, "example.com", "not.com"),
		Entry("match subdomain", true, "sub.example.com", "example.com"),
		Entry("match sub subdomain", true, "sub.sub.example.com", "example.com"),
		Entry("match if one matches", true, "sub.sub.example.com", "not.com", "example.com"),
		Entry("not match if none matches", false, "sub.sub.example.com", "not.com", "not2.com"),
		Entry("not match if none domains provided", false, "sub.sub.example.com"),
	)
})
