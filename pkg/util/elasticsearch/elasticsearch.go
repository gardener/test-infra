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

package elasticsearch

import (
	"bytes"
	"encoding/json"
	defaulterrors "errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// Client defines an interface to interact with an elastic search instance
type Client interface {
	Request(httpMethod, path string, payload io.Reader) ([]byte, error)
	Bulk([]byte) error
	BulkFromFile(file string) error
}

type client struct {
	*http.Client

	endpoint string
	username string
	password string
}

func NewClient(cfg Config) (Client, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = ""

	if cfg.Username == "" {
		return nil, errors.New("elasticsearch username has to be defined")
	}
	if cfg.Password == "" {
		return nil, errors.New("elasticsearch password has to be defined")
	}

	return &client{
		Client:   http.DefaultClient,
		endpoint: u.String(),
		username: cfg.Username,
		password: cfg.Password,
	}, nil
}

func (c *client) Request(httpMethod, rawPath string, payload io.Reader) ([]byte, error) {
	esURL, err := c.parseUrl(rawPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(httpMethod, esURL, payload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Add("Content-Type", "application/x-ndjson")
	req.Header.Add("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to do request to %s", esURL)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, errors.Errorf("request %s returned status code %d", esURL, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read response body")
	}
	return body, err
}

func (c *client) Bulk(data []byte) error {
	body, err := c.Request(http.MethodPost, "_bulk", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	bulkRes := &BulkResponse{}
	if err := json.Unmarshal(body, bulkRes); err != nil {
		return errors.Wrap(err, "unable to unmarshal bulk response")
	}

	if bulkRes.Errors {
		if len(bulkRes.Items) == 0 {
			return errors.New("elastic search returned an error")
		}
		var allErrors *multierror.Error
		for _, action := range bulkRes.Items {
			for _, item := range action {
				if item.Status < 200 || item.Status > 299 {
					allErrors = multierror.Append(allErrors, defaulterrors.New(item.Error))
				}
			}
		}
		return allErrors
	}

	return nil
}

func (c *client) BulkFromFile(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return c.Bulk(data)
}

func (c *client) parseUrl(rawPath string) (string, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, rawPath)
	return u.String(), nil
}

// BulkResponse is the response that is returned by elastic search when doing a bulk request
type BulkResponse struct {
	Took   int                           `json:"took"`
	Errors bool                          `json:"errors"`
	Items  []map[string]BulkResponseItem `json:"items"`
}

// BulkResponseItem is response of one document from a bulk request
type BulkResponseItem struct {
	Index  string `json:"_index"`
	Type   string `json:"_type"`
	ID     string `json:"_id"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}
