// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"net/http"
)

// dummyAuth is a dummy implementation that implements the authentication interface
// but with a dummy user.
// This auth method should only be used for local development
type dummyAuth struct {
	loggedIn bool
}

// NewDummyAuth returns a dummy implementation that implements the authentication interface
// but with a dummy user.
// This auth method should only be used for local development
func NewDummyAuth() Provider {
	return &dummyAuth{}
}

func (a *dummyAuth) Protect(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.loggedIn {
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
		}
		handler(w, r)
	}
}

func (a *dummyAuth) GetAuthContext(r *http.Request) (AuthContext, error) {
	if a.loggedIn {
		return AuthContext{
			User: "demo",
		}, nil
	}
	return AuthContext{}, errors.New("user not logged in")
}

func (a *dummyAuth) DisplayLogin() bool {
	return true
}

func (a *dummyAuth) Redirect(w http.ResponseWriter, r *http.Request) {}

func (a *dummyAuth) Login(w http.ResponseWriter, r *http.Request) {
	a.loggedIn = true
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
func (a *dummyAuth) Logout(w http.ResponseWriter, r *http.Request) {
	a.loggedIn = false
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
