// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains the testmachinery configuration values
type BotConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	Webserver       Webserver `json:"webserver"`
	Dashboard       Dashboard `json:"dashboard"`
	GitHubBot       GitHubBot `json:"githubBot"`
}

// Webserver configures the webserver that servres the bot and the dashboard
type Webserver struct {
	// HTTPPort specifies the port to listen for http traffic
	HTTPPort int `json:"httpPort"`

	// HTTPSPort specifies the port to listen for https traffic
	HTTPSPort int `json:"httpsPort"`

	// Certificate holds the certificate the should be used to server the https traffic
	// +optional
	Certificate Certificate `json:"certificate"`
}

// Certificate holds the certificate and its the private key
type Certificate struct {
	// Cert specifies the path to the certificate file
	Cert string `json:"cert"`

	// PrivateKey specifies the path to the private key file
	PrivateKey string `json:"privateKey"`
}

// Dashboard contains configuration values for the TestMachinery Dashboard
type Dashboard struct {
	// UIBasePath specifies the base path for static files and templates
	UIBasePath string `json:"UIBasePath"`

	// Authentication to restrict access to specific parts in the dashboard
	Authentication DashboardAuthentication `json:"authentication"`
}

// DashboardAuthenticationProvider is a enum to specify a dashboard authentication method
type DashboardAuthenticationProvider string

const (
	GitHubAuthProvider DashboardAuthenticationProvider = "github"
	NoAuthProvider     DashboardAuthenticationProvider = "noauth"
	DummyAuthProvider  DashboardAuthenticationProvider = "dummy"
)

// DashboardAuthentication to restrict access to specific parts in the dashboard
type DashboardAuthentication struct {
	// Provider defines the authentication provider that should be used to authenticate and authorize users
	// to view testruns.
	Provider DashboardAuthenticationProvider `json:"provider"`

	// CookieSecret is the secret for the cookie store
	// +optional
	CookieSecret string `json:"cookieSecret"`

	// GitHub holds the github provider specific configuration
	// +optional
	GitHub *GitHubAuthentication `json:"githubConfig"`
}

type GitHubAuthentication struct {
	// OAuth Github configuration that is used to protect parts of the dashboard
	// +optional
	OAuth *OAuth `json:"oAuth"`

	// Organization is the GitHub organization to restrict access to the bot
	// +optional
	Organization string `json:"organization"`
}

type OAuth struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURL  string `json:"redirectUrl"`
}

// GitHubBot contains the configuration for the github integration
type GitHubBot struct {
	// Enabled defines if the GitHub Bot integration should be enabled
	Enabled bool `json:"enabled"`

	// ConfigurationFilePath specifies the path to the configuration inside a repository that is watched by the bot
	ConfigurationFilePath string `json:"configurationFilePath"`

	// DefaultTeam is the slug name of the default team to grant permissions to perform bot commands
	DefaultTeam string `json:"defaultTeam"`

	// ApiUrl specifies the github api endpoint
	ApiUrl string `json:"apiUrl"`

	// AppID holds the ID of the GitHub App.
	AppID int `json:"appId"`

	// AppPrivateKeyPath is the path to the private key for the GitHub app.
	AppPrivateKeyPath string `json:"appPrivateKeyPath"`

	// GitHub webhook secret to verify payload
	WebhookSecret string `json:"webhookSecret"`

	// GitHubCache configures the cache for the github api
	GitHubCache GitHubCache `json:"cache"`
}
