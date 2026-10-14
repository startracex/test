package npmregistry

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

// Document fetches the packument (the document holding every version) of a package.
func (c *Client) Document(spec *specifier.RegistrySpecifier) (*manifest.Document, error) {
	reg, err := c.n.GetRegistry(spec.Namespace())
	if err != nil {
		return nil, fmt.Errorf("resolve registry for %s: %w", spec.Name, err)
	}
	req, err := http.NewRequest(http.MethodGet, reg.JoinPath(spec.Name).String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build document request for %s: %w", spec.Name, err)
	}
	return decodeJSON[*manifest.Document](c.a.Do(req))
}

// Search queries the registry search endpoint.
func (c *Client) Search(p search.Parameters) (*search.Response, error) {
	reg, err := c.n.GetRegistry("")
	if err != nil {
		return nil, fmt.Errorf("resolve registry: %w", err)
	}
	u := reg.JoinPath("/-/v1/search")
	u.RawQuery = p.Query()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}
	return decodeJSON[*search.Response](c.a.Do(req))
}

// Tarball downloads a package tarball from an absolute URL.
func (c *Client) Tarball(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build tarball request for %s: %w", rawURL, err)
	}
	return checkResponse(c.a.Do(req))
}

// Publish uploads one version of a package to the registry.
func (c *Client) Publish(name string, body io.Reader) error {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return fmt.Errorf("resolve registry for %s: %w", name, err)
	}
	req, err := http.NewRequest(http.MethodPut, reg.JoinPath(url.PathEscape(name)).String(), body)
	if err != nil {
		return fmt.Errorf("build publish request for %s: %w", name, err)
	}
	resp, err := checkResponse(c.a.Do(req))
	if err != nil {
		return fmt.Errorf("publish %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// Visibility reports whether a package is public on the registry.
func (c *Client) Visibility(name string) (bool, error) {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return false, fmt.Errorf("resolve registry for %s: %w", name, err)
	}
	req, err := http.NewRequest(http.MethodGet, reg.JoinPath(fmt.Sprintf("/-/package/%s/visibility", url.PathEscape(name))).String(), nil)
	if err != nil {
		return false, fmt.Errorf("build visibility request for %s: %w", name, err)
	}
	result, err := decodeJSON[struct {
		Public bool `json:"public"`
	}](c.a.Do(req))
	if err != nil {
		return false, fmt.Errorf("read visibility of %s: %w", name, err)
	}
	return result.Public, nil
}

// Published reports whether a given version is already on the registry.
func (c *Client) Published(name, version string) (bool, error) {
	reg, err := c.n.GetRegistry(name)
	if err != nil {
		return false, fmt.Errorf("resolve registry for %s: %w", name, err)
	}
	u := reg.JoinPath(name, version)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("build request for %s@%s: %w", name, version, err)
	}
	resp, err := c.a.Do(req)
	if err != nil {
		return false, fmt.Errorf("check whether %s@%s is published: %w", name, version, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

// checkResponse turns a non-2xx response into an error carrying the registry's response body.
func checkResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if detail := strings.TrimSpace(string(body)); detail != "" {
			return nil, fmt.Errorf("unexpected HTTP status %d for %s %s: %s",
				resp.StatusCode, resp.Request.Method, resp.Request.URL, detail)
		}
		return nil, fmt.Errorf("unexpected HTTP status %d for %s %s",
			resp.StatusCode, resp.Request.Method, resp.Request.URL)
	}
	return resp, nil
}

// decodeJSON checks the response and decodes its JSON body into T.
func decodeJSON[T any](resp *http.Response, err error) (T, error) {
	var zero T
	resp, err = checkResponse(resp, err)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()
	var v T
	if err := json.UnmarshalRead(resp.Body, &v); err != nil {
		return zero, fmt.Errorf("decode response from %s: %w", resp.Request.URL, err)
	}
	return v, nil
}
