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

package common

// Annotations
const (
	// AnnotationResumeTestrun is the annotation name to trigger resume on the testrun
	AnnotationResumeTestrun = "testmachinery.sapcloud.io/resume"

	// AnnotationTestrunPurpose is the annotation name to specify a purpose of the testrun
	AnnotationTestrunPurpose = "testmachinery.sapcloud.io/purpose"

	// AnnotationTemplateIDTestrun is the annotation to specify the name of the template the testun is rendered from
	AnnotationTemplateIDTestrun = "testrunner.testmachinery.sapcloud.io/templateID"

	// AnnotationLandscape is the annotation to specify the landscape this testrun is testing
	AnnotationLandscape = "testrunner.testmachinery.sapcloud.io/landscape"

	// AnnotationK8sVersion is the annotation to specify the k8s version the testrun is testing
	AnnotationK8sVersion = "testrunner.testmachinery.sapcloud.io/k8sVersion"

	// AnnotationCloudProvider is the annotation to specify the cloudprovider the testrun is testing
	AnnotationCloudProvider = "testrunner.testmachinery.sapcloud.io/cloudprovider"

	// AnnotationOperatingSystem is the annotation to specify the operating system of the shoot nodes the testrun is testing
	AnnotationOperatingSystem = "testrunner.testmachinery.sapcloud.io/operating-system"

	// AnnotationFlavorDescription is the annotation to describe the test flavor of the current run testrun
	AnnotationFlavorDescription = "testrunner.testmachinery.sapcloud.io/flavor-description"

	// AnnotationDimension is the annotation to specify the dimension the testrun is testing
	AnnotationDimension = "testrunner.testmachinery.sapcloud.io/dimension"

	// AnnotationGroupPurpose is the annotation to describe a run group with an arbitrary string
	AnnotationGroupPurpose = "testrunner.testmachinery.sapcloud.io/group-purpose"

	// AnnotationSystemStep is the testflow step annotation to specify that the step is a testmachinery system step.
	// It indicates that it should not be considered as a test and therefore should not count for a test to be failed.
	AnnotationSystemStep = "testmachinery.sapcloud.io/system-step"

	// LabelTestrunRunID is the label to specify the unique name of the run (multiple testruns) this test belongs to.
	// A run represents all tests that are running from one testrunner.
	LabelTestrunRunID = "testrunner.testmachinery.sapcloud.io/runID"

	// LabelIngested is the label that states whether the result of a testrun is already ingested into a persistent storage (db).
	LabelIngested = "testrunner.testmachinery.sapcloud.io/ingested"

	// LabelUploadedToGithub is the label to specify whether the testrun result was uploaded to github
	LabelUploadedToGithub = "testrunner.testmachinery.sapcloud.io/uploaded-to-github"

	// images
	DockerImageGardenerApiServer = "eu.gcr.io/gardener-project/gardener/apiserver"

	// Repositories
	TestInfraRepo          = "https://github.com/gardener/test-infra.git"
	GardenSetupRepo        = "https://github.com/gardener/garden-setup.git"
	GardenerRepo           = "https://github.com/gardener/gardener.git"
	GardenerExtensionsRepo = "https://github.com/gardener/gardener-extensions.git"

	PatternLatest = "latest"
)

var (
	// Default timeout of 4 hours to wait before resuming the testrun
	DefaultPauseTimeout = 14400
)
