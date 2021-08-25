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

package location

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v39/github"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/ghcache"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"

	"github.com/gardener/test-infra/pkg/util"
)

var githubContentTypeFile = "file"

// GitLocation represents the testDefLocation of type "git".
type GitLocation struct {
	log  logr.Logger
	Info *tmv1beta1.TestLocation

	config    *testmachinery.GitHubInstanceConfig
	repoOwner string
	repoName  string
	repoURL   *url.URL
	gitInfo   testdefinition.GitInfo
}

// NewGitLocation creates a TestDefLocation of type git.
func NewGitLocation(log logr.Logger, testDefLocation *tmv1beta1.TestLocation) (testdefinition.Location, error) {
	repoURL, err := url.Parse(testDefLocation.Repo)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url %s", testDefLocation.Repo)
	}
	config := getGitConfig(log, repoURL)
	repoOwner, repoName := util.ParseRepoURL(repoURL)

	return &GitLocation{
		Info:      testDefLocation,
		log:       log,
		config:    config,
		repoOwner: repoOwner,
		repoName:  repoName,
		repoURL:   repoURL,
	}, nil
}

// SetTestDefs adds its TestDefinitions to the TestDefinition Map.
func (l *GitLocation) SetTestDefs(testDefMap map[string]*testdefinition.TestDefinition) error {
	// ignore the location if the domain is excluded
	if util.DomainMatches(l.repoURL.Hostname(), testmachinery.Locations().ExcludeDomains...) {
		return nil
	}
	testDefs, err := l.getTestDefs()
	if err != nil {
		return err
	}
	for _, def := range testDefs {
		// Prioritize local testdefinitions over remote
		if testDefMap[def.Info.Name] == nil || testDefMap[def.Info.Name].Location.Type() != tmv1beta1.LocationTypeLocal {
			def.AddInputArtifacts(argov1.Artifact{
				Name: "repo",
				Path: testmachinery.TM_REPO_PATH,
			})
			testDefMap[def.Info.Name] = def
		}
	}
	return nil
}

// GetLocation returns the git location object.
func (l *GitLocation) GetLocation() *tmv1beta1.TestLocation {
	return l.Info
}

// Name returns the unique name of the git location consisting of the repository's owner, name and revision.
func (l *GitLocation) Name() string {
	name := fmt.Sprintf("%s-%s-%s", l.repoOwner, l.repoName, l.Info.Revision)
	return util.FormatArtifactName(name)
}

// Type returns the tmv1beta1.LocationTypeGit.
func (l *GitLocation) Type() tmv1beta1.LocationType {
	return tmv1beta1.LocationTypeGit
}

// GitInfo returns the git info for the current test location.
func (l *GitLocation) GitInfo() testdefinition.GitInfo {
	return l.gitInfo
}

func (l *GitLocation) getTestDefs() ([]*testdefinition.TestDefinition, error) {
	ctx := context.Background()
	defer ctx.Done()
	var definitions []*testdefinition.TestDefinition

	client, httpClient, err := l.getGitHubClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create github client for %s: %s", l.Info.Repo, err.Error())
	}

	tree, _, err := client.Git.GetTree(ctx, l.repoOwner, l.repoName, l.Info.Revision, false)
	if err != nil {
		return nil, fmt.Errorf("unable to get git tree for revision %s: %w", l.Info.Revision, err)
	}
	l.gitInfo.SHA = tree.GetSHA()
	if l.Info.Revision != l.gitInfo.SHA {
		l.gitInfo.Ref = l.Info.Revision
	}

	_, directoryContent, _, err := client.Repositories.GetContents(ctx, l.repoOwner, l.repoName,
		testmachinery.TestDefPath(), &github.RepositoryContentGetOptions{Ref: l.Info.Revision})
	if err != nil {
		return nil, fmt.Errorf("no testdefinitions can be found in %s: %s", l.Info.Repo, err.Error())
	}

	for _, file := range directoryContent {
		if *file.Type == githubContentTypeFile {
			l.log.V(5).Info("found file", "filename", *file.Name, "path", *file.Path)
			data, err := util.DownloadFile(httpClient, file.GetDownloadURL())
			if err != nil {
				return nil, err
			}
			def, err := util.ParseTestDef(data)
			if err != nil {
				l.log.V(5).Info(fmt.Sprintf("ignoring file: %s", err.Error()), "filename", *file.Name)
				continue
			}
			if def.Kind == tmv1beta1.TestDefinitionName && def.Name != "" {
				definition, err := testdefinition.New(&def, l, file.GetName())
				if err != nil {
					l.log.Info(fmt.Sprintf("unable to build testdefinition: %s", err.Error()), "filename", *file.Name)
					continue
				}
				definitions = append(definitions, definition)
				l.log.V(3).Info(fmt.Sprintf("found TestDefinition %s", def.Name))
			}
		}
	}

	return definitions, nil
}

func (l *GitLocation) getGitHubClient() (*github.Client, *http.Client, error) {
	httpClient, err := l.getHTTPClient()
	if err != nil {
		return nil, nil, err
	}
	client, err := github.NewEnterpriseClient(l.getGitHubAPI(), "", httpClient)
	if err != nil {
		return nil, nil, err
	}
	return client, httpClient, nil
}

func (l *GitLocation) getHTTPClient() (*http.Client, error) {
	if l.config != nil {
		trp, err := ghcache.Cache(l.log.WithName("ghCache"), testmachinery.GetConfig().GitHub.Cache, &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: l.config.SkipTls},
		})
		if err != nil {
			return nil, err
		}

		basicAuth := github.BasicAuthTransport{
			Username:  l.config.TechnicalUser.Username,
			Password:  l.config.TechnicalUser.AuthToken,
			Transport: trp,
		}
		l.log.V(3).Info(fmt.Sprintf("used gitconfig for %s to authenticate", l.config.HttpUrl))
		return basicAuth.Client(), nil
	}

	l.log.V(3).Info("insecure and unauthenticated git connection is used")
	trp, err := ghcache.Cache(l.log.WithName("ghCache"), testmachinery.GetConfig().GitHub.Cache, &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: trp,
	}, nil
}

// Legacy function. Maybe can be removed in the future when git config is necessary.
func (l *GitLocation) getGitHubAPI() string {
	if l.config != nil {
		return l.config.ApiUrl
	}
	var apiURL string
	if l.repoURL.Hostname() == "github.com" {
		apiURL = "https://api." + l.repoURL.Hostname()
	} else {
		apiURL = "https://" + l.repoURL.Hostname() + "/api/v3"
	}
	return apiURL
}

func getGitConfig(log logr.Logger, gitURL *url.URL) *testmachinery.GitHubInstanceConfig {
	httpURL := fmt.Sprintf("%s://%s", gitURL.Scheme, gitURL.Host)
	if testmachinery.GetConfig() == nil {
		log.V(5).Info("no testmachinery config defined")
		return nil
	}
	for i, secret := range testmachinery.GetGitHubSecrets() {
		if secret.HttpUrl == httpURL {
			log.V(5).Info(fmt.Sprintf("found secret %d for github %s", i, gitURL))
			return &secret
		}
	}
	return nil
}
