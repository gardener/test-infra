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
	"github.com/pkg/errors"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v27/github"
)

func New(log logr.Logger, appID int, keyFile string) (Client, error) {
	return &client{
		log:     log,
		appId:   appID,
		keyFile: keyFile,
		clients: make(map[int64]*github.Client, 0),
	}, nil
}

func (c *client) GetClient(installationID int64) (*github.Client, error) {
	if ghClient, ok := c.clients[installationID]; ok {
		return ghClient, nil
	}
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, c.appId, int(installationID), c.keyFile)
	if err != nil {
		return nil, err
	}

	c.clients[installationID] = github.NewClient(&http.Client{Transport: itr})
	return c.clients[installationID], nil
}

// Respond responds to an event
func (c *client) Respond(event *GenericRequestEvent, message string) error {
	ghClient, err := c.GetClient(event.InstallationID)
	if err != nil {
		return err
	}

	_, _, err = ghClient.Issues.CreateComment(context.TODO(), event.GetOwnerName(), event.GetRepositoryName(), event.Number, &github.IssueComment{
		Body: &message,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to respond to request")
	}

	return nil
}

// IsAuthorized checks if the author of the event is authorized to perform actions on the service
func (c *client) IsAuthorized(event *GenericRequestEvent) bool {
	if UserType(*event.Author.Type) == UserTypeBot {
		return false
	}

	gh, err := c.GetClient(event.InstallationID)
	if err != nil {
		c.log.V(3).Info(err.Error())
		return false
	}

	membership, _, err := gh.Organizations.GetOrgMembership(context.TODO(), event.GetAuthorName(), event.Repository.GetOwner().GetLogin())
	if err != nil {
		c.log.V(3).Info(err.Error())
		return false
	}
	if *membership.State != "active" {
		return false
	}
	return true
}
