// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componentdescriptor

import (
	"context"
	"fmt"
	"github.com/gardener/component-cli/pkg/commands/constants"
	cdcomponents "github.com/gardener/component-cli/pkg/components"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf/ctfutils"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/logging"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"os"
	"strings"

	"github.com/gardener/component-cli/ociclient"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/open-component-model/ocm/pkg/contexts/config/configutils"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

// GetComponents returns a list of all git/component dependencies.
// todo: re-enable component validation
func GetComponents(ctx context.Context, log logr.Logger, ociClient ociclient.Client, content []byte) (ComponentList, error) {
	components := components{
		components:   make([]*Component, 0),
		componentSet: make(map[Component]bool),
	}

	fs := osfs.New()
	if len(os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)) == 0 {
		// always use a local cache
		log.Info(fmt.Sprintf("no cache set (%s) using temporary dir", constants.ComponentRepositoryCacheDirEnvVar))
		tmpCacheDir, err := vfs.TempDir(fs, fs.FSTempDir(), "cache-")
		if err != nil {
			return nil, fmt.Errorf("unable to create temporary cache: %w", err)
		}
		if err := os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, tmpCacheDir); err != nil {
			return nil, fmt.Errorf("unable to set cache dir environment variable: %w", err)
		}
	}

	localCache := cdcomponents.NewLocalComponentCache(osfs.New())
	resolver := cdoci.NewResolver(ociClient, codec.DisableValidation(true)).WithLog(log).WithCache(localCache)

	compDesc := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(content, compDesc, codec.DisableValidation(true)); err != nil {
		return nil, err
	}
	if err := localCache.Store(ctx, compDesc); err != nil {
		return nil, err
	}

	compList, err := ctfutils.ResolveList(ctx, resolver, compDesc.GetEffectiveRepositoryContext(), compDesc.GetName(), compDesc.GetVersion())
	if err != nil {
		return nil, err
	}
	// add current component to list
	components.add(Component{
		Name:    compDesc.GetName(),
		Version: compDesc.GetVersion(),
	})
	for _, comp := range compList.Components {
		// todo: use the source to fetch the git repository and the real version/commit
		components.add(Component{
			Name:    comp.Name,
			Version: comp.Version,
		})
	}

	return components.components, nil
}

type Options struct {
	// CfgPath is the path to the .ocmconfig file. Per default, the library checks for the config file at
	// $HOME/.ocmconfig.
	CfgPath string
}

type Option func(options *Options)

// GetComponentsWithOCM returns a list of all components that are direct or transitive dependencies (the transitive closure) of
// the component described by the component descriptor stored in a file at cdPath.
// repoRef is expected to be an ocm repository reference ([<repo type>::]<host>[:<port>][/<base path>],
// e.g. ocm::example.com/example). It may be left empty, if one or multiple repositories are specified in the
// .ocmconfig file (in fact, it has to be left empty, if the repositories configured in the .ocmconfig shall be used).
// Credentials to the ocm repository can also be set in the .ocmconfig file.
func GetComponentsWithOCM(ctx context.Context, log logr.Logger, cdPath string, repoRef string, opts ...Option) (ComponentList, error) {
	// enables the ocm library to log with the logger configuration of this library
	logging.DefaultContext().SetBaseLogger(log)

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	if cdPath == "" {
		return make([]*Component, 0), nil
	}
	data, err := os.ReadFile(cdPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read component descriptor file %s: %s", cdPath, err.Error())
	}

	cd, err := compdesc.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("cannot decode component descriptor: %w", err)
	}

	if len(cd.References) == 0 {
		return []*Component{NewFromVersionedElement(cd)}, nil
	}

	octx := ocm.New(datacontext.MODE_EXTENDED)
	err = configutils.ConfigureContext(octx, options.CfgPath)
	if err != nil {
		return nil, fmt.Errorf("error configuring ocm context: %w", err)
	}

	var resolver ocm.ComponentVersionResolver
	if repoRef != "" {
		repoSpec, err := ocm.ParseRepoToSpec(octx, repoRef)
		if err != nil {
			return nil, fmt.Errorf("error parsing repository reference: %w", err)
		}
		repo, err := octx.RepositoryForSpec(repoSpec)
		if err != nil {
			return nil, err
		}
		resolver = repo
		log.Info("using repository specified by the repository argument to resolve referenced components, " +
			"resolvers defined in .ocmconfig are ignored!")
	} else {
		// TODO: remove after upgrading to ocm release >0.4.3
		// indirectly call the update method on the context (otherwise resolvers are not actually configured)
		_ = octx.BlobHandlers()
		resolver = octx.GetResolver()
		log.Info("using repositories specified in the .ocmconfig")
	}

	if resolver == nil {
		return nil, fmt.Errorf("no repositories configured, specify either a specific repository in the " +
			"repository argument or one or multiple repositories in the .ocmconfig file")
	}

	return resolveReferences(cd, resolver)
}

