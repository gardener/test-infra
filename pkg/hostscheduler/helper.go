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

package hostscheduler

import (
	"fmt"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/util/secrets"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func IsFlagsetError(err error) error {
	if err != nil {
		if !strings.HasPrefix(err.Error(), "flag provided but not defined") {
			return err
		}
	}
	return nil
}

// WriteHostKubeconfig writes a kubeconfig from a restclient to the kubeconfig host path
func WriteHostKubeconfig(k8sClient kubernetes.Interface) error {
	// Write kubeconfigPath to kubeconfigPath folder: $TM_KUBECONFIG_PATH/host.config
	log.Infof("Writing host kubeconfig to %s", HostKubeconfigPath())

	// Generate kubeconfig from restclient
	kubeconfig, err := secrets.GenerateKubeconfigFromRestConfig(k8sClient.RESTConfig(), "gke-host")
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(HostKubeconfigPath()), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for kubeconfig: %s", filepath.Dir(HostKubeconfigPath()), err.Error())
	}
	err = ioutil.WriteFile(HostKubeconfigPath(), kubeconfig, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write kubeconfig to %s: %s", HostKubeconfigPath(), err.Error())
	}

	return nil
}
