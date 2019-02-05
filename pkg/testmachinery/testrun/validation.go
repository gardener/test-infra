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

package testrun

import (
	"encoding/base64"
	"fmt"
	"reflect"

	"k8s.io/client-go/tools/clientcmd"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
)

// Validate validates a testrun.
func Validate(tr *tmv1beta1.Testrun) error {

	// validate global config
	for i, elem := range tr.Spec.Config {
		if err := config.Validate(fmt.Sprintf("spec.config.[%d]", i), elem); err != nil {
			return err
		}
	}

	// validate kubeconfigs
	k := reflect.ValueOf(tr.Spec.Kubeconfigs)
	typeOfK := k.Type()
	for i := 0; i < k.NumField(); i++ {
		if err := validateKubeconfig(fmt.Sprintf("spec.kubeconfig.%s", typeOfK.Field(i).Name), k.Field(i).String()); err != nil {
			return err
		}
	}

	testDefinitions, err := testdefinition.NewTestDefinitions(tr.Spec.TestLocations)
	if err != nil {
		return err
	}

	if err := testflow.Validate(fmt.Sprintf("spec.testFlow"), &tr.Spec.TestFlow, testDefinitions, false); err != nil {
		return err
	}

	if err := testflow.Validate(fmt.Sprintf("spec.onExit"), &tr.Spec.OnExit, testDefinitions, true); err != nil {
		return err
	}

	return nil
}

func validateKubeconfig(identifier, data string) error {
	kubeconfig, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("%s: Cannot decode: %s", identifier, err.Error())
	}

	_, err = clientcmd.Load(kubeconfig)
	if err != nil {
		return fmt.Errorf("%s: Cannot build config: %s", identifier, err.Error())
	}
	return nil
}
