// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ghcache

import (
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/go-logr/logr"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/apis/config"
)

var (
	ghCache       httpcache.Cache
	maxAgeSeconds int
	initOnce      sync.Once
)

// WithRateLimitControlCache adds the central GitHub cache to a http client.
// Call InitGitHubCache in advance for bootstrapping the cache
func WithRateLimitControlCache(log logr.Logger, delegate http.RoundTripper) (http.RoundTripper, error) {

	if ghCache == nil {
		return nil, errors.New("cache has not been initialized yet")
	}

	cachedTransport := httpcache.NewTransport(ghCache)
	cachedTransport.Transport = &cache{
		delegate:      delegate,
		maxAgeSeconds: maxAgeSeconds,
	}

	return &rateLimitControl{
		log:      log,
		delegate: cachedTransport,
		cache:    ghCache,
	}, nil

}

// InitGitHubCache initializes a central cache exactly once
// It returns a mem cache by default and a disk cache if a directory is defined
func InitGitHubCache(cfg *config.GitHubCache) {
	initOnce.Do(func() {
		if cfg == nil && internalConfig == nil {
			panic("no configuration is provided for the github cache")
		}
		if cfg == nil {
			cfg = internalConfig
		}

		maxAgeSeconds = cfg.MaxAgeSeconds

		if cfg.CacheDir == "" {
			ghCache = httpcache.NewMemoryCache()
			return
		}

		if err := os.MkdirAll(cfg.CacheDir, 0750); err != nil {
			panic(err)
		}
		if cfg.CacheDiskSizeGB == 0 {
			panic("disk cache size ha to be grater than 0")
		}

		ghCache = diskcache.NewWithDiskv(
			diskv.New(diskv.Options{
				BasePath:     path.Join(cfg.CacheDir, "data"),
				TempDir:      path.Join(cfg.CacheDir, "temp"),
				CacheSizeMax: cfg.CacheDiskSizeGB * uint64(1000000000), // GB to B
			}))
	})
}

var internalConfig *config.GitHubCache

// SetConfig sets the internal github cache configuration
func SetConfig(cfg *config.GitHubCache) {
	internalConfig = cfg
}

// AddFlags adds github cache flags to the given flagset
func AddFlags(flagset *flag.FlagSet) *config.GitHubCache {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	internalConfig = &config.GitHubCache{}
	flagset.StringVar(&internalConfig.CacheDir, "github-cache-dir", "",
		"Path directory that should be used to cache github requests")
	flagset.Uint64Var(&internalConfig.CacheDiskSizeGB, "github-cache-size", 1,
		"Size of the github cache in GB")
	flagset.IntVar(&internalConfig.MaxAgeSeconds, "github-cache-max-age", 3600,
		"Maximum age of a failed github response in seconds")
	return internalConfig
}
