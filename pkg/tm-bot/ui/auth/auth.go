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

package auth

import (
	"context"
	"encoding/gob"
	github2 "github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/google/go-github/v27/github"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	sessionName = "tm"
)

type Provider interface {
	Protect(http.HandlerFunc) http.HandlerFunc
	Redirect(w http.ResponseWriter, r *http.Request)
	GetAuthContext(r *http.Request) (AuthContext, error)
	Login(w http.ResponseWriter, r *http.Request)
}

type AuthContext struct {
	Token oauth2.Token
	User  string
}

type githubOAuth struct {
	log    logr.Logger
	store  sessions.Store
	config *oauth2.Config

	org string
}

func NewGitHubOAuth(log logr.Logger, org, clientID, clientSecret, redirectURL, cookieSecret string) *githubOAuth {
	gob.Register(AuthContext{})
	return &githubOAuth{
		log:   log,
		org:   org,
		store: sessions.NewCookieStore([]byte(cookieSecret)),
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
			RedirectURL: redirectURL,
			Scopes:      []string{"read:org"},
		},
	}
}

func (a *githubOAuth) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		defer ctx.Done()

		if a.config.ClientID == "" || a.config.ClientSecret == "" {
			a.log.Info("oauth client id or secret is not defined")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		aCtx, err := a.GetAuthContext(r)
		if err != nil {
			a.log.Error(err, "unable to get Token")

			authURL := a.config.AuthCodeURL(r.URL.String(), oauth2.AccessTypeOffline)
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			return
		}

		tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&aCtx.Token))
		client := github.NewClient(tc)

		membership, _, err := client.Organizations.GetOrgMembership(ctx, aCtx.User, a.org)
		if err != nil {
			a.log.Error(err, "unable to get org membership")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		if github2.MembershipStatus(membership.GetState()) != github2.MembershipStatusActive {
			a.log.Error(err, "user not active member of org", "user", aCtx.User, "org", a.org)
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		next(w, r)
	}
}

func (a *githubOAuth) Redirect(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	err := r.ParseForm()
	if err != nil {
		a.log.Error(err, "could not parse query")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	state := r.FormValue("state")

	httpClient := &http.Client{Timeout: 2 * time.Second}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	tok, err := a.config.Exchange(ctx, code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))
	client := github.NewClient(tc)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		a.log.Error(err, "unable to get authenticated User")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := a.store.Get(r, sessionName)
	if err != nil {
		a.log.Error(err, "unable to get session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	session.Values["context"] = AuthContext{
		Token: *tok,
		User:  user.GetLogin(),
	}
	if err := session.Save(r, w); err != nil {
		a.log.Error(err, "unable to save session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, state, http.StatusTemporaryRedirect)
}

// GetAuthContext get the Token from the cookie store
func (a *githubOAuth) GetAuthContext(r *http.Request) (AuthContext, error) {
	session, err := a.store.Get(r, sessionName)
	if err != nil {
		return AuthContext{}, errors.Wrap(err, "unable to get session store")
	}
	ctx, ok := session.Values["context"]
	if !ok {
		return AuthContext{}, errors.New("no context present")
	}
	switch c := ctx.(type) {
	case AuthContext:
		return c, nil
	default:
		return AuthContext{}, errors.New("malformed Token in cookie")
	}
}

func (a *githubOAuth) Login(w http.ResponseWriter, r *http.Request) {
	if a.config.ClientID == "" || a.config.ClientSecret == "" {
		a.log.Info("oauth client id or secret is not defined")
		http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
		return
	}
	authURL := a.config.AuthCodeURL("/", oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}
