// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
)

// Collect collects results of all testruns and writes them to a file.
// It returns a list with the names of failed testruns.
func (c *Collector) Collect(ctx context.Context, log logr.Logger, tmClient client.Client, namespace string, runs testrunner.RunList) ([]string, error) {
	var (
		testrunsFailed []string
		result         *multierror.Error
	)

	for _, run := range runs {
		runLogger := log.WithValues("testrun", run.Testrun.Name, "namespace", run.Testrun.Namespace)
		// Do only try to collect testruns results of testruns that ran into a timeout.
		// Any other error can not be retrieved.
		if run.Error != nil && !trerrors.IsTimeout(run.Error) {
			continue
		}

		if run.Testrun.Status.Phase == tmv1beta1.RunPhaseSuccess {
			runLogger.Info("Testrun finished successfully")
		} else {
			testrunsFailed = append(testrunsFailed, run.Testrun.Name)
			runLogger.Error(fmt.Errorf("testrun failed with phase %s", run.Testrun.Status.Phase), "")
		}
		fmt.Print(util.PrettyPrintStruct(run.Testrun.Status))
		output.RenderStatusTable(os.Stdout, run.Testrun.Status.Steps)
		fmt.Println(":---------------------------------------------------------------------------------------------:")
	}

	c.uploadStatusAssets(ctx, c.config, log, runs, tmClient)
	if err := c.postTestrunsSummaryInSlack(c.config, log, runs); err != nil {
		log.Error(err, "error while posting notification on slack")
	}
	fmt.Println(runs.RenderTable())
	if err := logger.PostToGitHubStepSummary(runs.RenderTableWithSymbols(tw.StyleMarkdown), true); err != nil {
		log.Error(err, "unable to post summary to github step summary")
	}

	return testrunsFailed, util.ReturnMultiError(result)
}

func getComponentsForUpload(
	ctx context.Context,
	runLogger logr.Logger,
	componentdescriptorPath string,
	assetComponents []string,
	repository string,
	ocmConfigPath string) ([]*componentdescriptor.Component, error) {
	components, err := componentdescriptor.GetComponents(ctx, runLogger, componentdescriptorPath, repository, func(opts *componentdescriptor.Options) {
		opts.CfgPath = ocmConfigPath
	})
	if err != nil {
		return nil, err
	}
	var componentsForUpload []*componentdescriptor.Component
	for _, componentName := range assetComponents {
		if component := components.Get(componentName); component == nil {
			runLogger.Error(err, "can't find component", "component", assetComponents)
		} else {
			componentsForUpload = append(componentsForUpload, component)
		}
	}
	return componentsForUpload, nil
}

func (c *Collector) uploadStatusAssets(ctx context.Context, cfg Config, log logr.Logger, runs testrunner.RunList, tmClient client.Client) {
	if !cfg.UploadStatusAsset {
		return
	}

	if len(cfg.AssetComponents) == 0 || cfg.GithubPassword == "" || cfg.GithubUser == "" || cfg.ComponentDescriptorPath == "" {
		err := errors.New("missing github password / github user / component descriptor path argument")
		log.Error(err, fmt.Sprintf("components: %s, ghUser: %s, ghPasswordLength: %d", cfg.AssetComponents, cfg.GithubUser, len(cfg.GithubPassword)))
		return
	}

	componentsForUpload, err := getComponentsForUpload(ctx, log, cfg.ComponentDescriptorPath, cfg.AssetComponents, cfg.Repository, cfg.OCMConfigPath)
	if err != nil {
		log.Error(err, "unable to get component for upload")
		return
	}
	if err := UploadStatusToGithub(log.WithName("github-upload"), runs, componentsForUpload, cfg.GithubUser, cfg.GithubPassword, cfg.AssetPrefix); err != nil {
		log.Error(err, "unable to upload status to github")
		return
	}

	if err := MarkTestrunsAsUploadedToGithub(log, tmClient, runs); err != nil {
		log.Error(err, "unable to mark testrun status as uploaded to github")
	}
}
