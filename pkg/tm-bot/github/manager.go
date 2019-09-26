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
	"sigs.k8s.io/yaml"

	"github.com/go-logr/logr"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v27/github"
)

func NewManager(log logr.Logger, appID int, keyFile, configFile string) (Manager, error) {
	return &manager{
		log:        log,
		configFile: configFile,
		appId:      appID,
		keyFile:    keyFile,
		clients:    make(map[int64]*github.Client, 0),
	}, nil
}

func (m *manager) GetClient(event *GenericRequestEvent) (Client, error) {

	ghClient, err := m.getGitHubClient(event.InstallationID)
	if err != nil {
		return nil, err
	}
	config, err := m.getConfig(ghClient, event.GetOwnerName(), event.GetRepositoryName(), event.Repository.GetDefaultBranch())
	if err != nil {
		return nil, err
	}

	return NewClient(m.log, ghClient, config)
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

func (m *manager) getGitHubClient(installationID int64) (*github.Client, error) {
	if ghClient, ok := m.clients[installationID]; ok {
		return ghClient, nil
	}
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, m.appId, int(installationID), m.keyFile)
	if err != nil {
		return nil, err
	}

	m.clients[installationID] = github.NewClient(&http.Client{Transport: itr})

	return m.clients[installationID], nil
}
