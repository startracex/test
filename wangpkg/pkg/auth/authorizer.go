package auth

import (
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/startracex/wangpkg/pkg/npmini"
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Authorizer interface {
	http.RoundTripper
	Close()
}

func New(rc *npmini.NpmINI, base http.RoundTripper, urls ...*url.URL) Authorizer {
	if base == nil {
		base = http.DefaultTransport
	}
	if rc != nil {
		for key := range candidatesKeys(urls...) {
			if auth, ok := newKeyed(key, rc, base); ok {
				return auth
			}
		}
	}
	return &noopTransport{base: base}
}

func newKeyed(key string, rc *npmini.NpmINI, base http.RoundTripper) (Authorizer, bool) {
	if token, _ := rc.Get(key + ":_authToken"); token != "" {
		return newHeaderTransport("Bearer "+token, base), true
	}

	if username, _ := rc.Get(key + ":username"); username != "" {
		if password, _ := rc.Get(key + ":_password"); password != "" {
			return newHeaderTransport("Basic "+basicAuth(username, password), base), true
		}
	}

	if auth, ok := rc.Get(key + ":_auth"); ok {
		return newHeaderTransport(auth, base), true
	}

	if certFile, ok := rc.Get(key + ":certfile"); ok {
		if keyFile, ok := rc.Get(key + ":keyfile"); ok {
			return newCertTransport(certFile, keyFile, base), true
		}
	}

	return nil, false
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
