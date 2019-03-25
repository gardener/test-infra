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
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	defaultActiveDeadlineSeconds int64 = 600
	archiveLogs                        = true
)

// Annotation keys defined on the testdefinition template
const (
	AnnotationTestDefName = "testmachinery.sapcloud.io/TestDefinition"
	AnnotationFlow        = "testmachinery.sapcloud.io/Flow"
	AnnotationPosition    = "testmachinery.sapcloud.io/Position" // position of the source step in the format row/colum
)

// New takes a CRD TestDefinition and its locations, and creates a TestDefinition object.
func New(def *tmv1beta1.TestDefinition, loc Location, fileName string) *TestDefinition {

	if err := Validate(fmt.Sprintf("Location: \"%s\"; File: \"%s\"", loc.Name(), fileName), def); err != nil {
		log.Warn(err)
	}

	if def.Spec.Image == "" {
		def.Spec.Image = testmachinery.BASE_IMAGE
	}
	if def.Spec.ActiveDeadlineSeconds == nil {
		def.Spec.ActiveDeadlineSeconds = &defaultActiveDeadlineSeconds
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
					Name: "kubeconfigs",
					Path: testmachinery.TM_KUBECONFIG_PATH,
				},
				{
					Name: "sharedFolder",
					Path: testmachinery.TM_SHARED_PATH,
				},
			},
		},
		Outputs: argov1.Outputs{
			Artifacts: []argov1.Artifact{
				{
					Name: testmachinery.ExportArtifact,
					Path: testmachinery.TM_EXPORT_PATH,
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
	td.AddConfig(td.Config)

	return td
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
	}
}

// SetPosition sets the unique name of the testdefinition and its execution position.
func (td *TestDefinition) SetPosition(flow string, row, column int) {
	td.Template.Metadata.Annotations = GetAnnotations(td.Info.Metadata.Name, flow, fmt.Sprintf("%d/%d", row, column))
}

// GetPosition returns annotations to identify the source step.
func (td *TestDefinition) GetPosition() map[string]string {
	return map[string]string{
		AnnotationFlow:     td.Template.Metadata.Annotations[AnnotationFlow],
		AnnotationPosition: td.Template.Metadata.Annotations[AnnotationPosition],
	}
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
func (td *TestDefinition) AddSerialStdOutput() {
	kubeconfigArtifact := argov1.Artifact{
		Name: "kubeconfigs",
		Path: testmachinery.TM_KUBECONFIG_PATH,
	}
	td.AddOutputArtifacts(kubeconfigArtifact)
	sharedFolderArtifact := argov1.Artifact{
		Name: "sharedFolder",
		Path: testmachinery.TM_SHARED_PATH,
	}
	td.AddOutputArtifacts(sharedFolderArtifact)
}

// AddConfig adds the config elements of different types (environment variable) to the TestDefinitions's template.
func (td *TestDefinition) AddConfig(configs []*config.Element) {
	for _, config := range configs {
		switch config.Info.Type {
		case tmv1beta1.ConfigTypeEnv:
			if config.Info.Value != "" {
				// add as input parameter to see parameters in argo ui
				td.AddInputParameter(config.Name(), fmt.Sprintf("%s: %s", config.Info.Name, config.Info.Value))
				td.AddEnvVars(apiv1.EnvVar{Name: config.Info.Name, Value: config.Info.Value})
			} else {
				// add as input parameter to see parameters in argo ui
				td.AddInputParameter(config.Name(), fmt.Sprintf("%s: %s", config.Info.Name, "from secret or configmap"))
				td.AddEnvVars(apiv1.EnvVar{
					Name: config.Info.Name,
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: config.Info.ValueFrom.ConfigMapKeyRef,
						SecretKeyRef:    config.Info.ValueFrom.SecretKeyRef,
					},
				})
			}
		case tmv1beta1.ConfigTypeFile:
			// https://github.com/argoproj/argo/blob/master/examples/secrets.yaml
			if config.Info.Value != "" {
				data, err := base64.StdEncoding.DecodeString(config.Info.Value)
				if err != nil {
					log.Warnf("Cannot decode value of %s: %s", config.Info.Name, err.Error())
					continue
				}

				// add as input parameter to see parameters in argo ui
				td.AddInputParameter(config.Name(), fmt.Sprintf("%s: %s", config.Info.Name, config.Info.Value))
				td.AddInputArtifacts(argov1.Artifact{
					Name: config.Name(),
					Path: config.Info.Path,
					ArtifactLocation: argov1.ArtifactLocation{
						Raw: &argov1.RawArtifact{
							Data: string(data),
						},
					},
				})
			} else if config.Info.ValueFrom != nil {
				td.AddInputParameter(config.Name(), fmt.Sprintf("%s: %s", config.Info.Name, "from secret or configmap"))
				td.AddVolumeMount(config.Name(), config.Info.Path, path.Base(config.Info.Path), true)
			}
		}

	}
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
