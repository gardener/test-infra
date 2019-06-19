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

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/cmd/hostscheduler/scheduler"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeconfigPath string
	clean          bool
	debug          bool
)

func init() {
	// configuration flags
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the gardener cluster kubeconfigPath")
	flag.BoolVar(&clean, "clean", false, "cleanup a previously used shoot. Which means to hibernate the shoot and cleanup left resources.")
	flag.BoolVar(&debug, "debug", false, "debug output.")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	defer ctx.Done()
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Warn("Set debug log level")
	}

	if kubeconfigPath == "" {
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfigPath = os.Getenv("KUBECONFIG")
		}
	}

	if kubeconfigPath == "" {
		log.Fatal("No gardener kubeconfigPath is specified")
	}

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, client.Options{
		Scheme: kubernetes.GardenScheme,
	})
	if err != nil {
		log.Fatal(err)
	}

	namespace, err := getNamespaceOfKubeconfig(kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	if clean {
		if err := scheduler.HibernateShoot(ctx, k8sClient); err != nil {
			log.Fatal(err.Error())
		}
		log.Infof("Successfully hibernated shoot")
		return
	}

	shoot, err := scheduler.ScheduleNewHostShoot(ctx, k8sClient, namespace)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = scheduler.WaitUntilShootIsReconciled(ctx, k8sClient, shoot)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error()))
	}

	log.Infof("Shoot %s successfully woken up and reconciled", shoot.Name)
}

func getNamespaceOfKubeconfig(kubeconfigPath string) (string, error) {
	data, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return "", errors.Wrapf(err, "cannot read file from %s", kubeconfigPath)
	}
	cfg, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return "", err
	}

	ns, _, err := cfg.Namespace()
	if err != nil {
		return "", err
	}
	return ns, nil
}
