package manifest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/startracex/wangpkg/pkg/npmini"
	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/startracex/wangpkg/pkg/versions"
)

type Document struct {
	Name     string                  `json:"name"`
	Versions versions.Map[*Manifest] `json:"versions"`
	DistTags map[string]string       `json:"dist-tags"`
	Time     map[string]time.Time    `json:"time"`
}

// Resolve returns the manifest of a version or dist-tag.
func (i *Document) Resolve(versionOrTag string) (*Manifest, error) {
	if versionOrTag == "" {
		versionOrTag = "latest"
	}
	v, m, err := i.resolve(versionOrTag)
	if err != nil {
		return nil, fmt.Errorf("resolve %q of %q: %w", versionOrTag, i.Name, err)
	}
	if m.Version != v {
		return nil, fmt.Errorf("version mismatch in %q: the manifest reports %q but %q was resolved", i.Name, m.Version, v)
	}
	return m, nil
}

func (i *Document) resolve(versionOrTag string) (version string, m *Manifest, err error) {
	if IsDistTag(versionOrTag) {
		if v, ok := i.DistTags[versionOrTag]; ok {
			if m, ok := i.Versions[v]; ok {
				return v, m, nil
			}
			return "", nil, fmt.Errorf("no version matches tag %q in %q package", versionOrTag, i.Name)
		}
	}

	matcher := new(versions.Matcher)
	if err := matcher.Add(versionOrTag); err != nil {
		return "", nil, fmt.Errorf("invalid version or constraint %q in %q package: %w", versionOrTag, i.Name, err)
	}
	semver, err := matcher.Resolve(i.Versions)
	if err != nil {
		return "", nil, fmt.Errorf("cannot resolve version %q in %q package: %w", versionOrTag, i.Name, err)
	}

	v := semver.String()
	if m, ok := i.Versions[v]; ok {
		return v, m, nil
	}
	return "", nil, fmt.Errorf("no version matches tag %q in %q package", versionOrTag, i.Name)
}

// newDocumentRequest builds the request fetching the packument of a package.
func newDocumentRequest(spec *specifier.RegistrySpecifier, rc *npmini.NpmINI) (*http.Request, error) {
	u, err := rc.GetRegistry(spec.Namespace())
	if err != nil {
		return nil, fmt.Errorf("resolve registry for %s: %w", spec.Name, err)
	}
	return http.NewRequest(http.MethodGet, u.JoinPath(spec.Name).String(), nil)
}
