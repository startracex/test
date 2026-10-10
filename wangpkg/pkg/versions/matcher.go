package versions

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
)

var (
	ErrVersionConflict            = errors.New("version conflict")
	ErrVersionNotMatches          = errors.New("version not matches")
	ErrInvalidVersionOrConstraint = errors.New("invalid version or constraint")
)

type Matcher struct {
	Version     *semver.Version
	Constraints map[string]*semver.Constraints
	mu          sync.RWMutex
}

func (m *Matcher) Add(request string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.add(request)
}

func (m *Matcher) add(request string) error {
	if v, err := semver.StrictNewVersion(request); err == nil {
		if m.Version != nil && !m.Version.Equal(v) {
			return fmt.Errorf("%w: %s and %s", ErrVersionConflict, m.Version, v)
		}
		m.Version = v
		return nil
	}

	if c, err := semver.NewConstraint(request); err == nil {
		key := c.String()
		if m.Constraints == nil {
			m.Constraints = map[string]*semver.Constraints{key: c}
		} else if _, ok := m.Constraints[key]; !ok {
			m.Constraints[key] = c
		}
		return nil
	}

	return fmt.Errorf("%w: %s", ErrInvalidVersionOrConstraint, request)
}

func (m *Matcher) checkConstraints(v *semver.Version) bool {
	for _, c := range m.Constraints {
		if !c.Check(v) {
			return false
		}
	}
	return true
}

func (m *Matcher) Check(v *semver.Version) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Version != nil {
		return m.Version.Equal(v)
	}
	return m.checkConstraints(v)
}

func (m *Matcher) Resolve(v iMap) (*semver.Version, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Version != nil {
		if ok := v.Has(m.Version); !ok {
			return nil, fmt.Errorf("%w: %s", ErrVersionNotMatches, m.Version)
		}
		return m.Version, nil
	}

	var maxVersion *semver.Version
	for v, err := range v.Iterator() {
		if err != nil {
			return nil, err
		}
		if m.checkConstraints(v) {
			if maxVersion == nil || v.GreaterThan(maxVersion) {
				maxVersion = v
			}
		}
	}
	if maxVersion == nil {
		return nil, ErrVersionNotMatches
	}
	return maxVersion, nil
}

func (m *Matcher) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Version != nil {
		return m.Version.String()
	}

	keys := make([]string, 0, len(m.Constraints))
	for key := range m.Constraints {
		keys = append(keys, key)
	}

	return strings.Join(keys, ", ")
}
