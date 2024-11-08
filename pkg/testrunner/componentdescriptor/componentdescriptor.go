// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package componentdescriptor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/logging"
	"ocm.software/ocm/api/config/configutils"
	"ocm.software/ocm/api/datacontext"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/compdesc"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type Options struct {
	// CfgPath is the path to the .ocmconfig file. Per default, the library checks for the config file at
	// $HOME/.ocmconfig.
	CfgPath string
}

type Option func(options *Options)

// GetComponents returns a list of all components that are direct or transitive dependencies (the transitive
// closure) of the component described by the component descriptor stored in a file at cdPath.
//
// repoRef is expected to be an ocm repository reference ([<repo type>::]<host>[:<port>][/<base path>],
// e.g. ocm::example.com/example). It may be left empty, if one or multiple repositories are specified in the
// ocm config file (in fact, it has to be left empty, if the repositories configured in the ocm config shall be used).
// Credentials to the ocm repository can also be set in the .ocmconfig file.
//
// For detailed information, for detailed information on how to configure the ocm config file, check the ocm command
// line tool help with 'ocm configfile --help'
func GetComponents(ctx context.Context, log logr.Logger, cdPath string, repoRef string, opts ...Option) (ComponentList, error) {
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
