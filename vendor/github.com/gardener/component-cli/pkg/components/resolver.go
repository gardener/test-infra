// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/pkg/commands/constants"
)

// ComponentResolver is a subset of the ctf Component Resolver that does not support a blob Resolver.
type ComponentResolver interface {
	Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, error)
}

// Resolver is component resolver that wraps the component descriptor ComponentResolver and adds support
// for a local cache.
type Resolver struct {
	log           logr.Logger
	fs            vfs.FileSystem
	ociResolver   *cdoci.Resolver
	decodeOptions []codec.DecodeOption
}

// New creates a new component Resolver.
func New(log logr.Logger, fs vfs.FileSystem, ociClient ociclient.Client, decodeOpts ...codec.DecodeOption) ComponentResolver {
	return &Resolver{
		log:           log.WithName("componentResolver"),
		fs:            fs,
		ociResolver:   cdoci.NewResolver(decodeOpts...).WithOCIClient(ociClient),
		decodeOptions: decodeOpts,
	}
}

// Resolve resolves a component descriptor from a local cache and falls back to the oci registry.
func (r Resolver) Resolve(ctx context.Context, repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, error) {
	cd, err := ResolveInLocalCache(r.fs, repoCtx, name, version, r.decodeOptions...)
	if err != nil {
		if !errors.Is(err, cdv2.NotFound) {
			return nil, err
		}
		// fallback to oci
		cd, _, err = r.ociResolver.WithRepositoryContext(repoCtx).Resolve(ctx, name, version)
		if err != nil {
			return nil, err
		}
		// try to write back to the local cache
		if err := AddToLocalCache(r.fs, cd); err != nil {
			r.log.Info("unable to store component descriptor in cache", "error", err.Error())
		}
		return cd, nil
	}
	return cd, err
}

// ResolveInLocalCache resolves a component descriptor from a local cache.
// The local cache is expected to have its root at $COMPONENT_REPOSITORY_CACHE_DIR.
// In the root directory each repository context has its own directory, whereas the directory name is $baseurl.replace("/", "-").
// Inside the repository context a component descriptor is cached under $component-name + "-" + $component-version
//
// E.g.
// Given COMPONENT_REPOSITORY_CACHE_DIR="/component-cache";baseUrl="eu.gcr.io/my-context/dev"; component-name="github.com/gardener/component-cli"; component-version="v0.1.0"
// results in the path "/component-cache/eu.gcr.io-my-context-dev/github.com/gardener/component-cli-v0.1.0"
func ResolveInLocalCache(fs vfs.FileSystem, repoCtx cdv2.RepositoryContext, name, version string, decodeOpts ...codec.DecodeOption) (*cdv2.ComponentDescriptor, error) {
	componentPath := LocalCachePath(repoCtx, name, version)

	data, err := vfs.ReadFile(fs, componentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cdv2.NotFound
		}
		return nil, fmt.Errorf("unable to read file from %q: %w", componentPath, err)
	}
	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(data, cd, decodeOpts...); err != nil {
		return nil, fmt.Errorf("unable to decode component descriptor from %q: %w", componentPath, err)
	}
	return cd, nil
}

// AddToLocalCache stores the given filesystem in the local cache.
// The local cache is expected to have its root at $COMPONENT_REPOSITORY_CACHE_DIR.
// In the root directory each repository context has its own directory, whereas the directory name is $baseurl.replace("/", "-").
// Inside the repository context a component descriptor is cached under $component-name + "-" + $component-version
//
// E.g.
// Given COMPONENT_REPOSITORY_CACHE_DIR="/component-cache";baseUrl="eu.gcr.io/my-context/dev"; component-name="github.com/gardener/component-cli"; component-version="v0.1.0"
// results in the path "/component-cache/eu.gcr.io-my-context-dev/github.com/gardener/component-cli-v0.1.0"
func AddToLocalCache(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) error {
	componentPath := LocalCachePath(cd.GetEffectiveRepositoryContext(), cd.GetName(), cd.GetVersion())

	data, err := codec.Encode(cd)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor")
	}
	if err := fs.MkdirAll(filepath.Dir(componentPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create components path %q: %w", filepath.Dir(componentPath), err)
	}
	if err := vfs.WriteFile(fs, componentPath, data, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write component to cache at %q: %w", componentPath, err)
	}
	return nil
}

// LocalCachePath returns the filepath for a component defined by its repository context, name and version.
func LocalCachePath(repoCtx cdv2.RepositoryContext, name, version string) string {
	cacheRoot := os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)
	repositoryDir := strings.ReplaceAll(repoCtx.BaseURL, "/", "-")
	return filepath.Join(cacheRoot, repositoryDir, name+"-"+version)
}
