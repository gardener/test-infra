package ghcache

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"k8s.io/test-infra/ghproxy/ghcache"
)

// Cache adds github caching to a http client
// Returns a mem cache by default and a disk cache if a directory is defined
func Cache(delegate http.RoundTripper) (http.RoundTripper, error) {
	if config == nil {
		return nil, errors.New("no configuration is provided for the github cache")
	}

	//ghcache logging to only log errors as unknown authorization like github app authorizations are log with a warning
	logrus.SetLevel(logrus.ErrorLevel)

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
