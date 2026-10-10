package npmregistry

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/npmini"
	"github.com/startracex/wangpkg/pkg/search"
	"github.com/startracex/wangpkg/pkg/specifier"
)

type Options struct {
	NpmINI      *npmini.NpmINI
	HTTPClient  *http.Client
	RetryClient *retryablehttp.Client
}

type Client struct {
	a *authClient
	n *npmini.NpmINI
}

func New(opts Options) *Client {
	return &Client{
		a: newClient(opts),
		n: opts.NpmINI,
	}
}

func (c *Client) NpmINI() *npmini.NpmINI {
	return c.n
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.a.Do(req)
}

func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.a.RoundTrip(req)
}

func (c *Client) Document(spec *specifier.RegistrySpecifier) (*manifest.Document, error) {
	reg, err := c.n.GetRegistry(spec.Namespace())
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, reg.JoinPath(spec.Name).String(), nil)
	if err != nil {
		return nil, err
	}
	return c.decodeJSON[*manifest.Document](c.a.Do(req))
}

func (c *Client) Manifest(spec *specifier.RegistrySpecifier, versionOrTag string) (*manifest.Manifest, error) {
	doc, err := c.Document(spec)
	if err != nil {
		return nil, err
	}
	return doc.Resolve(versionOrTag)
}

func (c *Client) Search(p search.Parameters) (*search.Response, error) {
	reg, err := c.n.GetRegistry("")
	if err != nil {
		return nil, err
	}
	u := reg.JoinPath("/-/v1/search")
	u.RawQuery = p.Query()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return c.decodeJSON[*search.Response](c.a.Do(req))
}

func (c *Client) Tarball(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return c.checkResponse(c.a.Do(req))
}

func (c *Client) Publish(name string, body []byte) error {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, reg.JoinPath(url.PathEscape(name)).String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp, err := c.checkResponse(c.a.Do(req))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

func (c *Client) Visibility(name string) (bool, error) {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequest(http.MethodGet, reg.JoinPath(fmt.Sprintf("/-/package/%s/visibility", url.PathEscape(name))).String(), nil)
	if err != nil {
		return false, err
	}
	result, err := c.decodeJSON[struct {
		Public bool `json:"public"`
	}](c.a.Do(req))
	if err != nil {
		return false, err
	}
	return result.Public, nil
}

func (c *Client) Published(name, version string) (bool, error) {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return false, err
	}
	u := reg.JoinPath(name, version)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}
	resp, err := c.a.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

func (c *Client) checkResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		return nil, fmt.Errorf("HTTP error: %d %s %s", resp.StatusCode, resp.Request.Method, resp.Request.URL.String())
	}
	return resp, nil
}

func (c *Client) decodeJSON[T any](resp *http.Response, err error) (T, error) {
	var zero T
	resp, err = c.checkResponse(resp, err)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()
	var v T
	if err := json.UnmarshalRead(resp.Body, &v); err != nil {
		return zero, err
	}
	return v, nil
}
