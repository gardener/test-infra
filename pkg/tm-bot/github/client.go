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
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	comerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/tm-bot/github/ghval"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
)

func NewClient(log logr.Logger, ghClient *github.Client, httpClient *http.Client, owner, defaultTeamName string, config map[string]json.RawMessage) (Client, error) {
	c := &client{
		log:        log,
		config:     config,
		client:     ghClient,
		httpClient: httpClient,
		owner:      owner,
	}

	if defaultTeamName != "" {
		var err error
		c.defaultTeam, _, err = ghClient.Teams.GetTeamBySlug(context.TODO(), owner, defaultTeamName)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Client returns the current github client
func (c *client) Client() *github.Client {
	return c.client
}

// GetRawConfig returns the repository configuration for a specific command
func (c *client) GetRawConfig(name string) (json.RawMessage, error) {
	config, ok := c.config[name]
	if !ok {
		c.log.V(3).Info("no config found", "plugin", name)
		return nil, comerrors.NewNotFoundError(fmt.Sprintf("no config found for %s", name))
	}
	return config, nil
}

// GetConfig parses the repository configuration for a specific command
func (c *client) GetConfig(name string, obj interface{}) error {
	raw, err := c.GetRawConfig(name)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(raw, obj); err != nil {
		return errors.Wrapf(err, "unable to unmarshal config for %s", name)
	}

	return nil
}

// ResolveConfigValue determines a GitHub config value and returns the referenced
// raw value, file content or commit hash as string
func (c *client) ResolveConfigValue(ctx context.Context, event *GenericRequestEvent, value *ghval.GitHubValue) (string, error) {
	if value.Value != nil {
		return *value.Value, nil
	}
	if value.PRHead != nil && *value.PRHead {
		return event.Head, nil
	}
	if value.Path != nil {
		rawContent, err := c.GetContent(ctx, event, *value.Path)
		if err != nil {
			return "", err
		}

		content := string(rawContent)
		if value.StructuredJSONPath != nil {
			var val interface{}
			_, err := util.RawJSONPath([]byte(content), *value.StructuredJSONPath, &val)
			if err != nil {
				return "", err
			}

			switch v := val.(type) {
			case string:
				return v, nil
			default:
				yamlData, err := yaml.Marshal(v)
				if err != nil {
					return "", err
				}
				return string(yamlData), nil
			}
		}

		return content, nil
	}
	return "", pluginerr.New("no value is defined", "no value is defined")
}

// UpdateComment edits specific comment and overwrites its message
func (c *client) UpdateComment(event *GenericRequestEvent, commentID int64, message string) error {
	_, _, err := c.client.Issues.EditComment(context.TODO(), event.GetOwnerName(), event.GetRepositoryName(), commentID, &github.IssueComment{
		Body: &message,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to edit comment")
	}

	return nil
}

// Comment responds to an event
func (c *client) Comment(ctx context.Context, event *GenericRequestEvent, message string) (int64, error) {
	comment, _, err := c.client.Issues.CreateComment(ctx, event.GetOwnerName(), event.GetRepositoryName(), event.Number, &github.IssueComment{
		Body: &message,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "unable to respond to request")
	}

	return comment.GetID(), nil
}

// UpdateStatus updates the status check for a pull request
func (c *client) UpdateStatus(ctx context.Context, event *GenericRequestEvent, state State, statusContext, description string) error {
	stateString := string(state)
	_, _, err := c.client.Repositories.CreateStatus(ctx, event.GetOwnerName(), event.GetRepositoryName(), event.Head, &github.RepoStatus{
		State:       &stateString,
		Description: &description,
		Context:     &statusContext,
	})
	return err
}

// GetContent downloads the content of the file for the given path
func (c *client) GetContent(ctx context.Context, event *GenericRequestEvent, path string) ([]byte, error) {
	contentRes, _, _, err := c.client.Repositories.GetContents(ctx, event.GetOwnerName(), event.GetRepositoryName(), path, &github.RepositoryContentGetOptions{Ref: event.Head})
	if err != nil {
		return nil, err
	}

	if contentRes.Type == nil || *contentRes.Type != ContentTypeFile {
		return nil, comerrors.NewWrongTypeError("found file is not of expected type")
	}

	return util.DownloadFile(c.httpClient, *contentRes.DownloadURL)
}

// GetPullRequest fetches the pull request for a event
func (c *client) GetPullRequest(ctx context.Context, event *GenericRequestEvent) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, event.GetOwnerName(), event.GetRepositoryName(), event.Number)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// GetHead returns the head commit for the event
func (c *client) GetHead(ctx context.Context, event *GenericRequestEvent) (string, error) {
	pr, err := c.GetPullRequest(ctx, event)
	if err != nil {
		return "", err
	}
	return pr.GetHead().GetSHA(), nil
}

// GetPullRequest fetches the issue for a event
func (c *client) GetIssue(event *GenericRequestEvent) (*github.Issue, error) {
	issue, _, err := c.client.Issues.Get(context.TODO(), event.GetOwnerName(), event.GetRepositoryName(), event.Number)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func (c *client) GetVersions(owner, repo string) ([]*semver.Version, error) {
	tags := make([]*semver.Version, 0)
	opts := &github.ListOptions{PerPage: 50}
	for {
		rawTags, res, err := c.client.Repositories.ListTags(context.TODO(), owner, repo, opts)
		if err != nil {
			return nil, err
		}

		for _, rawTag := range rawTags {
			version, err := semver.NewVersion(rawTag.GetName())
			if err != nil {
				continue
			}
			tags = append(tags, version)
		}

		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}

	return tags, nil
}
