package auth

import (
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/startracex/wangpkg/pkg/npmini"
)

func NewTransport(ni *npmini.NpmINI, base http.RoundTripper) http.RoundTripper {
	if base == nil || base == http.DefaultTransport {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConnsPerHost = 0
		base = t
	}

	attempts := retryablehttp.NewClient()
	attempts.Logger = nil
	attempts.HTTPClient = &http.Client{
		Transport: &lazyAuth{NpmINI: ni, RoundTripper: base},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	attempts.ErrorHandler = func(resp *http.Response, err error, _ int) (*http.Response, error) {
		if resp != nil {
			return resp, nil
		}
		return nil, err
	}

	return &retryablehttp.RoundTripper{Client: attempts}
}

func NewClient(ni *npmini.NpmINI, base http.RoundTripper) *http.Client {
	return &http.Client{Transport: NewTransport(ni, base)}
}

type lazyAuth struct {
	*npmini.NpmINI
	http.RoundTripper
}

func (l *lazyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	a := NewAuthorizer(l.NpmINI, l.RoundTripper, req.URL)
	resp, err := a.RoundTrip(req)
	a.Close()
	return resp, err
}
