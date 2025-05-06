// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http"
)

// noAuth is a dummy implementation that implements the authentication interface
// but with no authentication
type noAuth struct{}

// NewNoAuth returns the noAuth authentication provider that is a dummy implementation that implements the authentication interface
// but with no authentication
func NewNoAuth() Provider {
	return &noAuth{}
}

func (a *noAuth) Protect(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}
}

func (a *noAuth) GetAuthContext(r *http.Request) (AuthContext, error) {
	return AuthContext{}, nil
}

func (a *noAuth) DisplayLogin() bool {
	return false
}

// Redirect is a noop redirect implementation
func (a *noAuth) Redirect(w http.ResponseWriter, r *http.Request) {}

// Login is a the noop Login implementation
func (a *noAuth) Login(w http.ResponseWriter, r *http.Request) {}

// Logout is a the noop Logout implementation
func (a *noAuth) Logout(w http.ResponseWriter, r *http.Request) {}
