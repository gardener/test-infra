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

package common

import (
	"encoding/base64"
	"errors"
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/client-go/rest"
	"path"
	"time"

	clientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	gardeninformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// GetShootKeyFromShoot return a key for a Shoot in the format <shoot-namespace>/<shoot-name>
func GetShootKeyFromShoot(shoot *gardencorev1beta1.Shoot) string {
	if shoot == nil {
		return ""
	}
	return GetShootKey(shoot.ObjectMeta.Name, shoot.ObjectMeta.Namespace)
}

// GetShootKeyFromShoot return a key for a Shoot name and namespace in the format <shoot-namespace>/<shoot-name>
func GetShootKey(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// GetResultFile returns the name of the file for a key
func GetResultFile(dir, key string) string {
	file := fmt.Sprintf("%s.csv", base64.StdEncoding.EncodeToString([]byte(key)))
	return path.Join(GetResultDir(dir), file)
}

// GetResultDir returns the name of the result directory to store measurement files
func GetResultDir(dir string) string {
	return path.Join(dir, "measurements")
}

// Waiter runs the passed function <f> periodically in the given interval <interval>.
// The waiter can block if <block> is true and wait until <f> has been executed
// or just excute <f> and continue. A signal on <stopCh> will stop the execution.
func Waiter(f func(), interval time.Duration, block bool, stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
			time.Sleep(interval)
			if block {
				f()
				continue
			}
			go f()
		}
	}
}

// SetupInformerFactory takes a <kubeconfig> and setup factories to produce informers for k8s v1 and garden api group.
func SetupInformerFactory(restConfig *rest.Config) (k8sinformers.SharedInformerFactory, gardeninformers.SharedInformerFactory, error) {
	if restConfig == nil {
		return nil, nil, errors.New("no kubernetes config is defined")
	}

	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	} else if k8sClientSet == nil {
		return nil, nil, errors.New("k8sClientSet is nil")
	}

	// Get a client for the garden api group.
	gardenClientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	} else if gardenClientSet == nil {
		return nil, nil, errors.New("gardenClientSet is nil")
	}

	// Return the informer factories.
	return k8sinformers.NewSharedInformerFactory(k8sClientSet, 0), gardeninformers.NewSharedInformerFactory(gardenClientSet, 0), nil
}
