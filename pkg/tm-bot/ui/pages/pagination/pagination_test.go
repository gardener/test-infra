// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package pagination_test

import (
	"net/url"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/pages/pagination"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UI Pagination Suite")
}

var _ = Describe("pagination", func() {

	Context("current page", func() {
		It("should return items 1-3", func() {
			values := url.Values(map[string][]string{
				common.DashboardPaginationFrom: []string{"1"},
				common.DashboardPaginationTo:   []string{"3"},
			})

			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			res, p := pagination.SliceFromValues(list, values)
			Expect(res).To(ConsistOf("a", "b", "c"))
			Expect(p.Current).To(Equal(0))
		})

		It("should return default items per page if no pagination is given", func() {
			values := url.Values(map[string][]string{})
			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			res, p := pagination.SliceFromValues(list, values)
			Expect(res).To(ConsistOf("a", "b", "c", "d", "e"))
			Expect(p.Current).To(Equal(0))
		})

		It("should return items 2-3", func() {
			values := url.Values(map[string][]string{
				common.DashboardPaginationFrom: []string{"2"},
				common.DashboardPaginationTo:   []string{"3"},
			})
			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			res, p := pagination.SliceFromValues(list, values)
			Expect(res).To(ConsistOf("b", "c"))
			Expect(p.Current).To(Equal(1))
		})

		It("should return items 3-4", func() {
			values := url.Values(map[string][]string{
				common.DashboardPaginationFrom: []string{"3"},
				common.DashboardPaginationTo:   []string{"4"},
			})
			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			res, p := pagination.SliceFromValues(list, values)
			Expect(res).To(ConsistOf("c", "d"))
			Expect(p.Current).To(Equal(1))
		})
	})

	Context("pages", func() {
		It("should return 5 pages with a items per page of 1", func() {
			values := url.Values(map[string][]string{
				common.DashboardPaginationFrom: []string{"1"},
				common.DashboardPaginationTo:   []string{"1"},
			})
			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			_, res := pagination.SliceFromValues(list, values)
			Expect(res.Pages).To(HaveLen(5))
			Expect(res.Pages).To(ConsistOf(
				pagination.Page{From: 1, To: 1},
				pagination.Page{From: 2, To: 2},
				pagination.Page{From: 3, To: 3},
				pagination.Page{From: 4, To: 4},
				pagination.Page{From: 5, To: 5},
			))
		})

		It("should return 2 pages with a items per page of 3", func() {
			values := url.Values(map[string][]string{
				common.DashboardPaginationFrom: []string{"1"},
				common.DashboardPaginationTo:   []string{"3"},
			})
			list := pagInterface([]string{"a", "b", "c", "d", "e"})

			_, res := pagination.SliceFromValues(list, values)
			Expect(res.Pages).To(HaveLen(2))
			Expect(res.Pages).To(ConsistOf(
				pagination.Page{From: 1, To: 3},
				pagination.Page{From: 4, To: 5},
			))
		})
	})

})

type pagInterface []string

func (p pagInterface) Len() int           { return len(p) }
func (p pagInterface) Less(i, j int) bool { return p[i] < p[j] }
func (p pagInterface) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p pagInterface) GetPaginatedList(from, to int) pagination.Interface {
	to++
	if to > len(p) {
		return p
	}
	return p[from:to]
}
