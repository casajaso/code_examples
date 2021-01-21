/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/coreos/go-oidc"
	"github.com/pkg/browser"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type OktaSessionToken struct {
	SessionToken *oauth2.Token
	IDToken      string
	ClientID     string
	OktaServerUS string
	Stage        string
	Environment  string
}

// OktaOIDCClient data
type OktaOIDCClient struct {
	OktaServerUS  *url.URL
	Organization  string
	ClientID      string
	State         string
	CodeVerifier  string
	CodeChallenge string
	Redirect      *url.URL
	Success       *url.URL
	Failure       *url.URL
	Stage         string
	Environment   string
	oidc.Provider
}

// NewOktaOIDCClient create a new Okta OIDC Client
func (p *OktaProvider) NewOktaOIDCClient() (*OktaOIDCClient, error) {
	state, err := randomHex(20)
	if err != nil {
		return nil, fmt.Errorf("Failed to define PKCE `state` : (%s)", err)
	}

	codeVerifier, codeChallenge, err := pkce()
	if err != nil {
		return nil, fmt.Errorf("Failed to define PKCE `code - challenge/verifier`: (%s)", err)
	}

	return &OktaOIDCClient{
		OktaServerUS:  p.Config.OIDC.OktaServerUS,
		ClientID:      p.Config.OIDC.ClientID,
		Organization:  p.Config.OIDC.Organization,
		State:         state,
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Redirect:      p.Config.OIDC.Redirect,
		Success:       p.Config.OIDC.Success,
		Failure:       p.Config.OIDC.Failure,
		Stage:         p.Config.Stage,
		Environment:   p.Config.Environment,
	}, nil
}

// Authenticate redirects user to browser login for Okta authentication
func (o *OktaOIDCClient) Authenticate(config *Config) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	defer cancel()

	provider, err := oidc.NewProvider(ctx, o.OktaServerUS.String())
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize: `oidc provider` (%s)", err)
	}

	oidcConfig := &oidc.Config{
		ClientID: o.ClientID,
	}

	verifier := provider.Verifier(oidcConfig)

	oauth2Config := &oauth2.Config{
		ClientID:    o.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: o.Redirect.String(),
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
			"email",
		},
	}

	credChan := make(chan *[]byte, 1)

	o.redirectServer(ctx, cancel, credChan, *oauth2Config, verifier)

	srv := o.startRedirectServer()

	defer o.stopRedirectServer(srv)

	var newCreds []byte

	err = restoreWindowFocusAfter(func() error {

		err := browser.OpenURL(oauth2Config.AuthCodeURL(o.State,
			oauth2.SetAuthURLParam("code_challenge", o.CodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		))
		if err != nil {
			return err
		}

		select {
		case creds := <-credChan:
			newCreds = *creds
		case <-ctx.Done():
			return fmt.Errorf("Terminated before completing request: (%s)", ctx.Err())
		}

		return nil
	})
	return newCreds, err
}

