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

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"

	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/strconf"

	"k8s.io/client-go/tools/clientcmd"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
)

// Validate validates a testrun.
func Validate(log logr.Logger, tr *tmv1beta1.Testrun) error {
	var result *multierror.Error
	// validate locations
	if err := locations.ValidateLocations("spec", tr.Spec); err != nil {
		return err
	}

	// validate global config
	for i, elem := range tr.Spec.Config {
		if err := config.Validate(fmt.Sprintf("spec.config.[%d]", i), elem); err != nil {
			result = multierror.Append(result, err)
		}
	}

	// validate kubeconfigs
	k := reflect.ValueOf(tr.Spec.Kubeconfigs)
	typeOfK := k.Type()
	for i := 0; i < k.NumField(); i++ {
		if err := validateKubeconfig(fmt.Sprintf("spec.strconf.%s", typeOfK.Field(i).Name), k.Field(i).Interface().(*strconf.StringOrConfig)); err != nil {
			result = multierror.Append(result, err)
		}
	}

	locs, err := locations.NewLocations(log, tr.Spec)
	if err != nil {
		result = multierror.Append(result, err)
		return result
	}

	if err := testflow.Validate("spec.testflow", tr.Spec.TestFlow, locs, false); err != nil {
		result = multierror.Append(result, err)
	}

	if err := testflow.Validate("spec.onExit", tr.Spec.OnExit, locs, true); err != nil {
		result = multierror.Append(result, err)
	}

	return util.ReturnMultiError(result)
}

func validateKubeconfig(identifier string, kubeconfig *strconf.StringOrConfig) error {
	if kubeconfig == nil {
		return nil
	}
	if kubeconfig.Type == strconf.String {
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfig.String())
		if err != nil {
			return fmt.Errorf("%s: Cannot decode: %s", identifier, err.Error())
		}

		_, err = clientcmd.Load(kubeconfig)
		if err != nil {
			return fmt.Errorf("%s: Cannot build config: %s", identifier, err.Error())
		}
		return nil
	}
	if kubeconfig.Type == strconf.Config {
		return strconf.Validate(identifier, kubeconfig.Config())
	}

	return fmt.Errorf("%s: Undefined StringSecType %s", identifier, strconf.TypeToString(kubeconfig.Type))
}
