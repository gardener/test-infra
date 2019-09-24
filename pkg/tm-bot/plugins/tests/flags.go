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

package tests

import (
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/util/cmdvalues"
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
)

const (
	hostprovider        = "hostprovider"
	hostCloudprovider   = "host-cloudprovider"
	gardensetupRevision = "garden-setup-version"
	gardenerVersion     = "gardener-version"
	gardenerCommit      = "gardener-commit"
	kubernetesVersion   = "kubernetes-version"
	cloudprovider       = "cloudprovider"
)

func (t *test) ValidateFlags(flagset *pflag.FlagSet) error {
	return nil
}

func (t *test) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(t.Command(), pflag.ContinueOnError)

	flagset.StringVar(&t.config.Namespace, "namespace", "default", "Testrun namespace")
	flagset.Var(cmdvalues.NewHostProviderValue(&t.config.HostProvider, gardenerscheduler.Name), hostprovider, "Specify the provider for selecting the base cluster")
	flagset.StringVar(&t.config.GardenSetupRevision, gardensetupRevision, "master", "Specify the garden setup version to setup gardener")
	flagset.Var(cmdvalues.NewCloudProviderValue(&t.config.BaseClusterCloudprovider, v1beta1.CloudProviderGCP, v1beta1.CloudProviderGCP, v1beta1.CloudProviderAWS, v1beta1.CloudProviderAzure),
		hostCloudprovider, "Specify the cloudprovider of the host cluster. Optional and only affect gardener base cluster")
	flagset.StringVar(&t.config.Gardener.Version, gardenerVersion, "", "Specify the gardener to be deployed by garden setup")
	flagset.StringVar(&t.config.Gardener.ImageTag, "gardener-image", "", "Specify the gardener image tag to be deployed by garden setup")
	flagset.StringVar(&t.config.Gardener.Commit, gardenerCommit, "", "Specify the gardener commit that is deployed by garden setup")

	flagset.StringVar(&t.config.Shoots.Namespace, "project-namespace", "garden-core", "Specify the shoot namespace where the shoots should be created")
	flagset.StringArrayVar(&t.config.Shoots.KubernetesVersions, kubernetesVersion, []string{}, "Specify the kubernetes version to test")
	flagset.VarP(cmdvalues.NewCloudProviderArrayValue(&t.config.Shoots.CloudProviders, v1beta1.CloudProviderGCP, v1beta1.CloudProviderGCP, v1beta1.CloudProviderAWS, v1beta1.CloudProviderAzure), cloudprovider, "p", "Specify the cloudproviders to test.")

	flagset.StringVarP(&t.testLabel, "label", "l", string(testmachinery.TestLabelDefault), "Specify test label that should be fetched by the testmachinery")
	flagset.BoolVar(&t.hibernation, "hibernation", false, "test hibernation")
	flagset.BoolVar(&t.dryRun, "dry-run", false, "Print the rendered testrun")

	return flagset
}

func (t *test) ApplyDefaultConfig(client github.Client, event *github.GenericRequestEvent, flagset *pflag.FlagSet) {
	raw, err := client.GetConfig(t.Command())
	if err != nil {
		t.log.Error(err, "cannot get default config")
		return
	}
	var defaultConfig DefaultsConfig
	if err := yaml.Unmarshal(raw, &defaultConfig); err != nil {
		t.log.Error(err, "unable to parse default config")
		return
	}

	if !flagset.Changed(hostprovider) && defaultConfig.HostProvider != nil {
		t.config.HostProvider = *defaultConfig.HostProvider
	}
	if !flagset.Changed(hostprovider) && defaultConfig.BaseClusterCloudProvider != nil {
		t.config.BaseClusterCloudprovider = *defaultConfig.BaseClusterCloudProvider
	}
	if !flagset.Changed(gardensetupRevision) && defaultConfig.GardenSetup != nil && defaultConfig.GardenSetup.Revision != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.GardenSetup.Revision.Value())
		if err != nil {
			t.log.Error(err, "unable to resolve config value for garden setup revision")
		} else {
			t.config.GardenSetupRevision = val
		}
	}
	if !flagset.Changed(gardenerVersion) && defaultConfig.Gardener != nil && defaultConfig.Gardener.Version != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.Gardener.Version.Value())
		if err != nil {
			t.log.Error(err, "unable to resolve config value for gardener version")
		} else {
			t.config.Gardener.Version = val
		}
	}
	if !flagset.Changed(gardenerCommit) && defaultConfig.Gardener != nil && defaultConfig.Gardener.Commit != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.Gardener.Commit.Value())
		if err != nil {
			t.log.Error(err, "unable to resolve config value for gardener commit")
		} else {
			t.config.Gardener.Commit = val
		}
	}

	if !flagset.Changed(kubernetesVersion) && defaultConfig.Kubernetes != nil && defaultConfig.Kubernetes.Versions != nil {
		t.config.Shoots.KubernetesVersions = *defaultConfig.Kubernetes.Versions
	}
	if !flagset.Changed(cloudprovider) && defaultConfig.CloudProviders != nil {
		t.config.Shoots.CloudProviders = *defaultConfig.CloudProviders
	}
}
