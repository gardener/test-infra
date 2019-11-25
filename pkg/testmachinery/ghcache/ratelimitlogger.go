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
	"github.com/go-logr/logr"
	"github.com/gregjones/httpcache"
	"net/http"
)

var _ http.RoundTripper = &rateLimitLogger{}

type rateLimitLogger struct {
	log      logr.Logger
	delegate http.RoundTripper
}

func (l *rateLimitLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := l.delegate.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	total := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	hit := resp.Header.Get(httpcache.XFromCache)
	l.log.V(5).Info("GitHub rate limit", "hit", hit, "total", total, "remaining", remaining, "url", req.URL.String())
	if remaining == "0" {
		l.log.Error(nil, "GitHub request limit exceeded", "total", total, "remaining", remaining, "url", req.URL.String())
	}

	return resp, nil
}
