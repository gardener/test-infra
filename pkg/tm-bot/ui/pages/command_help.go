// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package pages

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
)

var AuthorizationTooltip = map[github.AuthorizationType]string{
	github.AuthorizationAll:        "Everyone is allowed to use the command",
	github.AuthorizationOrg:        "Everyone that is member of the org is allowed to use the command",
	github.AuthorizationTeam:       "Everyone that is in the default team is allowed to use the command",
	github.AuthorizationCodeOwners: "Only codeowners are allowed to use command. If no code owner exists the default team is used.",
}

type CommandHelpItem struct {
	Command       string
	Description   string
	Example       string
	Authorization github.AuthorizationType
}

type CommandHelpDetailedItem struct {
	CommandHelpItem
	AuthorizationTooltip string
	Usage                string
	Config               string
}

func NewCommandHelpPage(logger logr.Logger, auth auth.Provider, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		rawList := make([]CommandHelpItem, len(plugins.GetAll()))
		for i, plugin := range plugins.GetAll() {
			rawList[i] = CommandHelpItem{
				Command:       plugin.Command(),
				Description:   plugin.Description(),
				Example:       plugin.Example(),
				Authorization: plugin.Authorization(),
			}
		}
		params := map[string]interface{}{
			"plugins": rawList,
		}

		p.handleSimplePage("command-help.html", params)(w, r)
	}
}

func NewCommandDetailedHelpPage(logger logr.Logger, auth auth.Provider, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		pluginName := mux.Vars(r)["plugin"]
		_, plugin, err := plugins.Get(pluginName)
		if err != nil {
			logger.Error(err, "cannot get plugin")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		params := CommandHelpDetailedItem{
			CommandHelpItem: CommandHelpItem{
				Command:       plugin.Command(),
				Description:   plugin.Description(),
				Example:       plugin.Example(),
				Authorization: plugin.Authorization(),
			},
			AuthorizationTooltip: AuthorizationTooltip[plugin.Authorization()],
			Usage:                plugin.Flags().FlagUsages(),
			Config:               plugin.Config(),
		}
		p.handleSimplePage("command-help-detailed.html", params)(w, r)
	}
}
