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
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/cmdvalues"
	"github.com/gardener/test-infra/pkg/util/gardensetup"
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
)

const (
	hostprovider        = "hostprovider"
	hostCloudprovider   = "host-cloudprovider"
	gardensetupRevision = "garden-setup-version"
	gardenerVersion     = "gardener-version"
	gardenerCommit      = "gardener-commit"
	gardenerExtensions  = "gardener-extensions"
	kubernetesVersion   = "kubernetes-version"
	cloudprovider       = "cloudprovider"
)

func (t *test) ValidateConfig() error {
	if t.config.Gardener.Version != "" {
		if err := util.CheckDockerImageExists(common.DockerImageGardenerApiServer, t.config.Gardener.Version); err != nil {
			return errors.New(fmt.Sprintf("I am unable to find gardener images of version %s.\n Have you specified the right version?", t.config.Gardener.Version),
				"Maybe you should run the default gardener pipeline before trying to run the integration tests.")
		}
	}
	return nil
}

func (t *test) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(t.Command(), pflag.ContinueOnError)

	flagset.StringVar(&t.config.Namespace, "namespace", "default", "Testrun namespace")
	flagset.Var(cmdvalues.NewHostProviderValue(&t.config.HostProvider, gardenerscheduler.Name), hostprovider, "Specify the provider for selecting the base cluster")
	flagset.StringVar(&t.config.GardenSetupRevision, gardensetupRevision, "master", "Specify the garden setup version to setup gardener")
	flagset.Var(cmdvalues.NewCloudProviderValue(&t.config.BaseClusterCloudprovider, common.CloudProviderGCP, common.CloudProviderGCP, common.CloudProviderAWS, common.CloudProviderAzure),
		hostCloudprovider, "Specify the cloudprovider of the host cluster. Optional and only affect gardener base cluster")
	flagset.StringVar(&t.config.Gardener.Version, gardenerVersion, "", "Specify the gardener to be deployed by garden setup")
	flagset.StringVar(&t.config.Gardener.ImageTag, "gardener-image", "", "Specify the gardener image tag to be deployed by garden setup")
	flagset.StringVar(&t.config.Gardener.Commit, gardenerCommit, "", "Specify the gardener commit that is deployed by garden setup")
	flagset.StringVar(&t.gardenerExtensions, gardenerExtensions, "", "Specify the gardener extensions in the format <extension-name>=<repo>::<revision>")

	flagset.StringVar(&t.config.Shoots.Namespace, "project-namespace", "garden-core", "Specify the shoot namespace where the shoots should be created")
	flagset.StringArrayVar(&t.kubernetesVersions, kubernetesVersion, []string{}, "Specify the kubernetes version to test")
	flagset.VarP(cmdvalues.NewCloudProviderArrayValue(&t.cloudproviders, common.CloudProviderGCP, common.CloudProviderGCP, common.CloudProviderAWS, common.CloudProviderAzure), cloudprovider, "p", "Specify the cloudproviders to test.")

	flagset.StringVarP(&t.testLabel, "label", "l", string(testmachinery.TestLabelDefault), "Specify test label that should be fetched by the testmachinery")
	flagset.BoolVar(&t.hibernation, "hibernation", false, "test hibernation")
	flagset.BoolVar(&t.config.Pause, "pause", false, "Pauses the test before gardener is deleted and cleaned up. Resume with /resume")
	flagset.BoolVar(&t.dryRun, "dry-run", false, "Print the rendered testrun")

	return flagset
}

func (t *test) ApplyDefaultConfig(client github.Client, event *github.GenericRequestEvent, flagset *pflag.FlagSet) error {
	raw, err := client.GetConfig(t.Command())
	if err != nil {
		t.log.Error(err, "cannot get default config")
		return nil
	}
	var defaultConfig DefaultsConfig
	if err := yaml.Unmarshal(raw, &defaultConfig); err != nil {
		t.log.Error(err, "unable to parse default config")
		return errors.New("unable to parse default config", err.Error())
	}

	if !flagset.Changed(hostprovider) && defaultConfig.HostProvider != nil {
		t.config.HostProvider = *defaultConfig.HostProvider
	}
	if !flagset.Changed(hostCloudprovider) && defaultConfig.BaseClusterCloudProvider != nil {
		t.config.BaseClusterCloudprovider = *defaultConfig.BaseClusterCloudProvider
	}
	if !flagset.Changed(gardensetupRevision) && defaultConfig.GardenSetup != nil && defaultConfig.GardenSetup.Revision != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.GardenSetup.Revision.Value())
		if err != nil {
			return errors.New("unable to resolve default config value for garden setup revision", err.Error())
		}
		t.config.GardenSetupRevision = val
	}
	if !flagset.Changed(gardenerVersion) && defaultConfig.Gardener != nil && defaultConfig.Gardener.Version != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.Gardener.Version.Value())
		if err != nil {
			return errors.New("unable to resolve default config value for gardener version", err.Error())
		}
		t.config.Gardener.Version = val
	}
	if !flagset.Changed(gardenerCommit) && defaultConfig.Gardener != nil && defaultConfig.Gardener.Commit != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.Gardener.Commit.Value())
		if err != nil {
			return errors.New("unable to resolve default config value for gardener commit", err.Error())
		}
		t.config.Gardener.Commit = val
	}

	if err := t.applyDefaultExtensions(client, event, defaultConfig, flagset); err != nil {
		return err
	}

	if err := t.applyDefaultShootFlavors(defaultConfig, flagset); err != nil {
		return err
	}
	t.config.Shoots.Flavors, err = shootflavors.New(*defaultConfig.ShootFlavors)
	return err
}

