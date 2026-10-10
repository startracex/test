package npmregistry

import (
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/startracex/wangpkg/pkg/auth"
	"github.com/startracex/wangpkg/pkg/npmini"
)

type authClient struct {
	n *npmini.NpmINI
	c *http.Client
}

func (c *authClient) Do(req *http.Request) (*http.Response, error) {
	return c.c.Do(req)
}

func (c *authClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.c.Transport.RoundTrip(req)
}

func newClient(opts Options) *authClient {
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	baseRT := opts.HTTPClient.Transport
	if baseRT == nil || baseRT == http.DefaultTransport {
		baseRT = newTransport()
	}

	attempts := opts.RetryClient
	if attempts == nil {
		attempts = retryablehttp.NewClient()
		attempts.Logger = nil
	}

	attempts.HTTPClient = &http.Client{
		Transport: &lazyAuth{NpmINI: opts.NpmINI, RoundTripper: baseRT},
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

	c := &http.Client{
		Transport:     &retryablehttp.RoundTripper{Client: attempts},
		Timeout:       opts.HTTPClient.Timeout,
		Jar:           opts.HTTPClient.Jar,
		CheckRedirect: opts.HTTPClient.CheckRedirect,
	}

	return &authClient{n: opts.NpmINI, c: c}
}

func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = 0
	return t
}

type lazyAuth struct {
	*npmini.NpmINI
	http.RoundTripper
}

func (l *lazyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	a := auth.New(l.NpmINI, l.RoundTripper, req.URL)
	resp, err := a.RoundTrip(req)
	a.Close()
	return resp, err
}
