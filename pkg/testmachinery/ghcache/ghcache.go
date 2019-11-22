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
	"github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"k8s.io/test-infra/ghproxy/ghcache"
)

// Cache adds github caching to a http client
// Returns a mem cache by default and a disk cache if a directory is defined
func Cache(log logr.Logger, delegate http.RoundTripper) (http.RoundTripper, error) {
	if config == nil {
		return nil, errors.New("no configuration is provided for the github cache")
	}

	//ghcache logging to only log errors as unknown authorization like github app authorizations are log with a warning
	logrus.SetLevel(logrus.ErrorLevel)

	githubCache, err := getCache(delegate)
	if err != nil {
		return nil, err
	}

	return &rateLimitLogger{
		log:      log,
		delegate: githubCache,
	}, nil

}

func getCache(delegate http.RoundTripper) (http.RoundTripper, error) {
	if config.CacheDir == "" {
		return ghcache.NewMemCache(delegate, config.MaxConcurrency), nil
	}

	if err := os.MkdirAll(config.CacheDir, os.ModePerm); err != nil {
		return nil, err
	}
	if config.CacheDiskSizeGB == 0 {
		return nil, errors.New("disk cache size ha to be grater than 0")
	}

	return ghcache.NewDiskCache(delegate, config.CacheDir, config.CacheDiskSizeGB, config.MaxConcurrency), nil
}

// Config is the github cache configuration
type Config struct {
	CacheDir        string
	CacheDiskSizeGB int
	MaxConcurrency  int
}

var config *Config

// DeepCopy copies the configuration object
func (c *Config) DeepCopy() *Config {
	if c == nil {
		return &Config{}
	}
	cfg := *c
	return &cfg
}

func InitFlags(flagset *flag.FlagSet) *Config {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	config = &Config{}
	flagset.StringVar(&config.CacheDir, "github-cache-dir", "",
		"Path directory that should be used to cache github requests")
	flagset.IntVar(&config.CacheDiskSizeGB, "github-cache-size", 1,
		"Size of the github cache in GB")
	flagset.IntVar(&config.MaxConcurrency, "github-max-concurrency", 25,
		"Maximum concurrent requests to the Github api")
	return config
}
