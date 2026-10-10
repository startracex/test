package auth

import "net/http"

type noopTransport struct {
	base http.RoundTripper
}

func (t *noopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req)
}

func (a *noopTransport) Close() {}
