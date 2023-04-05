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

package github

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v50/github"
	"sigs.k8s.io/yaml"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery/ghcache"
)

type internalClientItem struct {
	ghClient   *github.Client
	httpClient *http.Client
}

func NewManager(log logr.Logger, cfg config.GitHubBot) (Manager, error) {
	return &manager{
		log:         log,
		cacheConfig: &cfg.GitHubCache,
		configFile:  cfg.ConfigurationFilePath,
		apiURL:      cfg.ApiUrl,
		appId:       cfg.AppID,
		keyFile:     cfg.AppPrivateKeyPath,
		defaultTeam: cfg.DefaultTeam,
		clients:     make(map[int64]*internalClientItem),
	}, nil
}

func (m *manager) GetClient(event *GenericRequestEvent) (Client, error) {

	intClient, err := m.getGitHubClient(event.InstallationID)
	if err != nil {
		return nil, err
	}
	config, err := m.getConfig(intClient.ghClient, event.GetOwnerName(), event.GetRepositoryName(), event.Repository.GetDefaultBranch())
	if err != nil {
		return nil, err
	}

	return NewClient(m.log, intClient.ghClient, intClient.httpClient, event.GetOwnerName(), m.defaultTeam, config)
}

func (m *manager) getConfig(c *github.Client, repo, owner, revision string) (map[string]json.RawMessage, error) {
	ctx := context.Background()
	defer ctx.Done()
	file, dir, _, err := c.Repositories.GetContents(ctx, repo, owner, m.configFile, &github.RepositoryContentGetOptions{Ref: revision})
	if err != nil {
		m.log.Error(err, "unable to get config", "owner", owner, "repo", repo, "revision", revision)
		return nil, nil
	}
	if len(dir) != 0 {
		m.log.Info("config path is a directory not a file", "owner", owner, "repo", repo, "revision", revision)
		return nil, nil
	}

	content, err := file.GetContent()
	if err != nil {
		return nil, err
	}

	var config map[string]json.RawMessage
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, err
	}

	return config, err
}

func (m *manager) getGitHubClient(installationID int64) (*internalClientItem, error) {
	if ghClient, ok := m.clients[installationID]; ok {
		return ghClient, nil
	}

	trp, err := ghcache.WithRateLimitControlCache(m.log.WithName("ghCache"), http.DefaultTransport)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.NewKeyFromFile(trp, m.appId, installationID, m.keyFile)
	if err != nil {
		return nil, err
	}
	itr.BaseURL = m.apiURL

	httpClient := &http.Client{Transport: itr}
	ghClient, err := github.NewEnterpriseClient(m.apiURL, "", httpClient)
	if err != nil {
		return nil, err
	}
	m.clients[installationID] = &internalClientItem{
		ghClient:   ghClient,
		httpClient: httpClient,
	}

	return m.clients[installationID], nil
}
