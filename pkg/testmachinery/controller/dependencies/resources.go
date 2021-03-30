// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package dependencies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/gardener/gardener-resource-manager/pkg/manager"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	helmengine "helm.sh/helm/v3/pkg/engine"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/config"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

// checkResourceManager checks if a resource manager ist deployed
func (e *DependencyEnsurer) checkResourceManager(ctx context.Context, namespace string) error {
	deployment := &appsv1.Deployment{}
	if err := e.client.Get(ctx, client.ObjectKey{Name: config.ResourceManagerDeploymentName, Namespace: namespace}, deployment); err != nil {
		return err
	}
	return kutil.CheckDeployment(deployment)
}

// createManagedResource creates or updates a managed resource
func (e *DependencyEnsurer) createManagedResource(
	ctx context.Context,
	namespace,
	name string,
	chart *helmChart,
	imageVector imagevector.ImageVector) error {

	data, err := chart.Render(namespace, imageVector)
	if err != nil {
		return fmt.Errorf("could not render chart: %w", err)
	}

	// Create or update secret containing the rendered rbac manifests
	if err := manager.NewSecret(e.client).
		WithNamespacedName(namespace, name).
		WithKeyValues(map[string][]byte{name: data}).
		Reconcile(ctx); err != nil {
		return fmt.Errorf("could not create or update secret '%s/%s' of managed resources: %w", namespace, name, err)
	}

	if err := manager.NewManagedResource(e.client).
		WithNamespacedName(namespace, name).
		WithSecretRef(name).
		Reconcile(ctx); err != nil {
		return fmt.Errorf("could not create or update managed resource '%s/%s': %w", namespace, name, err)
	}
	return nil
}

// helmChart is a internal helper struct for working with helm charts and managed resources
type helmChart struct {
	Name   string
	Path   string
	Images []string
	Values map[string]interface{}
}

const notesFileSuffix = "NOTES.txt"

func (c *helmChart) Render(namespace string, imageVector imagevector.ImageVector) ([]byte, error) {
	images, err := imagevector.FindImages(imageVector, c.Images)
	if err != nil {
		return nil, err
	}
	c.Values["images"] = imagevector.ImageMapToValues(images)

	chart, err := loader.LoadDir(c.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to load helm chart from %q: %w", c.Path, err)
	}

	options := chartutil.ReleaseOptions{
		Name:      c.Name,
		Namespace: namespace,
		Revision:  0,
	}
	values, err := chartutil.ToRenderValues(chart, c.Values, options, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to build helm values: %w", err)
	}

	files, err := helmengine.Render(chart, values)
	if err != nil {
		return nil, fmt.Errorf("unable to render helm chart: %w", err)
	}
	// Remove NOTES.txt and partials
	for k := range files {
		if strings.HasSuffix(k, notesFileSuffix) || strings.HasPrefix(path.Base(k), "_") {
			delete(files, k)
		}
	}
	return CreateManifest(files), nil
}

// CreateManifest returns the manifests of a rendered chart as as one big manifest.
func CreateManifest(manifests map[string]string) []byte {
	// Aggregate all valid manifests into one big doc.
	b := bytes.NewBuffer(nil)

	for name, mf := range manifests {
		b.WriteString("\n---\n# Source: " + name + "\n")
		b.WriteString(mf)
	}
	return b.Bytes()
}

// EncodeValues marshals a object to json and unmarshals it into a go struct that can be used
// in the helm values.
func EncodeValues(obj interface{}) (interface{}, error) {
	var encodedObj interface{}
	encodedObjBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("unable to encode object: %w", err)
	}
	if err := json.Unmarshal(encodedObjBytes, &encodedObj); err != nil {
		return nil, fmt.Errorf("unable to decode object %w", err)
	}
	return encodedObj, nil
}
