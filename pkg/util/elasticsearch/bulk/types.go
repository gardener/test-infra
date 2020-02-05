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

package bulk

// Bulk is the internal representation of a elastic search bulk request.
type Bulk struct {
	Metadata interface{} `json:",inline"`
	Source   []byte      `json:",inline"`
}

// BulkList is a list of bulks
type BulkList []*Bulk

// ESMetadata is the metadata of a bulk document.
type ESMetadata struct {
	Index ESIndex `json:"index,omitempty"`
}

// ESIndex is the elastic search index where the bulk data is stored.
type ESIndex struct {
	Index string `json:"_index,omitempty"`
	Type  string `json:"_type,omitempty"`
}
