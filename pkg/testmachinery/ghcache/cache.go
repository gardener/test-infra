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
