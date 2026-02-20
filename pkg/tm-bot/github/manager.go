// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v83/github"
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
	ghClient, err := github.NewClient(httpClient).WithEnterpriseURLs(m.apiURL, "")
	if err != nil {
		return nil, err
	}
	m.clients[installationID] = &internalClientItem{
		ghClient:   ghClient,
		httpClient: httpClient,
	}

	return m.clients[installationID], nil
}
