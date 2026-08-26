// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package pages

import (
	"net/http"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
)

func NewHomePage(logger logr.Logger, auth auth.Provider, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return p.handleSimplePage("index.html", nil)
}

func New404Page(logger logr.Logger, auth auth.Provider, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return p.handleSimplePage("404.html", nil)
}
