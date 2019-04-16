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

package testdefinition

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/google/go-github/github"

	"github.com/gardener/test-infra/pkg/util"
)

var githubContentTypeFile = "file"

// GitLocation represents the testDefLocation of type "git".
type GitLocation struct {
	info *tmv1beta1.TestLocation

	config    *testmachinery.GitConfig
	repoOwner string
	repoName  string
	repoURL   *url.URL
}

// NewGitLocation creates a TestDefLocation of type git.
func NewGitLocation(testDefLocation *tmv1beta1.TestLocation) (Location, error) {
	repoURL, err := url.Parse(testDefLocation.Repo)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse url %s", testDefLocation.Repo)
	}
	config := getGitConfig(repoURL)
	repoOwner, repoName := parseRepoURL(repoURL)

	return &GitLocation{testDefLocation, config, repoOwner, repoName, repoURL}, nil
}

// SetTestDefs adds its TestDefinitions to the TestDefinition Map.
func (l *GitLocation) SetTestDefs(testDefMap map[string]*TestDefinition) error {
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
	return l.info
}

// Name returns the unique name of the git location consiting of the repositorie's owner, name and revision.
func (l *GitLocation) Name() string {
	name := fmt.Sprintf("%s-%s-%s", l.repoOwner, l.repoName, l.info.Revision)
	return util.FormatArtifactName(name)
}

// Type returns the tmv1beta1.LocationTypeGit.
func (l *GitLocation) Type() tmv1beta1.LocationType {
	return tmv1beta1.LocationTypeGit
}

func (l *GitLocation) getTestDefs() ([]*TestDefinition, error) {
	var definitions []*TestDefinition

	client, err := l.getGitHubClient()
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("No testdefinitions found in %s", l.info.Repo)
	}

	_, directoryContent, _, err := client.Repositories.GetContents(context.Background(), l.repoOwner, l.repoName,
		testmachinery.TESTDEF_PATH, &github.RepositoryContentGetOptions{Ref: l.info.Revision})
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("No testdefinitions found in %s", l.info.Repo)
	}

	for _, file := range directoryContent {
		if *file.Type == githubContentTypeFile {
			log.Debugf("Found file %s in Path: %s", *file.Name, *file.Path)
			data, err := util.DownloadFile(l.getHTTPClient(), file.GetDownloadURL())
			if err != nil {
				return nil, err
			}
			def, err := util.ParseTestDef(data)
			if err == nil {
				if def.Kind == tmv1beta1.TestDefinitionName {
					if def.Kind == tmv1beta1.TestDefinitionName && def.Metadata.Name != "" {
						definition, err := New(&def, l, file.GetName())
						if err != nil {
							log.Debugf("cannot build testdefinition for %s: %s", *file.Name, err.Error())
							continue
						}
						definitions = append(definitions, definition)
						log.Debugf("found Testdefinition %s", def.Metadata.Name)
					}
				}
			} else {
				log.Debugf("Ignoring file %s : %s", *file.Name, err.Error())
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
		log.Debugf("Used gitconfig for %s to authenticate", l.config.HttpUrl)
		return basicAuth.Client()
	}

	log.Warn("Insecure and unauthenticated git connection is used")
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

func getGitConfig(gitURL *url.URL) *testmachinery.GitConfig {
	httpURL := fmt.Sprintf("%s://%s", gitURL.Scheme, gitURL.Host)
	if testmachinery.GetConfig() == nil {
		return nil
	}
	for _, secret := range testmachinery.GetConfig().GitSecrets {
		if secret != nil && secret.HttpUrl == httpURL {
			return secret
		}
	}
	return nil
}
