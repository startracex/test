package versions

import (
	"iter"

	"github.com/Masterminds/semver/v3"
)

type iMap interface {
	Has(*semver.Version) bool
	Iterator() iter.Seq2[*semver.Version, error]
}

type Map[T any] map[string]T

func (m Map[T]) Has(v *semver.Version) bool {
	_, ok := m[v.String()]
	return ok
}

func (m Map[T]) Iterator() iter.Seq2[*semver.Version, error] {
	return func(yield func(*semver.Version, error) bool) {
		for k := range m {
			sv, err := semver.StrictNewVersion(k)
			if !yield(sv, err) {
				return
			}
		}
	}
}