// The resolveReferences function is an auxiliary function for GetComponents. It implements a typical Breadth First
// Search (BFS) algorithm to traverse the (directed acyclic) graph of components (or rather component versions) to
// provide a flat list containing (name, version) of all components that are direct or transitive dependencies (the
// transitive closure) of the root component c.
// resolver can either a specific ocm repository (if all components are in the same repository) or a compound resolver
// covering multiple repositories
func resolveReferences(c *compdesc.ComponentDescriptor, resolver ocm.ComponentVersionResolver) ([]*Component, error) {
	components := make([]*Component, 0)
	// Set size to 0, as we cannot know the number of component version beforehand
	visited := make(map[Component]struct{}, 0)
	queue := make([]*compdesc.ComponentDescriptor, 0)

	visited[*NewFromVersionedElement(c)] = struct{}{}
	queue = append(queue, c)

	for len(queue) > 0 {
		c = queue[0]
		queue = queue[1:]

		components = append(components, NewFromVersionedElement(c))

		for _, ref := range c.References {
			ac, err := resolver.LookupComponentVersion(ref.GetComponentName(), ref.GetVersion())
			if err != nil {
				return nil, fmt.Errorf("error resolving component references: %w", err)
			}

			if _, ok := visited[*NewFromVersionedElement(ac)]; !ok {
				visited[*NewFromVersionedElement(ac)] = struct{}{}
				queue = append(queue, ac.GetDescriptor())
			}
		}
	}
	return components, nil
}

// GetComponentsFromFile read a component descriptor and returns a ComponentList
func GetComponentsFromFile(ctx context.Context, log logr.Logger, ociClient ociclient.Client, file string) (ComponentList, error) {
	if file == "" {
		return make(ComponentList, 0), nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read component descriptor file %s: %s", file, err.Error())
	}
	return GetComponents(ctx, log, ociClient, data)
}

// GetComponentsFromFileWithOCIOptions read a component descriptor and returns a ComponentList
func GetComponentsFromFileWithOCIOptions(ctx context.Context, log logr.Logger, ociOpts *ociopts.Options, file string) (ComponentList, error) {
	ociClient, _, err := ociOpts.Build(log, osfs.New())
	if err != nil {
		return nil, err
	}
	return GetComponentsFromFile(ctx, log, ociClient, file)
}

// GetComponentsFromLocations parses a list of components from a testruns's locations
func GetComponentsFromLocations(tr *tmv1beta1.Testrun) (ComponentList, error) {
	components := components{
		components:   make([]*Component, 0),
		componentSet: make(map[Component]bool),
	}
	for _, locSet := range tr.Spec.LocationSets {
		for _, loc := range locSet.Locations {
			components.add(Component{
				Name:    strings.Replace(loc.Repo, "https://", "", 1),
				Version: loc.Revision,
			})
		}
	}
	return components.components, nil
}

// JSON returns the json output for a list of components
// The list is converted into the format:
//
//	{
//		"component_name": {
//		 	"version": "0.0.0"
//		}
//	}
func (c ComponentList) JSON() map[string]ComponentJSON {
	components := make(map[string]ComponentJSON)
	for _, component := range c {
		components[component.Name] = ComponentJSON{
			Version: component.Version,
		}
	}
	return components
}

// Get returns the component by its name
func (c ComponentList) Get(name string) *Component {
	for _, component := range c {
		if component.Name == name {
			return component
		}
	}
	return nil
}

func (c *components) add(newComponents ...Component) {
	for _, item := range newComponents {
		component := item
		if !c.has(component) {
			c.components = append(c.components, &component)
			c.componentSet[component] = true
		}
	}
}

func (c *components) has(newComponent Component) bool {
	_, ok := c.componentSet[newComponent]
	return ok
}
