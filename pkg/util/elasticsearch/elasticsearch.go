// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/gardener/test-infra/pkg/apis/config"
)

// Client defines an interface to interact with an elastic search instance
type Client interface {
	Request(httpMethod, path string, payload io.Reader) ([]byte, error)
	RequestWithCtx(ctx context.Context, httpMethod, path string, payload io.Reader) ([]byte, error)
	Bulk([]byte) error
	BulkFromFile(file string) error
}

type client struct {
	*http.Client

	endpoint string
	username string
	password string
}

func NewClient(cfg config.ElasticSearch) (Client, error) {
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
	ctx := context.Background()
	defer ctx.Done()
	return c.RequestWithCtx(ctx, httpMethod, rawPath, payload)
}

func (c *client) RequestWithCtx(ctx context.Context, httpMethod, rawPath string, payload io.Reader) ([]byte, error) {
	esURL, err := c.parseUrlNoEscape(rawPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, httpMethod, esURL, payload)
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
		errorResponse, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("request %s returned status code %d with body %s", esURL, res.StatusCode, errorResponse)
	}

	body, err := io.ReadAll(res.Body)
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
		items := make([]map[string]BulkResponseItem, 0)
		if err := json.Unmarshal(bulkRes.Items, &items); err != nil {
			return errors.Wrap(err, "unable to parse bulk items")
		}
		if len(items) == 0 {
			return errors.New("elastic search returned an error")
		}
		var allErrors *multierror.Error
		for _, action := range items {
			for _, item := range action {
				if item.Status < 200 || item.Status > 299 {
					allErrors = multierror.Append(allErrors, fmt.Errorf("%#v", item.Error))
				}
			}
		}
		return allErrors
	}

	return nil
}

func (c *client) BulkFromFile(file string) error {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}
	return c.Bulk(data)
}

func (c *client) parseUrlNoEscape(rawPath string) (string, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, rawPath)
	var result string
	if u.Path == "" {
		result = u.Scheme + "://" + u.Host
	} else {
		result = u.Scheme + "://" + path.Join(u.Host, u.Path)
	}
	if u.RawQuery != "" {
		result += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		result += "#" + u.Fragment
	}
	return result, nil
}

// BulkResponse is the response that is returned by elastic search when doing a bulk request
type BulkResponse struct {
	Took   int             `json:"took"`
	Errors bool            `json:"errors"`
	Items  json.RawMessage `json:"items"`
}

// BulkResponseItem is response of one document from a bulk request
type BulkResponseItem struct {
	Index  string      `json:"_index"`
	Type   string      `json:"_type"`
	ID     string      `json:"_id"`
	Status int         `json:"status"`
	Error  interface{} `json:"error"`
}
