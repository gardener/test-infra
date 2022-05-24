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

package ghcache

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gregjones/httpcache"
)

// cache extends the default http caching by adding caching behavior to errornous responses like 404.
// Note that this will reduce the correctness of all responses.
type cache struct {
	delegate      http.RoundTripper
	maxAgeSeconds int
	httpCache     httpcache.Cache
}

var (
	// parallelRequests stores a Condition for each ongoing http request
	parallelRequests     = make(map[string]*sync.Cond)
	parallelRequestsLock sync.Mutex
)

func (c *cache) RoundTrip(req *http.Request) (*http.Response, error) {

	// avoid parallel requests in case the response is not yet cached
	// only GET and HEAD requests are cachable, thus we care only about those here
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return c.roundTrip(req)
	}

	isCached, err := httpcache.CachedResponse(c.httpCache, req)
	if err != nil {
		return nil, err
	}

	// in case of a cache miss, we check for parallel requests
	// if there is an ongoing request, wait on the Condition and proceed to get the response via the cache
	// if we are the first, a new Condition is added to the map and the request will be sent
	// once the request returns, remove the condition and then broadcast to awake waiting routines
	if isCached == nil {
		key := req.URL.String()

		parallelRequestsLock.Lock()
		activeRequest, ok := parallelRequests[key]
		if !ok {
			// no active request yet, we are the first!
			activeRequest = sync.NewCond(&sync.Mutex{})
			parallelRequests[key] = activeRequest
			parallelRequestsLock.Unlock()

			// run the actual request via the cache
			res, err := c.roundTrip(req)

			// remove the Condition for this key so no other requests can subscribe here
			parallelRequestsLock.Lock()
			delete(parallelRequests, key)
			parallelRequestsLock.Unlock()

			// lock the condition and wake up all other routines waiting
			activeRequest.L.Lock()
			activeRequest.Broadcast()
			activeRequest.L.Unlock()

			return res, err
		} else {
			// first, lock the Condition and then unlock the map access
			// this ensures Broadcast() will be sent to all subscribers
			activeRequest.L.Lock()
			parallelRequestsLock.Unlock()

			// being awaken from Wait() includes acquiring the lock, hence unlock is called
			activeRequest.Wait()
			activeRequest.L.Unlock()
		}
	}

	// a cache hit can be expected now
	return c.roundTrip(req)

}

func (c *cache) roundTrip(req *http.Request) (*http.Response, error) {
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
