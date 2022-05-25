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
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	"github.com/gregjones/httpcache"
)

var (
	_ http.RoundTripper = &rateLimitControl{}
	// parallelRequests stores a Condition for each ongoing http request
	parallelRequests     = make(map[string]*sync.Cond)
	parallelRequestsLock sync.Mutex
)

type rateLimitControl struct {
	log      logr.Logger
	delegate http.RoundTripper
	cache    httpcache.Cache
}

func (l *rateLimitControl) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.URL.String()
	l.log.V(5).Info("Starting RoundTrip", "key", key)
	// avoid parallel requests in case the response is not yet cached
	// only GET and HEAD requests are cachable, thus we care only about those here
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		isCached, err := httpcache.CachedResponse(l.cache, req)
		if err != nil {
			return nil, err
		}

		// in case of a cache miss, we check for parallel requests
		// if there is an ongoing request, wait on the Condition and proceed to get the response via the cache
		// if we are the first, a new Condition is added to the map and the request will be sent
		// once the request returns, remove the condition and then broadcast to awake waiting routines
		if isCached == nil {
			l.log.V(5).Info("Request not yet cached", "key", key)
			parallelRequestsLock.Lock()
			activeRequest, ok := parallelRequests[key]
			if !ok {
				// no active request yet, we are the first!
				l.log.V(5).Info("First request", "key", key)
				activeRequest = sync.NewCond(&sync.Mutex{})
				parallelRequests[key] = activeRequest
				parallelRequestsLock.Unlock()

				l.log.V(5).Info("First request roundtrip", "key", key)
				// run the actual request via the cache
				res, err := l.delegate.RoundTrip(req)
				// remove the Condition for this key so no other requests can subscribe here
				parallelRequestsLock.Lock()
				delete(parallelRequests, key)
				parallelRequestsLock.Unlock()

				// lock the condition and wake up all other routines waiting
				activeRequest.L.Lock()
				isCached, err = httpcache.CachedResponse(l.cache, req)
				if isCached == nil {
					l.log.V(5).Info("Not yet cached", "key", key)
				} else {
					l.log.V(5).Info("Cache entry is present", "key", key)
				}
				l.log.V(5).Info("Wake up waiting requests", "key", key)
				activeRequest.Broadcast()
				activeRequest.L.Unlock()

				l.logRateLimitInfo(req, res)
				return res, err
			} else {
				// first, lock the Condition and then unlock the map access
				// this ensures Broadcast() will be sent to all subscribers
				l.log.V(5).Info("Similar request already in-flight", "key", key)
				activeRequest.L.Lock()
				parallelRequestsLock.Unlock()

				// being awakened from Wait() includes acquiring the lock, hence unlock is called
				activeRequest.Wait()
				activeRequest.L.Unlock()
				l.log.V(5).Info("Awaken", "key", key)

			}
		}

	}

	l.log.V(5).Info("Roundtrip with cache", "key", key)
	// a cache hit for GET/HEAD requests is expected now
	resp, err := l.delegate.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	l.logRateLimitInfo(req, resp)
	return resp, nil
}

func (l *rateLimitControl) logRateLimitInfo(req *http.Request, resp *http.Response) {
	total := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	hit := resp.Header.Get(httpcache.XFromCache)
	l.log.V(5).Info("GitHub rate limit", "hit", hit, "total", total, "remaining", remaining, "url", req.URL.String())
	if remaining == "0" {
		l.log.Error(nil, "GitHub request limit exceeded", "total", total, "remaining", remaining, "url", req.URL.String())
	}
}
