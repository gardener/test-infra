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

package gardenerscheduler

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util/cmdutil"
	"github.com/gardener/test-infra/pkg/util/cmdvalues"
	"os"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/pkg/errors"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name             hostscheduler.Provider = "gardener"
	CloudProviderAll common.CloudProvider   = "all"
)

var Register hostscheduler.Register = func(m *hostscheduler.Registrations) {
	m.Add(&registration{
		scheduler: &gardenerscheduler{},
	})
}

func (r *registration) Name() hostscheduler.Provider {
	return Name
}
func (r *registration) Description() string {
	return ""
}
func (r *registration) Interface() hostscheduler.Interface {
	return r.scheduler
}
func (r *registration) RegisterFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&r.kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to the gardener cluster kubeconfigPath")
	flagset.StringVar(&r.scheduler.shootName, "name", "", "Name of the shoot")

	cpVal := cmdvalues.NewCloudProviderValue(&r.cloudprovider, CloudProviderAll, CloudProviderAll, common.CloudProviderGCP, common.CloudProviderAWS, common.CloudProviderAzure)
	flagset.Var(cpVal, "cloudprovider", "Specify the cloudprovider of the shoot that should be taken from the pool")

	cmdutil.ViperHelper.BindPFlag("gardener.kubeconfig", flagset.Lookup("kubeconfig"))
}

func (r *registration) PreRun(cmd *cobra.Command, args []string) error {
	r.scheduler.log = logger.Log.WithName(string(r.Name()))

	if r.kubeconfigPath == "" {
		return errors.New("No kubeconfig defined")
	}
	if _, err := os.Stat(r.kubeconfigPath); err != nil {
		return fmt.Errorf("kubeconfig at %s cannot be found", r.kubeconfigPath)
	}
	k8sClient, err := kubernetes.NewClientFromFile("", r.kubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: kubernetes.GardenScheme,
	}))
	if err != nil {
		return err
	}

	namespace, err := getNamespaceOfKubeconfig(r.kubeconfigPath)
	if err != nil {
		return err
	}

	if gardenv1beta1.CloudProvider(r.cloudprovider) == "" {
		return fmt.Errorf("%s is not a supported cloudprovider. Use one of %s, %s, %s, %s, %s, %s, %s", r.cloudprovider, CloudProviderAll,
			gardenv1beta1.CloudProviderAWS, gardenv1beta1.CloudProviderGCP, gardenv1beta1.CloudProviderAzure, gardenv1beta1.CloudProviderAlicloud, gardenv1beta1.CloudProviderOpenStack, gardenv1beta1.CloudProviderPacket)
	}

	r.scheduler.cloudprovider = r.cloudprovider
	r.scheduler.client = k8sClient
	r.scheduler.namespace = namespace
	return nil
}
