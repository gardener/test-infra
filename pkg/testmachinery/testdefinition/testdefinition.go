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
	"errors"
	"fmt"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/common"

	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/util"

	argov1alpha1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

var (
	DefaultActiveDeadlineSeconds = intstr.FromInt(600)
	archiveLogs                  = true
)

// New takes a CRD TestDefinition and its locations, and creates a TestDefinition object.
func New(def *tmv1beta1.TestDefinition, loc Location, fileName string) (*TestDefinition, error) {

	if err := validation.ValidateTestDefinition(field.NewPath(fmt.Sprintf("Location: \"%s\"; File: \"%s\"", loc.Name(), fileName)), def); len(err) != 0 {
		return nil, err.ToAggregate()
	}

	if def.Spec.Image == "" {
		def.Spec.Image = testmachinery.BaseImage()
	}
	if def.Spec.ActiveDeadlineSeconds == nil {
		def.Spec.ActiveDeadlineSeconds = &DefaultActiveDeadlineSeconds
	}

	template := &argov1alpha1.Template{
		Name: "",
		Metadata: argov1alpha1.Metadata{
			Annotations: map[string]string{
				common.AnnotationTestDefName: def.Name,
			},
		},
		ArchiveLocation: &argov1alpha1.ArtifactLocation{
			ArchiveLogs: &archiveLogs,
		},
		ActiveDeadlineSeconds: def.Spec.ActiveDeadlineSeconds,
		Container: &apiv1.Container{
			Image:      def.Spec.Image,
			Command:    def.Spec.Command,
			Args:       def.Spec.Args,
			Resources:  def.Spec.Resources,
			WorkingDir: testmachinery.TM_REPO_PATH,
			Env: []apiv1.EnvVar{
				{
					Name:  testmachinery.TM_REPO_PATH_NAME,
					Value: testmachinery.TM_REPO_PATH,
				},
				{
					Name:  testmachinery.TM_KUBECONFIG_PATH_NAME,
					Value: testmachinery.TM_KUBECONFIG_PATH,
				},
				{
					Name:  testmachinery.TM_SHARED_PATH_NAME,
					Value: testmachinery.TM_SHARED_PATH,
				},
				{
					Name:  testmachinery.TM_EXPORT_PATH_NAME,
					Value: testmachinery.TM_EXPORT_PATH,
				},
				{
					Name:  testmachinery.TM_PHASE_NAME,
					Value: "{{inputs.parameters.phase}}",
				},
				{
					Name:  testmachinery.TM_GIT_SHA_NAME,
					Value: loc.GitInfo().SHA,
				},
				{
					Name:  testmachinery.TM_GIT_REF_NAME,
					Value: loc.GitInfo().Ref,
				},
			},
		},
		Inputs: argov1alpha1.Inputs{
			Parameters: []argov1alpha1.Parameter{
				{Name: "phase"},
			},
			Artifacts: make([]argov1alpha1.Artifact, 0),
		},
		Outputs: argov1alpha1.Outputs{
			Artifacts: make([]argov1alpha1.Artifact, 0),
		},
	}

	outputArtifacts := []argov1alpha1.Artifact{
		{
			Name:     testmachinery.ExportArtifact,
			Path:     testmachinery.TM_EXPORT_PATH,
			Optional: true,
		},
	}

	td := &TestDefinition{
		Info:            def,
		Location:        loc,
		FileName:        fileName,
		Template:        template,
		inputArtifacts:  make(ArtifactSet),
		outputArtifacts: make(ArtifactSet),
		config:          config.NewSet(config.New(def.Spec.Config, config.LevelTestDefinition)...),
	}
	td.AddOutputArtifacts(outputArtifacts...)

	return td, nil
}

// New returns a deep copy of the TestDefinition.
func (td *TestDefinition) Copy() *TestDefinition {
	template := td.Template.DeepCopy()
	template.Name = fmt.Sprintf("%s-%s", td.Info.GetName(), util.RandomString(5))
	return &TestDefinition{
		Info:            td.Info,
		Location:        td.Location,
		FileName:        td.FileName,
		Template:        template,
		Volumes:         td.Volumes,
		inputArtifacts:  td.inputArtifacts.Copy(),
		outputArtifacts: td.outputArtifacts.Copy(),
		config:          td.config.Copy(),
	}
}

func (td *TestDefinition) SetName(name string) {
	td.AddAnnotation(common.AnnotationTestDefID, name)
	td.Template.Name = name
}
func (td *TestDefinition) GetName() string {
	return td.Template.Name
}

func (td *TestDefinition) SetSuspend() {
	td.Template.Suspend = &argov1alpha1.SuspendTemplate{}
}

