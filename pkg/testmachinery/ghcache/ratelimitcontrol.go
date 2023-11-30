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
	"context"
	"net/http"
	"sync"
	"time"

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
				l.log.V(5).Info("No other requests in-flight, performing full roundtrip to fill cache", "key", key)
				activeRequest = sync.NewCond(&sync.Mutex{})
				parallelRequests[key] = activeRequest
				parallelRequestsLock.Unlock()

				// run the actual request via the cache
				res, err := l.delegate.RoundTrip(req)
				// remove the Condition for this key so no other requests can subscribe here
				parallelRequestsLock.Lock()
				delete(parallelRequests, key)
				parallelRequestsLock.Unlock()

				// The cache stores the response only after req.Body has been closed.
				// This happens way up the call stack, so we cannot wake up routines waiting yet.
				// They would find the cache still being empty. Hence, the wake-up call has to
				// run separately with a timeout
				go l.wakeUpCallFromCache(req, activeRequest)

				l.logRateLimitInfo(req, res)
				return res, err
			} else {
				// first, lock the Condition and then unlock the map access
				// this ensures Broadcast() will be sent to all subscribers
				l.log.V(5).Info("Similar request already in-flight, waiting for it to complete to get cached response", "key", key)
				activeRequest.L.Lock()
				parallelRequestsLock.Unlock()

				// being awakened from Wait() includes acquiring the lock, hence unlock is called
				activeRequest.Wait()
				activeRequest.L.Unlock()
			}
		}
	}

	l.log.V(5).Info("Roundtrip will check for cached response", "key", key)
	resp, err := l.delegate.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	l.logRateLimitInfo(req, resp)
	return resp, nil
}

func (l *rateLimitControl) logRateLimitInfo(req *http.Request, resp *http.Response) {
	if req == nil {
		l.log.V(2).Info("Rate limit logger: Request is nil")
		return
	}
	if resp == nil {
		l.log.V(2).Info("Rate limit logger: Response is nil")
		return
	}

	total := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	hit := resp.Header.Get(httpcache.XFromCache)
	l.log.V(5).Info("GitHub rate limit", "hit", hit, "total", total, "remaining", remaining, "url", req.URL.String())
	if remaining == "0" {
		l.log.Error(nil, "GitHub request limit exceeded", "total", total, "remaining", remaining, "url", req.URL.String())
	}
}

func (l *rateLimitControl) wakeUpCallFromCache(req *http.Request, reqCondition *sync.Cond) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	for {
		isCached, err := httpcache.CachedResponse(l.cache, req)
		if err != nil || isCached != nil {
			reqCondition.L.Lock()
			reqCondition.Broadcast()
			reqCondition.L.Unlock()
			return
		}
		select {
		case <-ctx.Done():
			reqCondition.L.Lock()
			reqCondition.Broadcast()
			reqCondition.L.Unlock()
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
