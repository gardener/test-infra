// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

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
}
