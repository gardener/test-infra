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
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/version"
	"github.com/go-logr/logr"
	"html/template"
	"net/http"
	"path/filepath"
)

type Page struct {
	basePath string
	log      logr.Logger
	auth     auth.Authentication
}

type globalSettings struct {
	Authenticated bool
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

func makeBaseTemplateSettings(global globalSettings) func(string, interface{}) baseTemplateSettings {
	return func(pageName string, arguments interface{}) baseTemplateSettings {
		return baseTemplateSettings{
			globalSettings: global,
			PageName:       pageName,
			Arguments:      arguments,
		}
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
			Authenticated: isAuthenticated,
			User: user{
				Name: aCtx.User,
			},
		}

		base := filepath.Join(p.basePath, "templates", "base.html")
		fp := filepath.Join(p.basePath, "templates", templateName)

		tmpl := template.New(templateName)
		tmpl.Funcs(map[string]interface{}{
			"settings": makeBaseTemplateSettings(global),
			"version":  func() string { return version.Get().String() },
		})
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
