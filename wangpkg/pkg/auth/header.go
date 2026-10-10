package auth

import "net/http"

type headerTransport struct {
	base          http.RoundTripper
	authorization string
}

func newHeaderTransport(authorization string, base http.RoundTripper) *headerTransport {
	return &headerTransport{
		authorization: authorization,
		base:          base,
	}
}

func (a *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", a.authorization)
	return a.base.RoundTrip(req)
}
func (a *headerTransport) Close() {}
