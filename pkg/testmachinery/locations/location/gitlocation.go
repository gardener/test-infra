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
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/go-logr/logr"
	"net/http"
	"net/url"
	"strings"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/google/go-github/v27/github"

	"github.com/gardener/test-infra/pkg/util"
)

var githubContentTypeFile = "file"

// GitLocation represents the testDefLocation of type "git".
type GitLocation struct {
	log  logr.Logger
	Info *tmv1beta1.TestLocation

	config    *testmachinery.GitConfig
	repoOwner string
	repoName  string
	repoURL   *url.URL
}

// NewGitLocation creates a TestDefLocation of type git.
func NewGitLocation(log logr.Logger, testDefLocation *tmv1beta1.TestLocation) (testdefinition.Location, error) {
	repoURL, err := url.Parse(testDefLocation.Repo)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url %s", testDefLocation.Repo)
	}
	config := getGitConfig(log, repoURL)
	repoOwner, repoName := parseRepoURL(repoURL)

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
	testDefs, err := l.getTestDefs()
	if err != nil {
		return err
	}
	for _, def := range testDefs {
		// Prioritize local testdefinitions over remote
		if testDefMap[def.Info.Metadata.Name] == nil || testDefMap[def.Info.Metadata.Name].Location.Type() != tmv1beta1.LocationTypeLocal {
			def.AddInputArtifacts(argov1.Artifact{
				Name: "repo",
				Path: testmachinery.TM_REPO_PATH,
			})
			testDefMap[def.Info.Metadata.Name] = def
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

func (l *GitLocation) getTestDefs() ([]*testdefinition.TestDefinition, error) {
	var definitions []*testdefinition.TestDefinition

	client, err := l.getGitHubClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create github client for %s: %s", l.Info.Repo, err.Error())
	}

	_, directoryContent, _, err := client.Repositories.GetContents(context.Background(), l.repoOwner, l.repoName,
		testmachinery.TESTDEF_PATH, &github.RepositoryContentGetOptions{Ref: l.Info.Revision})
	if err != nil {
		return nil, fmt.Errorf("no testdefinitions can be found in %s: %s", l.Info.Repo, err.Error())
	}

	for _, file := range directoryContent {
		if *file.Type == githubContentTypeFile {
			l.log.V(5).Info("found file", "filename", *file.Name, "path", *file.Path)
			data, err := util.DownloadFile(l.getHTTPClient(), file.GetDownloadURL())
			if err != nil {
				return nil, err
			}
			def, err := util.ParseTestDef(data)
			if err != nil {
				l.log.V(5).Info(fmt.Sprintf("ignoring file: %s", err.Error()), "filename", *file.Name)
				continue
			}
			if def.Kind == tmv1beta1.TestDefinitionName && def.Metadata.Name != "" {
				definition, err := testdefinition.New(&def, l, file.GetName())
				if err != nil {
					l.log.Info(fmt.Sprintf("unable to build testdefinition: %s", err.Error()), "filename", *file.Name)
					continue
				}
				definitions = append(definitions, definition)
				l.log.V(3).Info(fmt.Sprintf("found TestDefinition %s", def.Metadata.Name))
			}
		}
	}

	return definitions, nil
}

func (l *GitLocation) getGitHubClient() (*github.Client, error) {
	client, err := github.NewEnterpriseClient(l.getGitHubAPI(), "", l.getHTTPClient())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (l *GitLocation) getHTTPClient() *http.Client {
	if l.config != nil {
		basicAuth := github.BasicAuthTransport{
			Username: l.config.TechnicalUser.Username,
			Password: l.config.TechnicalUser.AuthToken,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: l.config.SkipTls},
			},
		}
		l.log.V(3).Info(fmt.Sprintf("used gitconfig for %s to authenticate", l.config.HttpUrl))
		return basicAuth.Client()
	}

	l.log.V(3).Info("insecure and unauthenticated git connection is used")
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// GitHub helper functions
func parseRepoURL(url *url.URL) (repoOwner, repoName string) {
	repoNameComponents := strings.Split(url.Path, "/")
	repoOwner = repoNameComponents[1]
	repoName = strings.Replace(repoNameComponents[2], ".git", "", 1)
	return
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

func getGitConfig(log logr.Logger, gitURL *url.URL) *testmachinery.GitConfig {
	httpURL := fmt.Sprintf("%s://%s", gitURL.Scheme, gitURL.Host)
	if testmachinery.GetConfig() == nil {
		log.V(5).Info("no testmachinery config defined")
		return nil
	}
	for i, secret := range testmachinery.GetConfig().GitSecrets {
		if secret.HttpUrl == httpURL {
			log.V(5).Info(fmt.Sprintf("found secret %d for github %s", i, gitURL))
			return &secret
		}
	}
	return nil
}
