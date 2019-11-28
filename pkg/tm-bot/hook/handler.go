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

package hook

import (
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/echo"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/resume"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/skip"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/xkcd"
	tests2 "github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/pkg/errors"
	"net/http"

	ghutils "github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
)

type Handler struct {
	log logr.Logger

	ghMgr              ghutils.Manager
	webhookSecretToken []byte
}

func New(log logr.Logger, ghMgr ghutils.Manager, webhookSecretToken string, watch watch.Watch, runs *tests2.Runs) (*Handler, error) {

	persistence, err := plugins.NewKubernetesPersistence(watch.Client(), "state", "ts-test")
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup plugin persistence")
	}
	plugins.Setup(log.WithName("plugins"), persistence)

	// register plugins.Plugin()
	plugins.Register(echo.New())
	xkcdPlugin, err := xkcd.New()
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize xkcd plugin")
	}
	plugins.Register(xkcdPlugin)

	plugins.Register(tests.New(log, runs))
	plugins.Register(skip.New(log))
	plugins.Register(resume.New(log, watch.Client()))

	if err := plugins.ResumePlugins(ghMgr); err != nil {
		return nil, errors.Wrap(err, "unable to resume running plugins")
	}

	return &Handler{
		log:                log,
		ghMgr:              ghMgr,
		webhookSecretToken: []byte(webhookSecretToken),
	}, nil
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, h.webhookSecretToken)
	if err != nil {
		h.log.Error(err, "payload validation failed")
		http.Error(w, "validation failed", http.StatusInternalServerError)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		h.log.Error(err, "unable to parse webhook")
		http.Error(w, "unable to parse webhook", http.StatusInternalServerError)
		return
	}

	switch event := event.(type) {
	case *github.IssueCommentEvent:
		if event.GetIssue().IsPullRequest() && ghutils.EventActionType(event.GetAction()) == ghutils.EventActionTypeCreated {
			h.handleGenericEvent(w, &ghutils.GenericRequestEvent{
				InstallationID: event.GetInstallation().GetID(),
				ID:             event.GetIssue().GetID(),
				Number:         event.GetIssue().GetNumber(),
				Repository:     event.GetRepo(),
				Body:           event.GetComment().GetBody(),
				Author:         event.GetComment().GetUser(),
			})
		}
		break
	default:
		http.Error(w, "event not handled", http.StatusNoContent)
		return
	}

	if _, err := w.Write([]byte{}); err != nil {
		h.log.Error(err, "unable to send response to github")
	}
}

func (h *Handler) handleGenericEvent(w http.ResponseWriter, event *ghutils.GenericRequestEvent) {
	h.log.V(5).Info("handle generic event", "user", event.GetAuthorName(), "id", event.ID, "number", event.Number)

	// ignore messages from bots
	if ghutils.UserType(*event.Author.Type) != ghutils.UserTypeUser {
		return
	}

	client, err := h.ghMgr.GetClient(event)
	if err != nil {
		h.log.Error(err, "unable to build client", "user", event.GetAuthorName())
		http.Error(w, "internal error", http.StatusUnauthorized)
		return
	}

	go func() {
		if err := plugins.HandleRequest(client, event); err != nil {
			h.log.Error(err, "")
		}
	}()
}
