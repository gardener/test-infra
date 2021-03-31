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

package precompute

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
)

type BulkResponse struct {
	ErrorsOccurred bool       `json:"errors"`
	Items          []BulkItem `json:"items,omitempty"`
}

type BulkItem struct {
	Index BulkItemIndex `json:"index"`
}

type BulkItemIndex struct {
	Index      string             `json:"_index"`
	Type       string             `json:"_type"`
	ID         string             `json:"_id"`
	HTTPStatus int                `json:"status"`
	Error      BulkItemIndexError `json:"error"`
}

type BulkItemIndexError struct {
	Type   string `json:"type"`
	Reason string `json:"reason,omitempty"`
}

type QueryResponse struct {
	ScrollID string `json:"_scroll_id,omitempty"`
	Hits     Hits   `json:"hits,omitempty"`
}

type Hits struct {
	Total   Total    `json:"total,omitempty"`
	Results []Result `json:"hits,omitempty"`
}

type Total struct {
	Value int `json:"value"`
}

type Result struct {
	Index       string               `json:"_index"`
	DocID       string               `json:"_id"`
	StepSummary metadata.StepSummary `json:"_source,omitempty"`
}

func BuildBulkUpdateQuery(items []Result) (path string, payload io.Reader, err error) {
	path = "/_bulk"
	var buffer strings.Builder
	for _, item := range items {
		buffer.WriteString(fmt.Sprintf("{\"index\":{\"_index\":\"%s\",\"_id\":\"%s\"}}\n", item.Index, item.DocID))
		bytes, err := json.Marshal(item.StepSummary)
		if err != nil {
			return "", nil, err
		}
		buffer.Write(bytes)
		buffer.WriteString("\n")
	}

	payload = strings.NewReader(buffer.String())
	return
}

func BuildScrollQueryInitial(index string, pageSize int) (path string, payload io.Reader) {
	path = fmt.Sprintf("/%s/_search?scroll=1m", index)

	// default paging size
	if pageSize == 0 {
		pageSize = 100
	}

	// add a filter (or add as json object to the must array) if you want to restrict the dataset for experimenting/debugging
	// "range": {
	//    "tm.tr.startTime": {
	//        "gte": "2020-09-30T16:45:24.249Z",
	//        "lte": "2020-10-01T16:45:24.249Z",
	//        "format": "strict_date_optional_time"
	//    }
	// }
	query := `
		{
			"size": %d,
			"query": {
				"bool": {
					"must": [],
					"filter": [
						{
							"match_all": {}
						},
						{
							"match_phrase": {
								"type.keyword": "teststep"
							}
						}
					],
					"should": [],
					"must_not": []
				}
			}
		}`
	query = fmt.Sprintf(query, pageSize)

	payload = strings.NewReader(query)
	return
}

func BuildScrollQueryNextPage(scrollID string) (path string, payload io.Reader) {
	path = "/_search/scroll"
	query := `
		{
			"scroll": "1m",
			"scroll_id": "%s"
		}`
	query = fmt.Sprintf(query, scrollID)
	payload = strings.NewReader(query)
	return
}