func (td *TestDefinition) GetTemplate() (*argov1alpha1.Template, error) {
	for _, cfg := range td.config {
		switch cfg.Info.Type {
		case tmv1beta1.ConfigTypeEnv:
			td.addConfigAsEnv(cfg)
		case tmv1beta1.ConfigTypeFile:
			if err := td.addConfigAsFile(cfg); err != nil {
				return nil, err
			}
		}
	}
	return td.Template, nil
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
	wantedLabels := strings.Split(label, ",")

	for _, wantedLabel := range wantedLabels {
		hasLabel := false
		for _, haveLabel := range td.Info.Spec.Labels {
			if strings.HasPrefix(wantedLabel, "!") && strings.TrimPrefix(wantedLabel, "!") == haveLabel {
				return false
			}
			if haveLabel == wantedLabel {
				hasLabel = true
				break
			}
		}
		if !hasLabel {
			return false
		}
	}
	return true
}

// AddEnvVars adds environment variables to the container of the TestDefinition's template.
func (td *TestDefinition) AddEnvVars(envs ...apiv1.EnvVar) {
	td.Template.Container.Env = append(td.Template.Container.Env, envs...)
}

// AddInputArtifacts adds argo artifacts to the input of the TestDefinitions's template.
func (td *TestDefinition) AddInputArtifacts(artifacts ...argov1alpha1.Artifact) {
	if td.inputArtifacts == nil {
		td.inputArtifacts = make(ArtifactSet)
	}
	for _, a := range artifacts {
		if !td.inputArtifacts.Has(a.Name) {
			td.Template.Inputs.Artifacts = append(td.Template.Inputs.Artifacts, a)
			td.inputArtifacts.Add(a.Name)
		}
	}
}

// AddOutputArtifacts adds argo artifacts to the output of the TestDefinitions's template.
func (td *TestDefinition) AddOutputArtifacts(artifacts ...argov1alpha1.Artifact) {
	if td.outputArtifacts == nil {
		td.outputArtifacts = make(ArtifactSet)
	}
	for _, a := range artifacts {
		if !td.outputArtifacts.Has(a.Name) {
			td.Template.Outputs.Artifacts = append(td.Template.Outputs.Artifacts, a)
			td.outputArtifacts.Add(a.Name)
		}
	}
}

// AddInputParameter adds a parameter to the input of the TestDefinitions's template.
func (td *TestDefinition) AddInputParameter(name, value string) {
	td.Template.Inputs.Parameters = append(td.Template.Inputs.Parameters, argov1alpha1.Parameter{
		Name:  name,
		Value: argov1alpha1.AnyStringPtr(value)},
	)
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

// AddVolume adds a volume to a TestDefinitions's template.
func (td *TestDefinition) AddVolume(volume apiv1.Volume) {
	td.Template.Volumes = append(td.Template.Volumes, volume)
}

// AddStdOutput adds the Kubeconfig output to the TestDefinitions's template.
func (td *TestDefinition) AddStdOutput(global bool) {
	td.AddOutputArtifacts(GetStdOutputArtifacts(global)...)
}

func (td *TestDefinition) GetConfig() config.Set {
	return td.config
}

// AddConfig adds the config elements of different types (environment variable) to the TestDefinitions's template.
func (td *TestDefinition) AddConfig(configs []*config.Element) {
	for _, e := range configs {
		td.config.Add(e)
	}
}

func (td *TestDefinition) addConfigAsEnv(element *config.Element) {
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
}

func (td *TestDefinition) addConfigAsFile(element *config.Element) error {
	if element.Info.Value != "" {
		data, err := base64.StdEncoding.DecodeString(element.Info.Value)
		if err != nil {
			return fmt.Errorf("cannot decode value of %s: %s", element.Info.Name, err.Error())
		}

		// add as input parameter to see parameters in argo ui
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, element.Info.Path))
		// Add the file path as env var with the config name to the pod
		td.AddEnvVars(apiv1.EnvVar{Name: element.Info.Name, Value: element.Info.Path})
		td.AddInputArtifacts(argov1alpha1.Artifact{
			Name: element.Name(),
			Path: element.Info.Path,
			ArtifactLocation: argov1alpha1.ArtifactLocation{
				Raw: &argov1alpha1.RawArtifact{
					Data: string(data),
				},
			},
		})
		return nil
	}
	if element.Info.ValueFrom != nil {
		// add as input parameter to see parameters in argo ui
		td.AddInputParameter(element.Name(), fmt.Sprintf("%s: %s", element.Info.Name, element.Info.Path))
		// Add the file path as env var with the config name to the pod
		td.AddEnvVars(apiv1.EnvVar{Name: element.Info.Name, Value: element.Info.Path})
		td.AddVolumeMount(element.Name(), element.Info.Path, path.Base(element.Info.Path), true)
		return td.AddVolumeFromConfig(element)
	}

	// this should never happen as it is already validated
	return errors.New("either value or value from has to be defined")
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
func (td *TestDefinition) AddAnnotation(key, value string) {
	if td.Template.Metadata.Annotations == nil {
		td.Template.Metadata.Annotations = make(map[string]string)
	}
	td.Template.Metadata.Annotations[key] = value
}