// setRedirectHandler handles the redirect from Okta to the authentication server
func (o *OktaOIDCClient) redirectServer(
	ctx context.Context,
	cancel context.CancelFunc,
	credChan chan *[]byte,
	config oauth2.Config,
	verifier *oidc.IDTokenVerifier,
) error {
	http.HandleFunc(o.Redirect.Path, func(w http.ResponseWriter, r *http.Request) {

		defer cancel()

		defer func() {
			var err error
			if err != nil {
				w.Write([]byte(fmt.Sprintf("<script>window.location = '%s'</script>", o.Failure.String())))
			} else {
				w.Write([]byte(fmt.Sprintf("<script>window.location = '%s'</script>", o.Success.String())))
			}
		}()

		//verify the state hasnt been forged or tampered with (Cross-Site Request Forgery (CSRF, XSRF) mitigation)
		if r.URL.Query().Get("state") != o.State {
			redirectErrorHandler(w, r, http.StatusNotFound, fmt.Errorf("Unable to get state (expected \"state\": \"%#v\" got \"%#v\")", o.State, r.URL.Query().Get("state")))
			return
		}
		//exchange for session token
		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"),
			oauth2.SetAuthURLParam("code_verifier", o.CodeVerifier),
		)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}

		// tokenSource := config.TokenSource(ctx, oauth2Token)
		// token, err := tokenSource.Token()

		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "Failed to parse Id token: "+err.Error(), http.StatusInternalServerError)
			log.Error(fmt.Errorf("Failed to parse Id token"))
			return
		}

		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}

		var claims struct {
			Email string `json:"email"`
			Sub   string `json:"sub"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse token claims: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
		}

		tok := &OktaSessionToken{
			SessionToken: oauth2Token,
			IDToken:      rawIDToken,
			ClientID:     config.ClientID,
			OktaServerUS: o.OktaServerUS.String(),
			Stage:        o.Stage,
			Environment:  o.Environment,
		}

		data, err := json.Marshal(tok)
		if err != nil {
			http.Error(w, "Failed to marshal OktaSessionToken: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
		}

		t := &OktaSessionToken{}

		err = json.Unmarshal(data, &t)
		if err != nil {
			http.Error(w, "Failed to unmarshal OktaSessionToken: "+err.Error(), http.StatusInternalServerError)
			log.Error(err)
		}

		log.Infof("Granted Okta session token expiring in: (%s)", oauth2Token.Expiry.Sub(time.Now()).String())

		credChan <- &data
	})
	return nil
}

// startRedirectServer starts up the authentication server
func (o *OktaOIDCClient) startRedirectServer() *http.Server {
	srv := &http.Server{Addr: o.Redirect.Host}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Unable to start OIDC redirect: (%s)", err)
		}
	}()
	return srv
}

// stopRedirectServer shuts down the authentication server
func (o *OktaOIDCClient) stopRedirectServer(srv *http.Server) {
	if err := srv.Close(); err != nil {
		log.Errorf("Unable to terminate OIDC redirect: (%s)", err)
	}
}

func redirectErrorHandler(w http.ResponseWriter, r *http.Request, status int, msg error) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, msg)
	}
}

// restoreWindowFocusAfter restores focus to terminal window from browser login
func restoreWindowFocusAfter(f func() error) error {
	if Platform != "darwin" {
		f()
		return nil
	}

	windowWithFocusRaw, err := exec.Command("osascript", "-e", "tell application \"System Events\""+
		" to return name of first application process whose frontmost is true").Output()
	if eErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("Unable to determine focus: (%s)", string(eErr.Stderr))
	} else if err != nil {
		return fmt.Errorf("Unable to restore focus: (%s)", string(eErr.Stderr))
	}
	windowWithFocus := strings.TrimSpace(string(windowWithFocusRaw))

	// Don't return immedately, even if f returns an error. Try to restore
	// focus first.
	focusErr := f()

	if err != nil {
		// Do not try to regain focus if finding the window to focus failed
		return focusErr
	}

	err = exec.Command("osascript", "-e",
		fmt.Sprintf("tell application \"%s\" to activate", windowWithFocus)).Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("Unable to restore focus: (%s)", string(exitErr.Stderr))
	} else if err != nil {
		return err
	}
	return focusErr
}

// pkce creates the code verifier and challenge
func pkce() (string, string, error) {
	codeVerifier, err := randomHex(30)
	if err != nil {
		return "", "", err
	}

	hash := sha256.New()
	hash.Write([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

	return codeVerifier, codeChallenge, nil
}

// randomHex creates a random hex for code verifier for PKCE
func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// OktaClient interface
type OktaClient interface {
	Authenticate(*Config) ([]byte, error)
}

// OktaProviderOptions data
type OktaProviderOptions struct {
}

// OktaApplyDefaults provides defaults for provider settings
func (o OktaProviderOptions) OktaApplyDefaults() OktaProviderOptions {
	return o
}

// OktaProvider data
type OktaProvider struct {
	Options ProvidorOptions
	OktaProviderOptions
	Keyring  keyring.Keyring
	ClientID string
	Config   *Config
}

// Retrieve fetches the Okta IdP token after Okta browser authentication
func (p *OktaProvider) Retrieve() ([]byte, error) {

	oktaClient, err := p.getOktaClient()
	if err != nil {
		return nil, err
	}

	creds, err := oktaClient.Authenticate(p.Config)
	if err != nil {
		return nil, err
	}

	// session := OktaUserAuthn{SessionToken: creds}

	// bytes, err := json.Marshal(session)
	// if err != nil {
	// 	return "", err
	// }

	if p.Keyring != nil {
		log.Infof("Saving Okta session token to keyring-item: [%s: (%s)]", AppName, OktaSessionName)
		newSessionItem := keyring.Item{
			Key:                         fmt.Sprintf("%s,%s,%s", OktaSessionName, p.Config.Stage, p.Config.Environment),
			Data:                        creds,
			Label:                       fmt.Sprintf("%s: (%s)", AppName, OktaSessionName),
			KeychainNotTrustApplication: false,
		}

		err = p.Keyring.Set(newSessionItem)
		if err != nil {
			return nil, fmt.Errorf("Unable to save keyring item %s: (%s)", newSessionItem.Key, err)
		}
	}
	return creds, err
}

// refresh fetches the Okta IdP token after Okta browser authentication
func RefreshOktaSessionToken(kr keyring.Keyring, item keyring.Item, config *Config) error {

	o := &OktaSessionToken{}

	err := json.Unmarshal(item.Data, &o)
	if err != nil {
		return err
	}

	if time.Since(o.SessionToken.Expiry) > config.OIDC.SessionLaps {
		err := fmt.Errorf("Token experation: (%s) exceededs: (%s) - login required\n",
			time.Since(o.SessionToken.Expiry),
			config.OIDC.SessionLaps,
		)
		return err
	}

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, o.OktaServerUS)
	if err != nil {
		return err
	}

	oauth2Config := &oauth2.Config{
		ClientID:    o.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: config.OIDC.Redirect.String(),
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
			"email",
		},
	}

	ts := oauth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: o.SessionToken.RefreshToken})
	token, err := ts.Token()

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return err
	}

	oidcConfig := &oidc.Config{
		ClientID: o.ClientID,
	}

	verifier := provider.Verifier(oidcConfig)

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return err
	}

	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	tok := &OktaSessionToken{
		SessionToken: token,
		IDToken:      rawIDToken,
		ClientID:     o.ClientID,
		OktaServerUS: o.OktaServerUS,
		Stage:        o.Stage,
		Environment:  o.Environment,
	}

	creds, err := json.Marshal(tok)
	if err != nil {
		return err
	}

	if kr != nil {
		newSessionItem := keyring.Item{
			Key:                         fmt.Sprintf("%s,%s,%s", OktaSessionName, config.Stage, config.Environment),
			Data:                        creds,
			Label:                       fmt.Sprintf("%s: (%s)", AppName, OktaSessionName),
			KeychainNotTrustApplication: false,
		}

		err = DeleteOktaSession(kr)
		if err != nil {
			return fmt.Errorf("Unable to remove keyring item %s: (%s)", newSessionItem.Key, err)
		}
		log.Infof("Writing Okta session token to keyring - %s: (%s)", AppName, OktaSessionName)
		err = kr.Set(newSessionItem)
		if err != nil {
			return fmt.Errorf("Unable to save keyring item %s: (%s)", newSessionItem.Key, err)
		}
	}
	return nil
}

// getOktaClient returns new Okta OIDC Client
func (p *OktaProvider) getOktaClient() (OktaClient, error) {
	return p.NewOktaOIDCClient()
}

// NewLoginProvider creates provider for Okta Login
func NewLoginProvider(k keyring.Keyring, opts OktaProviderOptions, clientOpts ProvidorOptions, config *Config) (*OktaProvider, error) {
	opts = opts.OktaApplyDefaults()
	return &OktaProvider{
		Options:  clientOpts,
		Keyring:  k,
		Config:   config,
		ClientID: config.OIDC.ClientID,
	}, nil
}

// RetrieveOktaToken retrieves Okta IdP Token
func (p *OktaProvider) RetrieveOktaToken() ([]byte, error) {
	provider := OktaProvider{
		Keyring:  p.Keyring,
		Config:   p.Config,
		ClientID: p.Config.OIDC.ClientID,
	}

	creds, err := provider.Retrieve()
	if err != nil {
		return nil, fmt.Errorf("Failed requesting Okta Session token: (%s)", err)
	}

	return creds, nil
}

func DeleteOktaSession(kr keyring.Keyring) error {
	keys, err := kr.Keys()
	if err != nil {
		return fmt.Errorf("Unable to parse keyring keys: (%s)", err)
	}

	for _, k := range keys {
		if strings.Contains(k, OktaSessionName) {
			item := k
			log.Infof("Removing Okta Session token: (%s)", item)
			err := kr.Remove(item)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
