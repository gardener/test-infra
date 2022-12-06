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

package template

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/gardener/gardener/pkg/utils"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

func addAnnotationsToTestrun(tr *tmv1beta1.Testrun, annotations map[string]string) {
	if tr == nil {
		return
	}
	tr.Annotations = utils.MergeStringMaps(tr.Annotations, annotations)
}

func getGardenerVersionFromComponentDescriptor(componentDescriptor componentdescriptor.ComponentList) string {
	for _, component := range componentDescriptor {
		if component == nil {
			continue
		}
		if component.Name == "github.com/gardener/gardener" {
			return component.Version
		}
	}
	return ""
}

func readFileValues(files []string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for _, file := range files {
		var newValues map[string]interface{}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read file %s", file)
		}
		if err := yaml.Unmarshal(data, &newValues); err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal yaml file %s", file)
		}
		values = utils.MergeMaps(values, newValues)
	}
	return values, nil
}

// determineAbsoluteShootChartPath returns the chart to render for the specific shoot flavor
func determineAbsoluteShootChartPath(parameters *internalParameters, chart *string) (string, error) {
	if chart == nil {
		return parameters.ChartPath, nil
	}
	if filepath.IsAbs(*chart) {
		return *chart, nil
	}

	cDir, err := filepath.Abs(filepath.Dir(parameters.FlavorConfigPath))
	if err != nil {
		return "", err
	}
	return filepath.Join(cDir, *chart), nil
}

// encodeRawObject marshals an object into json and encodes it as base64
func encodeRawObject(obj interface{}) (string, error) {
	if reflect.ValueOf(obj).IsNil() {
		return "", nil
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
