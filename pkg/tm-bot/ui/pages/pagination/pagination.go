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
	"errors"
	"net/url"
	"sort"
	"strconv"

	"github.com/gardener/test-infra/pkg/common"
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

// defaultItems that are displayed on one page - 1
const defaultItemsPerPage = 49

// SliceFromValues slices the given list into the values gathered form the url values
func SliceFromValues(list Interface, values url.Values) (Interface, Pages) {
	sort.Sort(list)

	var (
		pages = Pages{Pages: []Page{}, Current: 0, ItemCount: list.Len()}
		itemsPerPage = defaultItemsPerPage
	)

	from, to, err := parseRange(values)
	if err == nil {
		itemsPerPage = to - from
	}
	if from <= 0 {
		from = 1
	}
	if to <= 0 {
		to = from + itemsPerPage
	}



	if list.Len() < itemsPerPage {
		return list, Pages{Pages: []Page{}, Current: 0, ItemCount: list.Len()}
	}

	// get following pages
	page := Page{
		From: from,
		To:   to,
	}

	pages.Pages = previousPages(page, 1, itemsPerPage)
	pages.Pages = append(pages.Pages, page)
	pages.Current = len(pages.Pages) - 1

	pages.Pages = append(pages.Pages, nextPages(page, pages.ItemCount, itemsPerPage)...)

	// convert page number to index
	from = from - 1
	to = to - 1

	return list.GetPaginatedList(from, to), pages
}

func nextPages(base Page, max, itemsPerPage int) []Page {
	pages := make([]Page, 0)
	page := getNextPage(base.To+1, max, itemsPerPage)
	for page != nil {
		pages = append(pages, *page)
		page = getNextPage(page.To+1, max, itemsPerPage)
	}
	return pages
}

func getNextPage(start, max, itemsPerPage int) *Page {
	if start > max {
		return nil
	}
	end := start + itemsPerPage
	if end > max {
		end = max
	}
	return &Page{
		From: start,
		To:   end,
	}
}

func previousPages(base Page, min, itemsPerPage int) []Page {
	if base.From <= min {
		return nil
	}
	pages := make([]Page, 0)
	page := getPreviousPage(base.From-1, min, itemsPerPage)
	for page != nil {
		pages = append([]Page{*page}, pages...)
		page = getPreviousPage(page.From-1, min, itemsPerPage)
	}
	return pages
}

// getPreviousPage returns the next previous page from the end
func getPreviousPage(end, min, itemsPerPage int) *Page {
	if end < min {
		return nil
	}
	start := end - itemsPerPage
	if start < min {
		start = min
	}
	return &Page{
		From: start,
		To:   end,
	}
}

func parseRange(values url.Values) (from, to int, err error) {
	fromString, ok := values[common.DashboardPaginationFrom]
	if !ok {
		err = errors.New("from not found")
		return
	}

	toString, ok := values[common.DashboardPaginationTo]
	if !ok {
		err = errors.New("to not found")
		return
	}

	from, err = strconv.Atoi(fromString[0])
	if err != nil {
		return
	}
	if from <= 0 {
		err = errors.New("index out of range")
		return
	}

	to, err = strconv.Atoi(toString[0])
	if err != nil {
		return
	}
	if to <= 0 {
		err = errors.New("index out of range")
		return
	}

	if to < from {
		to = 0
		from = 0
		err = errors.New("to greater than from")
	}

	return
}
