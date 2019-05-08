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

package testdefinition

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/util"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	DefaultActiveDeadlineSeconds int64 = 600
	archiveLogs                        = true
)

// Annotation keys defined on the testdefinition template
const (
	AnnotationTestDefName = "testmachinery.sapcloud.io/TestDefinition"
	AnnotationFlow        = "testmachinery.sapcloud.io/Flow"
	AnnotationPosition    = "testmachinery.sapcloud.io/Position" // position of the source step in the format row/colum
)

// New takes a CRD TestDefinition and its locations, and creates a TestDefinition object.
func New(def *tmv1beta1.TestDefinition, loc Location, fileName string) (*TestDefinition, error) {

	if err := Validate(fmt.Sprintf("Location: \"%s\"; File: \"%s\"", loc.Name(), fileName), def); err != nil {
		return nil, err
	}

	if def.Spec.Image == "" {
		def.Spec.Image = testmachinery.BASE_IMAGE
	}
	if def.Spec.ActiveDeadlineSeconds == nil {
		def.Spec.ActiveDeadlineSeconds = &DefaultActiveDeadlineSeconds
	}

	template := &argov1.Template{
		Name: "",
		ArchiveLocation: &argov1.ArtifactLocation{
			ArchiveLogs: &archiveLogs,
		},
		ActiveDeadlineSeconds: def.Spec.ActiveDeadlineSeconds,
		Container: &apiv1.Container{
			Image:      def.Spec.Image,
			Command:    def.Spec.Command,
			Args:       def.Spec.Args,
			WorkingDir: testmachinery.TM_REPO_PATH,
			Env: []apiv1.EnvVar{
				{
					Name:  "TM_KUBECONFIG_PATH",
					Value: testmachinery.TM_KUBECONFIG_PATH,
				},
				{
					Name:  "TM_SHARED_PATH",
					Value: testmachinery.TM_SHARED_PATH,
				},
				{
					Name:  "TM_EXPORT_PATH",
					Value: testmachinery.TM_EXPORT_PATH,
				},
				{
					Name:  "TM_PHASE",
					Value: "{{inputs.parameters.phase}}",
				},
			},
		},
		Inputs: argov1.Inputs{
			Parameters: []argov1.Parameter{
				{Name: "phase"},
			},
			Artifacts: []argov1.Artifact{
				{
					Name:     "kubeconfigs",
					Path:     testmachinery.TM_KUBECONFIG_PATH,
					Optional: true,
				},
				{
					Name:     "sharedFolder",
					Path:     testmachinery.TM_SHARED_PATH,
					Optional: true,
				},
			},
		},
		Outputs: argov1.Outputs{
			Artifacts: []argov1.Artifact{
				{
					Name:     testmachinery.ExportArtifact,
					Path:     testmachinery.TM_EXPORT_PATH,
					Optional: true,
				},
			},
		},
	}

	td := &TestDefinition{
		Info:     def,
		Location: loc,
		FileName: fileName,
		Template: template,
		Config:   config.New(def.Spec.Config),
	}
	if err := td.AddConfig(td.Config); err != nil {
		return nil, err
	}

	return td, nil
}

// Copy returns a deep copy of the TestDefinition.
func (td *TestDefinition) Copy() *TestDefinition {
	template := td.Template.DeepCopy()
	template.Name = fmt.Sprintf("%s-%s", td.Info.Metadata.Name, util.RandomString(5))
	return &TestDefinition{
		Info:     td.Info,
		Location: td.Location,
		FileName: td.FileName,
		Template: template,
		Config:   td.Config,
		Volumes:  td.Volumes,
	}
}

func (td *TestDefinition) SetName(name string) {
	td.Template.Name = name
}
func (td *TestDefinition) GetName() string {
	return td.Template.Name
}

// HasBehavior checks if the testrun has defined a specific behavior like serial or disruptiv.
func (td *TestDefinition) HasBehavior(behavior string) bool {
	for _, b := range td.Info.Spec.Behavior {
		if b == behavior {
			return true
		}
	}
	return false
}

