// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
func (_ *noAuth) Redirect(w http.ResponseWriter, r *http.Request) {}

// Login is a the noop Login implementation
func (a *noAuth) Login(w http.ResponseWriter, r *http.Request) {}

// Logout is a the noop Logout implementation
func (a *noAuth) Logout(w http.ResponseWriter, r *http.Request) {}
