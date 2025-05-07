// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/util"
)

// 50 mb
const maxBufferSize = 50 * 1024 * 1024

// Marshal creates an elastic search bulk json of its metadata and sources and returns a list of bulk files with a max size of 50mb
func (b *Bulk) Marshal() ([]byte, error) {
	meta, err := util.MarshalNoHTMLEscape(b.Metadata)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal ElasticsearchBulk %s", err.Error())
	}

	buf := bytes.NewBuffer([]byte{})
	buf.Write(meta)
	buf.Write(b.Source)

	return buf.Bytes(), nil
}

// NewList creates a list of Bulks with the same metadata
func NewList(meta interface{}, sources [][]byte) BulkList {
	bulks := make([]*Bulk, 0)
	for _, source := range sources {
		bulks = append(bulks, &Bulk{
			Metadata: meta,
			Source:   source,
		})
	}

	return bulks
}

func (l BulkList) Marshal() ([][]byte, error) {
	content := [][]byte{}

	buffer := bytes.NewBuffer([]byte{})
	for _, bulk := range l {
		data, err := bulk.Marshal()
		if err != nil {
			return nil, err
		}

		if (buffer.Len() + len(data)) >= maxBufferSize {
			content = append(content, buffer.Bytes())
			buffer = bytes.NewBuffer([]byte{})
		}
		buffer.Write(data)
	}
	content = append(content, buffer.Bytes())

	return content, nil
}

// ParseExportedFiles reads jsondocuments line by line from an expected file where multiple jsons are separated by newline.
func ParseExportedFiles(log logr.Logger, name string, stepMeta interface{}, docs []byte) BulkList {
	// first try to parse document as normal json.
	var jsonBody map[string]interface{}
	err := json.Unmarshal(docs, &jsonBody)
	if err == nil {
		jsonBody["tm"] = stepMeta
		patchedDoc, err := util.MarshalNoHTMLEscape(jsonBody)
		if err != nil {
			log.Info("cannot marshal exported json with metadata", "file", name)
			return make(BulkList, 0)
		}
		bulk := &Bulk{
			Source: patchedDoc,
			Metadata: ESMetadata{
				Index: ESIndex{
					Index: fmt.Sprintf("tm-%s", name),
				},
			},
		}
		return []*Bulk{bulk}
	}

	// if the document is not in json format try to parse it as newline delimited json
	return parseExportedBulkFormat(log, name, stepMeta, docs)
}

func parseExportedBulkFormat(log logr.Logger, name string, stepMeta interface{}, docs []byte) BulkList {
	bulks := make(BulkList, 0)
	var meta map[string]interface{}
	for doc := range util.ReadLines(docs) {
		var jsonBody map[string]interface{}
		err := json.Unmarshal(doc, &jsonBody)
		if err != nil {
			log.V(5).Info(fmt.Sprintf("cannot unmarshal document %s", err.Error()))
			continue
		}
		// if a bulk is defined we preifx the index with tm- to ensure it does not collide with any other index
		if jsonBody["index"] != nil {
			meta = jsonBody
			meta["index"].(map[string]interface{})["_index"] = fmt.Sprintf("tm-%s", meta["index"].(map[string]interface{})["_index"])
			continue
		}
		// construct own bulk with index = tm-<testdef name>
		jsonBody["tm"] = stepMeta
		patchedDoc, err := util.MarshalNoHTMLEscape(jsonBody) // json.Marshal(jsonBody)
		if err != nil {
			log.V(3).Info(fmt.Sprintf("cannot marshal artifact %s", err.Error()))
			continue
		}
		bulk := &Bulk{
			Source:   patchedDoc,
			Metadata: meta,
		}
		if meta == nil {
			bulk.Metadata = ESMetadata{
				Index: ESIndex{
					Index: fmt.Sprintf("tm-%s", name),
				},
			}
		}

		bulks = append(bulks, bulk)
		meta = nil
	}
	return bulks
}
