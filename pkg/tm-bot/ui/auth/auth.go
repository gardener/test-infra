// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v72/github"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	github2 "github.com/gardener/test-infra/pkg/tm-bot/github"
)

const (
	sessionName = "tm"
	// max cookie age is 24 hours
	maxAge = 24 * 60 * 60

	oauthStateCookieName = "oauthstate"
)

type Provider interface {
	// Protect is a middleware to protect a page to unauthorized access
	Protect(http.HandlerFunc) http.HandlerFunc

	// Redirect is the function that is called on a callback
	Redirect(w http.ResponseWriter, r *http.Request)

	// GetAuthContext should return the current AuthContext
	GetAuthContext(r *http.Request) (AuthContext, error)

	// Login should login a user and maybe redirect the request to a IDP
	Login(w http.ResponseWriter, r *http.Request)

	// Logout should logout a user and delete the session
	Logout(w http.ResponseWriter, r *http.Request)

	// DisplayLogin should return true if a login should be doable.
	DisplayLogin() bool
}

type AuthContext struct {
	Token oauth2.Token
	User  string
}

type githubOAuth struct {
	log    logr.Logger
	store  sessions.Store
	config *oauth2.Config

	org      string
	hostname string
}

func NewGitHubOAuth(log logr.Logger, githubHostname, org, clientID, clientSecret, redirectURL, cookieSecret string) *githubOAuth {
	gob.Register(AuthContext{})
	authURL := "https://" + githubHostname + "/login/oauth/authorize"
	tokenURL := "https://" + githubHostname + "/login/oauth/access_token"

	return &githubOAuth{
		log:      log,
		org:      org,
		hostname: githubHostname,
		store:    sessions.NewCookieStore([]byte(cookieSecret)),
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			RedirectURL: redirectURL,
			Scopes:      []string{"read:org"},
		},
	}
}

func (a *githubOAuth) DisplayLogin() bool {
	return true
}

func (a *githubOAuth) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		defer ctx.Done()

		if a.config.ClientID == "" || a.config.ClientSecret == "" {
			a.log.Info("oauth client id or secret is not defined")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		aCtx, err := a.GetAuthContext(r)
		if err != nil {
			a.log.Error(err, "unable to get Token")
			state, err := generateStateParameter()
			if err != nil {
				a.log.Error(err, "unable to generate random state")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			session, err := a.store.Get(r, sessionName)
			if err != nil {
				a.log.Error(err, "unable to get session store")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			session.AddFlash(r.RequestURI)
			if err := session.Save(r, w); err != nil {
				a.log.Error(err, "unable to save session store")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			stateCookie := http.Cookie{
				Name:    oauthStateCookieName,
				Value:   state,
				Expires: time.Now().Add(3 * time.Minute),
			}
			http.SetCookie(w, &stateCookie)
			authURL := a.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			return
		}

		tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&aCtx.Token))
		client, err := a.getGHClient(tc)
		if err != nil {
			a.log.Error(err, "failed to setup github (enterprise) client")
			http.Redirect(w, r, "/500", http.StatusInternalServerError)
			return
		}

		membership, _, err := client.Organizations.GetOrgMembership(ctx, aCtx.User, a.org)
		if err != nil {
			a.log.Error(err, "unable to get org membership")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		if github2.MembershipStatus(membership.GetState()) != github2.MembershipStatusActive {
			a.log.Error(err, "user not active member of org", "user", aCtx.User, "org", a.org)
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		next(w, r)
	}
}

func (a *githubOAuth) Redirect(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	err := r.ParseForm()
	if err != nil {
		a.log.Error(err, "could not parse query")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	state := r.FormValue("state")

	oauthState, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		a.log.Error(err, "could not get state cookie")
		http.Redirect(w, r, "/404", http.StatusBadRequest)
		return
	}
	oauthState.MaxAge = -1
	http.SetCookie(w, oauthState)

	if state != oauthState.Value {
		a.log.Error(err, "oauth state mismatch")
		http.Redirect(w, r, "/404", http.StatusBadRequest)
		return
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	tok, err := a.config.Exchange(ctx, code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))
	client, err := a.getGHClient(tc)
	if err != nil {
		a.log.Error(err, "failed to setup github (enterprise) client")
		http.Redirect(w, r, "/500", http.StatusInternalServerError)
		return
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		a.log.Error(err, "unable to get authenticated User")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session, err := a.store.Get(r, sessionName)
	if err != nil {
		a.log.Error(err, "unable to get session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if session.Options == nil {
		session.Options = &sessions.Options{}
	}

	redirectURI := "/"
	if flashes := session.Flashes(); len(flashes) > 0 {
		f := flashes[0].(string)
		if strings.HasPrefix(f, "/testrun") {
			redirectURI = f
		}
	}
	session.Options.MaxAge = maxAge
	session.Values["context"] = AuthContext{
		Token: *tok,
		User:  user.GetLogin(),
	}
	if err := session.Save(r, w); err != nil {
		a.log.Error(err, "unable to save session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}

// GetAuthContext get the Token from the cookie store
func (a *githubOAuth) GetAuthContext(r *http.Request) (AuthContext, error) {
	session, err := a.store.Get(r, sessionName)
	if err != nil {
		return AuthContext{}, errors.Wrap(err, "unable to get session store")
	}
	ctx, ok := session.Values["context"]
	if !ok {
		return AuthContext{}, errors.New("no context present")
	}
	switch c := ctx.(type) {
	case AuthContext:
		return c, nil
	default:
		return AuthContext{}, errors.New("malformed Token in cookie")
	}
}

func (a *githubOAuth) Login(w http.ResponseWriter, r *http.Request) {
	if a.config.ClientID == "" || a.config.ClientSecret == "" {
		a.log.Info("oauth client id or secret is not defined")
		http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
		return
	}

	state, err := generateStateParameter()
	if err != nil {
		a.log.Error(err, "unable to generate random state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	stateCookie := http.Cookie{
		Name:    oauthStateCookieName,
		Value:   state,
		Expires: time.Now().Add(3 * time.Minute),
	}

	http.SetCookie(w, &stateCookie)
	authURL := a.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (a *githubOAuth) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := a.store.Get(r, sessionName)
	if err != nil {
		a.log.Error(err, "unable to get session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		a.log.Error(err, "unable to save session store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// getGHClient returns a GitHub client or GitHub enterprise client based on the hostname of the oauth config
func (a *githubOAuth) getGHClient(client *http.Client) (*github.Client, error) {
	if a.hostname != "github.com" {
		githubUrl := "https://" + a.hostname
		return github.NewClient(client).WithEnterpriseURLs(githubUrl, "")
	}
	return github.NewClient(client), nil
}

func generateStateParameter() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	return state, nil
}
