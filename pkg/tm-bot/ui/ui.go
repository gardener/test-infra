// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"net/http"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"

	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/pages"
)

func Serve(log logr.Logger, runs *tests.Runs, basePath string, a auth.Provider, r *mux.Router) {
	fs := http.FileServer(http.Dir(filepath.Join(basePath, "static")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/oauth/redirect", a.Redirect)
	r.HandleFunc("/login", a.Login)
	r.HandleFunc("/logout", a.Logout)

	page := pages.New(log, runs, a, basePath)

	r.HandleFunc("/command-help", pages.NewCommandHelpPage(log, a, basePath))
	r.HandleFunc("/command-help/{plugin}", pages.NewCommandDetailedHelpPage(log, a, basePath))
	r.HandleFunc("/pr-status", pages.NewPRStatusPage(page))
	r.HandleFunc("/pr-status/{testrun}", pages.NewPRStatusDetailPage(log, a, basePath))
	r.HandleFunc("/testruns", a.Protect(pages.NewTestrunsPage(page)))
	r.HandleFunc("/testrun/{namespace}/{testrun}", a.Protect(pages.NewTestrunPage(page)))
	r.HandleFunc("/404", pages.New404Page(log, a, basePath))
	r.HandleFunc("/", pages.NewHomePage(log, a, basePath))
}
