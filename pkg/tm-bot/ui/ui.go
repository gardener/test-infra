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

package ui

import (
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/pages"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
)

func Serve(log logr.Logger, runs *tests.Runs, basePath string, a auth.Provider, r *mux.Router) {
	fs := http.FileServer(http.Dir(filepath.Join(basePath, "static")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/oauth/redirect", a.Redirect)
	r.HandleFunc("/login", a.Login)

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
