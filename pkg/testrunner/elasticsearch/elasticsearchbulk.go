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

package elasticsearch

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/util"

	log "github.com/sirupsen/logrus"
)

var newLine = []byte("\n")

// Marshal creates a elastic search bulk json of its metadata and sources.
func (b *Bulk) Marshal() (*bytes.Buffer, error) {
	meta, err := util.MarshalNoHTMLEscape(b.Metadata)
	if err != nil {
		return nil, fmt.Errorf("Cannot marshal ElasticsearchBulk %s", err.Error())
	}

	buf := bytes.NewBuffer([]byte{})

	for _, source := range b.Sources {
		buf.Write(meta)
		buf.Write(source)
	}

	return buf, nil
}

// ParseExportedFiles reads jsondocuments line by line from an expected file where multiple jsons are separated by newline.
func ParseExportedFiles(name string, stepMeta interface{}, docs []byte) []byte {

	// first try to parse document as normal json.
	var jsonBody map[string]interface{}
	err := json.Unmarshal(docs, &jsonBody)
	if err == nil {
		jsonBody["tm"] = stepMeta
		patchedDoc, err := json.Marshal(jsonBody)
		if err != nil {
			log.Warnf("Cannot mashal exported json with metadata from %s", name)
			return []byte{}
		}
		bulk := Bulk{
			Sources: [][]byte{patchedDoc},
			Metadata: ESMetadata{
				Index: ESIndex{
					Index: name,
					Type:  "_doc",
				},
			},
		}

		docBuf, err := bulk.Marshal()
		if err != nil {
			log.Warnf("Cannot use exported json from %s", name)
			return []byte{}
		}
		return docBuf.Bytes()
	}

	// if the document is not in json format try to parse it as bulk format
	return parseExportedBulkFormat(name, stepMeta, docs)
}

func parseExportedBulkFormat(name string, stepMeta interface{}, docs []byte) []byte {
	bulks := bytes.NewBuffer([]byte{})
	var meta map[string]interface{}

	scanner := bufio.NewScanner(bytes.NewReader(docs))
	for scanner.Scan() {
		doc := scanner.Bytes()
		var jsonBody map[string]interface{}
		err := json.Unmarshal(doc, &jsonBody)
		if err != nil {
			log.Errorf("Cannot unmarshal document %s", err.Error())
			continue
		}

		if jsonBody["index"] != nil {
			meta = jsonBody
			meta["index"].(map[string]interface{})["_index"] = fmt.Sprintf("tm-%s", meta["index"].(map[string]interface{})["_index"])
			continue
		}

		jsonBody["tm"] = stepMeta
		patchedDoc, err := util.MarshalNoHTMLEscape(jsonBody) // json.Marshal(jsonBody)
		if err != nil {
			log.Errorf("Cannot marshal artifact %s", err.Error())
			continue
		}

		bulk := Bulk{
			Sources:  [][]byte{patchedDoc},
			Metadata: meta,
		}
		if meta == nil {
			bulk.Metadata = ESMetadata{
				Index: ESIndex{
					Index: name,
					Type:  "_doc",
				},
			}
		}

		buf, err := bulk.Marshal()
		if err != nil {
			log.Debugf("Cannot unmarshal %s", err.Error())
			meta = nil
			continue
		}
		bulks.Write(buf.Bytes())
		meta = nil
	}
	if err := scanner.Err(); err != nil {
		log.Warnf("Error reading json: %s", err.Error())
	}

	return bulks.Bytes()
}
