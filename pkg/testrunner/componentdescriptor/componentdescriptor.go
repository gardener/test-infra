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
package componentdescriptor

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gardener/component-cli/ociclient"
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/gardener/component-cli/pkg/commands/constants"
	cdcomponents "github.com/gardener/component-cli/pkg/components"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf/ctfutils"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// GetComponents returns a list of all git/component dependencies.
// todo: re-enable component validation
func GetComponents(ctx context.Context, log logr.Logger, ociClient ociclient.Client, content []byte) (ComponentList, error) {
	components := components{
		components:   make([]*Component, 0),
		componentSet: make(map[Component]bool),
	}

	compDesc := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(content, compDesc, codec.DisableValidation(true)); err != nil {
		return nil, err
	}
	resolver := cdoci.NewResolver(ociClient, codec.DisableValidation(true)).WithLog(log)
	if len(os.Getenv(constants.ComponentRepositoryCacheDirEnvVar)) != 0 {
		resolver.WithCache(cdcomponents.NewLocalComponentCache(osfs.New()))
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

// GetComponentsFromFile read a component descriptor and returns a ComponentList
func GetComponentsFromFile(ctx context.Context, log logr.Logger, ociClient ociclient.Client, file string) (ComponentList, error) {
	if file == "" {
		return make(ComponentList, 0), nil
	}
	data, err := ioutil.ReadFile(file)
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
// {
//	"component_name": {
//	 	"version": "0.0.0"
//	}
// }
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
