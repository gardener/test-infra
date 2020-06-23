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

package run_template

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	testrunnerTemplate "github.com/gardener/test-infra/pkg/testrunner/template"
)

type options struct {
	testrunnerConfig testrunner.Config
	collectConfig    result.Config
	shootParameters  testrunnerTemplate.Parameters

	shootFlavors []*shootflavors.ExtendedFlavorInstance

	fs                   *pflag.FlagSet
	dryRun               bool
	testrunNamePrefix    string
	shootPrefix          string
	tmKubeconfigPath     string
	filterPatchVersions  bool
	failOnError          bool
	testrunFlakeAttempts int
	timeout              int64
}

// NewOptions creates a new options struct.
func NewOptions() *options {
	return &options{
		testrunnerConfig: testrunner.Config{},
		collectConfig:    result.Config{},
		shootParameters:  testrunnerTemplate.Parameters{},
	}
}

// Complete parses the given parameters into the internal struct.
func (o *options) Complete() error {
	o.dryRun, _ = o.fs.GetBool("dry-run")

	gardenK8sClient, err := kubernetes.NewClientFromFile("", o.shootParameters.GardenKubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: kubernetes.GardenScheme,
	}))
	if err != nil {
		logger.Log.Error(err, "unable to build garden kubernetes client", "file", o.tmKubeconfigPath)
		os.Exit(1)
	}

	o.testrunnerConfig.Timeout = time.Duration(o.timeout) * time.Second
	o.collectConfig.ComponentDescriptorPath = o.shootParameters.ComponentDescriptorPath

	if o.shootParameters.FlavorConfigPath != "" {
		flavors, err := GetShootFlavors(o.shootParameters.FlavorConfigPath, gardenK8sClient, o.shootPrefix, o.filterPatchVersions)
		if err != nil {
			logger.Log.Error(err, "unable to parse shoot flavors from test configuration")
			os.Exit(1)
		}
		o.shootFlavors = flavors.GetShoots()
	}

	return nil
}

func GetShootFlavors(cfgPath string, k8sClient kubernetes.Interface, shootPrefix string, filterPatchVersions bool) (*shootflavors.ExtendedFlavors, error) {
	// read and parse test shoot configuration
	dat, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "unable to read test shoot configuration file from %s", cfgPath)
	}

	flavors := common.ExtendedShootFlavors{}
	if err := yaml.Unmarshal(dat, &flavors); err != nil {
		return nil, err
	}

	return shootflavors.NewExtended(k8sClient.Client(), flavors.Flavors, shootPrefix, filterPatchVersions)
}

// Validate validates the options
func (o *options) Validate() error {
	if len(o.tmKubeconfigPath) == 0 {
		return errors.New("tm-kubeconfig-path is required")
	}
	if len(o.testrunNamePrefix) == 0 {
		return errors.New("testrun-prefix is required")
	}
	if len(o.shootParameters.GardenKubeconfigPath) == 0 {
		return errors.New("gardener-kubeconfig-path is required")
	}
	if len(o.shootPrefix) == 0 {
		return errors.New("shoot-name is required")
	}
	return nil
}

func (o *options) AddFlags(fs *pflag.FlagSet) error {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.StringVar(&o.tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	fs.StringVar(&o.testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	fs.StringVarP(&o.testrunnerConfig.Namespace, "namespace", "n", "default", "Namesapce where the testrun should be deployed.")
	fs.Int64Var(&o.timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	fs.IntVar(&o.testrunnerConfig.FlakeAttempts, "testrun-flake-attempts", 0, "Max number of testruns until testrun is successful")
	fs.BoolVar(&o.failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")
	fs.BoolVar(&o.collectConfig.EnableTelemetry, "enable-telemetry", false, "Enables the measurements of metrics during execution")
	fs.BoolVar(&o.testrunnerConfig.Serial, "serial", false, "executes all testruns of a bucket only after the previous bucket has finished")
	fs.IntVar(&o.testrunnerConfig.BackoffBucket, "backoff-bucket", 0, "Number of parallel created testruns per backoff period")
	fs.DurationVar(&o.testrunnerConfig.BackoffPeriod, "backoff-period", 0, "Time to wait between the creation of testrun buckets")

	fs.StringVar(&o.collectConfig.ConcourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// status asset upload
	fs.BoolVar(&o.collectConfig.UploadStatusAsset, "upload-status-asset", false, "Upload testrun status as a github release asset.")
	fs.StringVar(&o.collectConfig.GithubUser, "github-user", os.Getenv("GITHUB_USER"), "GitHUb username.")
	fs.StringVar(&o.collectConfig.GithubPassword, "github-password", os.Getenv("GITHUB_PASSWORD"), "Github password.")
	fs.StringArrayVar(&o.collectConfig.AssetComponents, "asset-component", []string{}, "The github components to which the testrun status shall be attached as an asset.")
	fs.StringVar(&o.collectConfig.AssetPrefix, "asset-prefix", "", "Prefix of the asset name.")

	// slack notification
	fs.StringVar(&o.collectConfig.SlackToken, "slack-token", "", "Client token to authenticate")
	fs.StringVar(&o.collectConfig.SlackChannel, "slack-channel", "", "Client channel id to send the message to.")
	fs.StringVar(&o.collectConfig.ConcourseURL, "concourse-url", "", "Concourse job URL.")
	fs.BoolVar(&o.collectConfig.PostSummaryInSlack, "post-summary-in-slack", false, "Post testruns summary in slack.")

	// parameter flags
	fs.StringVar(&o.shootParameters.DefaultTestrunChartPath, "testruns-chart-path", "", "Path to the default testruns chart.")
	fs.StringVar(&o.shootParameters.FlavoredTestrunChartPath, "flavored-testruns-chart-path", "", "Path to the testruns chart to test shoots.")
	fs.StringVar(&o.shootParameters.GardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")

	fs.StringVar(&o.shootParameters.FlavorConfigPath, "flavor-config", "", "Path to shoot test configuration.")

	fs.StringVar(&o.shootPrefix, "shoot-name", "", "Shoot name which is used to run tests.")
	fs.BoolVar(&o.filterPatchVersions, "filter-patch-versions", false, "Filters patch versions so that only the latest patch versions per minor versions is used.")

	fs.StringVar(&o.shootParameters.ComponentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	fs.StringVar(&o.shootParameters.Landscape, "landscape", "", "Current gardener landscape.")

	fs.StringArrayVar(&o.shootParameters.SetValues, "set", make([]string, 0), "setValues additional helm values")
	fs.StringArrayVarP(&o.shootParameters.FileValues, "values", "f", make([]string, 0), "yaml value files to override template values")

	// DEPRECATED FLAGS
	// is now handled by the testmachinery
	fs.Int64("interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")
	fs.StringVar(&o.collectConfig.OutputDir, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	fs.String("es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	fs.String("es-endpoint", "", "endpoint of the elasticsearch instance")
	fs.String("es-username", "", "username to authenticate against a elasticsearch instance")
	fs.String("es-password", "", "password to authenticate against a elasticsearch instance")
	fs.String("s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	fs.Bool("s3-ssl", false, "S3 has SSL enabled.")
	if err := fs.MarkDeprecated("interval", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("output-dir-path", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-config-name", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-endpoint", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-username", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("es-password", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("s3-endpoint", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}
	if err := fs.MarkDeprecated("s3-ssl", "DEPRECATED: will not we used anymore"); err != nil {
		return err
	}

	o.fs = fs

	return nil
}