func (t *test) applyDefaultShootFlavors(defaultConfig DefaultsConfig, flagset *pflag.FlagSet) error {

	if flagset.Changed(cloudprovider) && flagset.Changed(kubernetesVersion) {
		flavors := make([]*common.ShootFlavor, len(t.cloudproviders))
		for i, cp := range t.cloudproviders {
			versions := util.ConvertStringArrayToVersions(t.kubernetesVersions)
			flavors[i] = &common.ShootFlavor{
				Provider: cp,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &versions,
				},
			}
		}
		defaultConfig.ShootFlavors = &flavors
		return nil
	}
	if flagset.Changed(cloudprovider) && defaultConfig.ShootFlavors != nil {
		flavors := make([]*common.ShootFlavor, 0)
		for _, flavor := range *defaultConfig.ShootFlavors {
			if util.ContainsCloudprovider(t.cloudproviders, flavor.Provider) {
				flavors = append(flavors, flavor)
			}
		}
		if len(flavors) != len(t.cloudproviders) {
			return errors.New("a kubernetes version for cloudproviders is missing", "")
		}
		defaultConfig.ShootFlavors = &flavors
		return nil
	}

	if flagset.Changed(kubernetesVersion) && defaultConfig.ShootFlavors != nil {
		flavors := *defaultConfig.ShootFlavors
		for i := range flavors {
			versions := util.ConvertStringArrayToVersions(t.kubernetesVersions)
			flavors[i].KubernetesVersions = common.ShootKubernetesVersionFlavor{
				Versions: &versions,
			}
		}
		defaultConfig.ShootFlavors = &flavors
		return nil
	}
	if defaultConfig.ShootFlavors != nil {
		return nil
	}
	return errors.New("At least one cloudprovider and one kubernetes version has to be defined", "neither a default configuration nor cloudprovider and kubernetes flags are defined.")
}

func (t *test) applyDefaultExtensions(client github.Client, event *github.GenericRequestEvent, defaultConfig DefaultsConfig, flagset *pflag.FlagSet) error {
	var (
		err        error
		defaultExt common.GSExtensions
		flagExt    common.GSExtensions
	)
	if flagset.Changed(gardenerExtensions) {
		flagExt, err = gardensetup.ParseFlag(t.gardenerExtensions)
		if err != nil {
			return err
		}
	}
	if defaultConfig.GardenerExtensions != nil {
		val, err := client.ResolveConfigValue(event, defaultConfig.GardenerExtensions)
		if err != nil {
			return errors.New("unable to resolve default config value for gardener extensions", err.Error())
		}

		var rawDep map[string]common.GSVersion
		if err := yaml.Unmarshal([]byte(val), &rawDep); err != nil {
			return errors.Wrap(err, "unable to parse default config value for gardener extensions")
		}
		defaultExt = gardensetup.ConvertRawDependenciesToInternalExtensionConfig(rawDep)
	}

	if flagset.Changed(gardenerExtensions) && defaultConfig.GardenerExtensions == nil {
		t.config.GardenerExtensions = flagExt
		return nil
	}
	if !flagset.Changed(gardenerExtensions) && defaultConfig.GardenerExtensions != nil {
		t.config.GardenerExtensions = defaultExt
		return nil
	}
	if flagset.Changed(gardenerExtensions) && defaultConfig.GardenerExtensions != nil {
		t.config.GardenerExtensions = gardensetup.MergeExtensions(defaultExt, flagExt)
		return nil
	}

	return errors.New("gardener extensions have to be defined", "no gardener extensions are either defined by flag nor by the default config")
}
