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
	"encoding/base32"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"strings"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"

	"github.com/gardener/test-infra/pkg/util"
)

// LocalLocation represents the testDefLocation of type "local".
type LocalLocation struct {
	info        *tmv1beta1.TestLocation
	testdefPath string
	name        string
}

// NewLocalLocation creates a TestDefLocation of type git.
func NewLocalLocation(testDefLocation *tmv1beta1.TestLocation) Location {

	hash := fnv.New32().Sum([]byte(testDefLocation.HostPath))
	b32 := base32.StdEncoding.EncodeToString(hash)

	return &LocalLocation{testDefLocation, fmt.Sprintf("%s/%s", testDefLocation.HostPath, testmachinery.TESTDEF_PATH), strings.ToLower(b32[0:5])}
}

// SetTestDefs adds its TestDefinitions to the TestDefinition Map.
func (l *LocalLocation) SetTestDefs(testDefMap map[string]*TestDefinition) error {
	testDefs, err := l.readTestDefs()
	if err != nil {
		return err
	}
	for _, def := range testDefs {
		def.AddVolumeMount(l.Name(), testmachinery.TM_REPO_PATH, "", false)
		testDefMap[def.Info.Metadata.Name] = def
	}
	return nil
}

// GetLocation returns the local location object
func (l *LocalLocation) GetLocation() *tmv1beta1.TestLocation {
	return l.info
}

// Name returns the generated name of the local location.
func (l *LocalLocation) Name() string {
	return l.name
}

// Type returns the tmv1beta1.LocationTypeLocal.
func (l *LocalLocation) Type() tmv1beta1.LocationType {
	return tmv1beta1.LocationTypeLocal
}

func (l *LocalLocation) readTestDefs() ([]*TestDefinition, error) {
	definitions := []*TestDefinition{}
	files, err := ioutil.ReadDir(l.testdefPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", l.testdefPath, file.Name()))
			if err != nil {
				log.Debugf("Cannot read file %s from %s", file.Name(), l.testdefPath)
				continue
			}
			def, err := util.ParseTestDef(data)
			if err == nil {
				if def.Kind == tmv1beta1.TestDefinitionName && def.Metadata.Name != "" {
					definition, err := New(&def, l, file.Name())
					if err != nil {
						log.Debugf("cannot build testdefinition %s: %s", file.Name(), err.Error())
						continue
					}
					definitions = append(definitions, definition)
				}
			}
		}
	}

	return definitions, nil
}

// GetVolume returns the k8s volume object to the hostPath.
func (l *LocalLocation) GetVolume() apiv1.Volume {
	dirType := apiv1.HostPathDirectory
	return apiv1.Volume{
		Name: l.Name(),
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: l.info.HostPath,
				Type: &dirType,
			},
		},
	}
}
