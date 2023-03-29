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

package pages

import (
	"html/template"
	"net/http"
	"path/filepath"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/version"
)

type Page struct {
	basePath string
	log      logr.Logger
	auth     auth.Provider
	runs     *tests.Runs
}

type globalSettings struct {
	DisplayLogin  bool
	Authenticated bool
	URL           string
	User          user
}

type baseTemplateSettings struct {
	globalSettings
	PageName  string
	Arguments interface{}
}

type user struct {
	Name string
}

func New(logger logr.Logger, runs *tests.Runs, auth auth.Provider, basePath string) *Page {
	return &Page{
		basePath: basePath,
		log:      logger,
		auth:     auth,
		runs:     runs,
	}
}

func (p *Page) handleSimplePage(templateName string, param interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAuthenticated := true
		aCtx, err := p.auth.GetAuthContext(r)
		if err != nil {
			p.log.V(3).Info(err.Error())
			isAuthenticated = false
		}
		global := globalSettings{
			DisplayLogin:  p.auth.DisplayLogin(),
			Authenticated: isAuthenticated,
			User: user{
				Name: aCtx.User,
			},
			URL: r.URL.String(),
		}

		base := filepath.Join(p.basePath, "templates", "base.html")
		components := filepath.Join(p.basePath, "templates", "components/*")
		fp := filepath.Join(p.basePath, "templates", templateName)

		tmpl := template.New(templateName)
		tmpl.Funcs(map[string]interface{}{
			"settings":     makeBaseTemplateSettings(global),
			"urlAddParams": addURLParams,
			"version":      func() string { return version.Get().String() },
		})
		tmpl.Funcs(sprig.FuncMap())
		tmpl, err = tmpl.ParseGlob(components)
		if err != nil {
			p.log.Error(err, "unable to parse components")
			return
		}
		tmpl, err = tmpl.ParseFiles(base, fp)
		if err != nil {
			p.log.Error(err, "unable to parse files")
			return
		}
		if err := tmpl.Execute(w, map[string]interface{}{
			"global": global,
			"page":   param,
		}); err != nil {
			p.log.Error(err, "unable to execute template")
		}
	}
}
