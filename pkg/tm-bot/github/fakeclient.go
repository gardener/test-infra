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
	"encoding/json"
	"github.com/gardener/test-infra/pkg/tm-bot/github/ghval"
	"github.com/google/go-github/v27/github"
)

type fakeClient struct{}

func NewFakeClient() (Client, error) {
	return &fakeClient{}, nil
}

// Client returns the current github client
func (c *fakeClient) Client() *github.Client {
	return &github.Client{}
}

// GetConfig returns the repository configuration for a specific command
func (c *fakeClient) GetConfig(name string) (json.RawMessage, error) {
	return []byte{}, nil
}

// ResolveConfigValue determines a GitHub config value and returns the referenced
// raw value, file content or commit hash as string
func (c *fakeClient) ResolveConfigValue(event *GenericRequestEvent, value *ghval.GitHubValue) (string, error) {
	return "", nil
}

// UpdateComment edits specific comment and overwrites its message
func (c *fakeClient) UpdateComment(event *GenericRequestEvent, commentID int64, message string) error {
	return nil
}

// Comment responds to an event
func (c *fakeClient) Comment(event *GenericRequestEvent, message string) (int64, error) {
	return 0, nil
}

// UpdateStatus updates the status check for a pull request
func (c *fakeClient) UpdateStatus(event *GenericRequestEvent, state State, statusContext, description string) error {
	return nil
}

// IsAuthorized checks if the author of the event is authorized to perform actions on the service
func (c *fakeClient) IsAuthorized(authorizationType AuthorizationType, event *GenericRequestEvent) bool {
	return false
}

// GetPullRequest fetches the pull request for a event
func (c *fakeClient) GetPullRequest(event *GenericRequestEvent) (*github.PullRequest, error) {
	return &github.PullRequest{}, nil
}

// GetPullRequest fetches the issue for a event
func (c *fakeClient) GetIssue(event *GenericRequestEvent) (*github.Issue, error) {
	return &github.Issue{}, nil
}
