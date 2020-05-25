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

package tm_bot

import (
	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/hook"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"net/http"
)

type options struct {
	log        logr.Logger
	restConfig *rest.Config
	cfg        *config.BotConfiguration
}

func (o *options) Complete(stopCh chan struct{}) (*mux.Router, error) {
	w, err := testrunner.StartWatchController(o.log, o.restConfig, stopCh)
	if err != nil {
		return nil, err
	}

	runs := tests.NewRuns(w)

	r := mux.NewRouter()
	r.Use(loggingMiddleware(o.log.WithName("trace")))
	r.HandleFunc("/healthz", healthz(o.log.WithName("health"))).Methods(http.MethodGet)

	if err := o.setupGitHubBot(r, runs); err != nil {
		return nil, err
	}

	o.setupDashboard(r, runs)
	return r, nil
}

func (o *options) setupDashboard(router *mux.Router, runs *tests.Runs) {
	a := auth.NewNoAuth()
	if o.cfg.Dashboard.Authentication.Enabled {
		authCfg := o.cfg.Dashboard.Authentication
		a = auth.NewGitHubOAuth(o.log.WithName("authentication"), authCfg.Organization, authCfg.OAuth.ClientID, authCfg.OAuth.ClientSecret, authCfg.OAuth.RedirectURL, authCfg.CookieSecret)
	}

	ui.Serve(o.log, runs, o.cfg.Dashboard.UIBasePath, a, router)
}

func (o *options) setupGitHubBot(router *mux.Router, runs *tests.Runs) error {
	cfg := o.cfg.GitHubBot
	if !cfg.Enabled {
		return nil
	}
	ghClient, err := github.NewManager(o.log.WithName("github"), cfg)
	if err != nil {
		return errors.Wrap(err, "unable to initialize github client")
	}
	hooks, err := hook.New(o.log.WithName("hooks"), ghClient, cfg.WebhookSecret, runs)
	if err != nil {
		return errors.Wrap(err, "unable to initialize webhooks handler")
	}

	router.HandleFunc("/events", hooks.HandleWebhook).Methods(http.MethodPost)
	return nil
}

func loggingMiddleware(log logr.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.V(10).Info(r.RequestURI, "method", r.Method)
			next.ServeHTTP(w, r)
		})
	}
}
