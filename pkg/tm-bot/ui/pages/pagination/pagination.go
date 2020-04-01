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

package pagination

import (
	"github.com/gardener/test-infra/pkg/common"
	"net/url"
	"sort"
	"strconv"
)

type Interface interface {
	sort.Interface
	GetPaginatedList(from, to int) Interface
}

type Pages struct {
	Pages     []Page
	Current   int
	ItemCount int
}

type Page struct {
	From int
	To   int
}

const itemsPerPage = 50

// SliceFromValues slices the given list into the values gathered form the url values
func SliceFromValues(list Interface, values url.Values) (Interface, Pages) {
	sort.Sort(list)
	if list.Len() < itemsPerPage {
		return list, Pages{Pages: []Page{}, Current: 0, ItemCount: list.Len()}
	}

	indexLength := list.Len() - 1

	pages := Pages{Pages: []Page{}, Current: 0, ItemCount: list.Len()}
	for i := 0; i < (pages.ItemCount / itemsPerPage); i++ {
		from := i * itemsPerPage
		to := from + itemsPerPage - 1
		if to > indexLength {
			to = indexLength
		}
		pages.Pages = append(pages.Pages, Page{
			From: from,
			To:   to,
		})
	}

	fromString, ok := values[common.DashboardPaginationFrom]
	if !ok {
		return list.GetPaginatedList(0, itemsPerPage), pages
	}

	toString, ok := values[common.DashboardPaginationTo]
	if !ok {
		return list.GetPaginatedList(0, itemsPerPage), pages
	}

	from, err := strconv.Atoi(fromString[0])
	if err != nil {
		return list.GetPaginatedList(0, itemsPerPage), pages
	}
	to, err := strconv.Atoi(toString[0])
	if err != nil {
		return list.GetPaginatedList(0, itemsPerPage), pages
	}

	pages.Current = from / itemsPerPage

	return list.GetPaginatedList(from, to), pages
}
