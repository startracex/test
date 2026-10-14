package auth

import (
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/startracex/wangpkg/pkg/npmini"
)

type Authorizer interface {
	http.RoundTripper
	Close()
}

func NewAuthorizer(ni *npmini.NpmINI, base http.RoundTripper, urls ...*url.URL) Authorizer {
	if base == nil {
		base = http.DefaultTransport
	}
	if ni != nil {
		for key := range candidatesKeys(urls...) {
			if auth, ok := newKeyed(key, ni, base); ok {
				return auth
			}
		}
	}
	return &noopTransport{base: base}
}

func newKeyed(key string, ni *npmini.NpmINI, base http.RoundTripper) (Authorizer, bool) {
	if token, _ := ni.Get(key + ":_authToken"); token != "" {
		return newHeaderTransport("Bearer "+token, base), true
	}

	if username, _ := ni.Get(key + ":username"); username != "" {
		if password, _ := ni.Get(key + ":_password"); password != "" {
			return newHeaderTransport("Basic "+basicAuth(username, password), base), true
		}
	}

	if auth, ok := ni.Get(key + ":_auth"); ok {
		return newHeaderTransport(auth, base), true
	}

	if certFile, ok := ni.Get(key + ":certfile"); ok {
		if keyFile, ok := ni.Get(key + ":keyfile"); ok {
			return newCertTransport(certFile, keyFile, base), true
		}
	}

	return nil, false
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
