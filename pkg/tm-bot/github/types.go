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

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v39/github"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/tm-bot/github/ghval"
)

type Manager interface {
	GetClient(event *GenericRequestEvent) (Client, error)
}

// Client is the github client interface
type Client interface {
	Client() *github.Client

	GetHead(ctx context.Context, event *GenericRequestEvent) (string, error)
	GetIssue(event *GenericRequestEvent) (*github.Issue, error)
	GetPullRequest(ctx context.Context, event *GenericRequestEvent) (*github.PullRequest, error)
	GetVersions(owner, repo string) ([]*semver.Version, error)
	GetContent(ctx context.Context, event *GenericRequestEvent, path string) ([]byte, error)

	IsAuthorized(authorizationType AuthorizationType, event *GenericRequestEvent) bool

	GetConfig(name string, obj interface{}) error
	GetRawConfig(name string) (json.RawMessage, error)
	ResolveConfigValue(ctx context.Context, event *GenericRequestEvent, value *ghval.GitHubValue) (string, error)

	UpdateComment(event *GenericRequestEvent, commentID int64, message string) error
	Comment(ctx context.Context, event *GenericRequestEvent, message string) (int64, error)
	UpdateStatus(ctx context.Context, event *GenericRequestEvent, state State, statusContext, description string) error
}

// GenericRequestEvent is the generic request from github triggering the tm bot
type GenericRequestEvent struct {
	// InstallationID is the github app ID
	InstallationID int64

	// ID is the unique github id of the comment
	ID int64

	// Number is the number of the PR
	Number int

	// Head is the sha of the current PR's head commit
	Head string

	// Repository is the event's source repository
	Repository *github.Repository

	// Body comprises the message body of the commit
	Body string

	// Author is the event's author
	Author *github.User
}

// RepositoryKey is the unique name for a repository
type RepositoryKey struct {
	Owner      string
	Repository string
}

type manager struct {
	log         logr.Logger
	cacheConfig *config.GitHubCache
	configFile  string

	apiURL      string
	appId       int64
	keyFile     string
	clients     map[int64]*internalClientItem
	defaultTeam string
}

type client struct {
	log        logr.Logger
	config     map[string]json.RawMessage
	client     *github.Client
	httpClient *http.Client

	owner       string
	defaultTeam *github.Team
}

// AuthorizationType represents the usergroup that is allowed to do the action
type AuthorizationType string

const (
	AuthorizationAll        AuthorizationType = "all"
	AuthorizationOrg        AuthorizationType = "org"
	AuthorizationTeam       AuthorizationType = "team"
	AuthorizationCodeOwners AuthorizationType = "codeowners"
	AuthorizationOrgAdmin   AuthorizationType = "org-admin"
)

// EventActionType represents the action type of a github event
type EventActionType string

const (
	EventActionTypeCreated EventActionType = "created"
	EventActionTypeDeleted EventActionType = "deleted"
	EventActionTypeEdited  EventActionType = "edited"
)

// UserType represents the type of an owner
type UserType string

const (
	UserTypeUser         UserType = "User"
	UserTypeBot          UserType = "Bot"
	UserTypeOrganization UserType = "Organization"
)

// Status state of a commit
type State string

const (
	StateError   State = "error"
	StateFailure State = "failure"
	StatePending State = "pending"
	StateSuccess State = "success"
)

// MembershipRole represents the membership role of organizations and teams
type MembershipRole string

const (
	MembershipRoleAdmin      MembershipRole = "admin"
	MembershipRoleMember     MembershipRole = "member"
	MembershipRoleMaintainer MembershipRole = "maintainer"
)

// MembershipStatus represents the current membership status of a user
type MembershipStatus string

const (
	MembershipStatusActive MembershipStatus = "active"
)

// ContentType represents the type of a content response
type ContentType string

const (
	ContentTypeFile = "file"
	ContentTypeDir  = "dir"
)