// HasLabel checks if the TestDefinition has a specific label. (Group in testdef)
func (td *TestDefinition) HasLabel(label string) bool {
	for _, l := range td.Info.Spec.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// AddEnvVars adds environment varibales to the container of the TestDefinition's template.
func (td *TestDefinition) AddEnvVars(envs ...apiv1.EnvVar) {
	td.Template.Container.Env = append(td.Template.Container.Env, envs...)
}

// AddInputArtifacts adds argo artifacts to the input of the TestDefinitions's template.
func (td *TestDefinition) AddInputArtifacts(artifacts ...argov1.Artifact) {
	td.Template.Inputs.Artifacts = append(td.Template.Inputs.Artifacts, artifacts...)
}

// AddOutputArtifacts adds argo artifacts to the output of the TestDefinitions's template.
func (td *TestDefinition) AddOutputArtifacts(artifacts ...argov1.Artifact) {
	td.Template.Outputs.Artifacts = append(td.Template.Outputs.Artifacts, artifacts...)
}

// AddInputParameter adds a parameter to the input of the TestDefinitions's template.
func (td *TestDefinition) AddInputParameter(name, value string) {
	td.Template.Inputs.Parameters = append(td.Template.Inputs.Parameters, argov1.Parameter{Name: name, Value: &value})
}

// AddVolumeMount adds a mount to the container of the TestDefinitions's template.
func (td *TestDefinition) AddVolumeMount(name, path, subpath string, readOnly bool) {
	td.Template.Container.VolumeMounts = append(td.Template.Container.VolumeMounts, apiv1.VolumeMount{
		Name:      name,
		MountPath: path,
		SubPath:   subpath,
		ReadOnly:  readOnly,
	})
}

// AddSerialStdOutput adds the Kubeconfig output to the TestDefinitions's template.
func (td *TestDefinition) AddSerialStdOutput(global bool) {
	kubeconfigArtifact := argov1.Artifact{
		Name:     "kubeconfigs",
		Path:     testmachinery.TM_KUBECONFIG_PATH,
		Optional: true,
	}
	sharedFolderArtifact := argov1.Artifact{
		Name:     "sharedFolder",
		Path:     testmachinery.TM_SHARED_PATH,
		Optional: true,
	}

	if global {
		kubeconfigArtifact.GlobalName = kubeconfigArtifact.Name
		sharedFolderArtifact.GlobalName = sharedFolderArtifact.Name
	}

	td.AddOutputArtifacts(kubeconfigArtifact)
	td.AddOutputArtifacts(sharedFolderArtifact)
}

// AddConfig adds the config elements of different types (environment variable) to the TestDefinitions's template.
func (td *TestDefinition) AddConfig(configs []*config.Element) error {
	for _, cfg := range configs {
		switch cfg.Info.Type {
		case tmv1beta1.ConfigTypeEnv:
			if err := td.addConfigAsEnv(cfg); err != nil {
				return err
			}
		case tmv1beta1.ConfigTypeFile:
			if err := td.addConfigAsFile(cfg); err != nil {
				return err
			}
		}
	}

	return nil
}

func (td *TestDefinition) addConfigAsEnv(element *config.Element) error {
	if element.Info.Value != "" {
		// add as input parameter to see parameters in argo ui
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, element.Info.Value))
		td.AddEnvVars(apiv1.EnvVar{Name: element.Info.Name, Value: element.Info.Value})
	} else {
		// add as input parameter to see parameters in argo ui
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, "from secret or configmap"))
		td.AddEnvVars(apiv1.EnvVar{
			Name: element.Info.Name,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: element.Info.ValueFrom.ConfigMapKeyRef,
				SecretKeyRef:    element.Info.ValueFrom.SecretKeyRef,
			},
		})
	}
	return nil
}

func (td *TestDefinition) addConfigAsFile(element *config.Element) error {
	if element.Info.Value != "" {
		data, err := base64.StdEncoding.DecodeString(element.Info.Value)
		if err != nil {
			return fmt.Errorf("cannot decode value of %s: %s", element.Info.Name, err.Error())
		}

		// add as input parameter to see parameters in argo ui
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, element.Info.Value))
		td.AddInputArtifacts(argov1.Artifact{
			Name: element.Name(),
			Path: element.Info.Path,
			ArtifactLocation: argov1.ArtifactLocation{
				Raw: &argov1.RawArtifact{
					Data: string(data),
				},
			},
		})
	} else if element.Info.ValueFrom != nil {
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, "from secret or configmap"))
		td.AddVolumeMount(element.Name(), element.Info.Path, path.Base(element.Info.Path), true)
		return td.AddVolumeFromConfig(element)
	}

	return nil
}

func (td *TestDefinition) AddVolumeFromConfig(cfg *config.Element) error {
	vol, err := cfg.Volume()
	if err != nil {
		return err
	}
	td.Volumes = append(td.Volumes, *vol)
	return nil
}

// GetAnnotations returns Template annotations for a testdefinition
func GetAnnotations(name, flow, position string) map[string]string {
	annotations := map[string]string{
		AnnotationTestDefName: name,
		AnnotationFlow:        flow,
		AnnotationPosition:    position,
	}
	return annotations
}
