// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ghcache

import (
	"fmt"
	"net/http"
)

// cache extends the default http caching by adding caching behavior to errornous responses like 404.
// Note that this will reduce the correctness of all responses.
type cache struct {
	delegate      http.RoundTripper
	maxAgeSeconds int
}

func (c *cache) RoundTrip(req *http.Request) (*http.Response, error) {
	if c.maxAgeSeconds <= 0 {
		return c.delegate.RoundTrip(req)
	}
	res, err := c.delegate.RoundTrip(req)
	if err != nil {
		return res, err
	}
	if req.Method == http.MethodGet {
		if res.StatusCode == http.StatusNotFound {
			res.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", c.maxAgeSeconds))
		}
	}
	return res, err
}
