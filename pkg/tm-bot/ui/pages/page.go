// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
