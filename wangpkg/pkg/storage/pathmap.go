package storage

import (
	"path"
	"strings"
)

// PathMap is a map of paths to values
// Zero value is ready to use
type PathMap[T any] struct {
	m        map[string]T
	prefix   string
	hasValue bool
}

func (pm *PathMap[T]) Prefix() string {
	return pm.prefix
}

func (pm *PathMap[T]) Set(name string, val T) {
	if pm.m == nil {
		pm.m = make(map[string]T)
	}

	name = path.Clean(name)
	pm.m[name] = val

	if !pm.hasValue {
		pm.prefix = path.Dir(name)
		pm.hasValue = true
		return
	}

	pm.prefix = shrinkPrefix(pm.prefix, name)
}

func (pm *PathMap[T]) Get(fullName string) (T, bool) {
	v, ok := pm.m[fullName]
	return v, ok
}

func (pm *PathMap[T]) Len() int {
	return len(pm.m)
}

func (pm *PathMap[T]) Map() map[string]T {
	return pm.m
}

func (pm *PathMap[T]) Relative(fullname string) string {
	if pm.prefix == "" {
		return fullname
	}
	return strings.TrimPrefix(fullname, pm.prefix+"/")
}

func shrinkPrefix(prefix, name string) string {
	if prefix == "" {
		return ""
	}

	pParts := strings.Split(prefix, "/")
	nParts := strings.Split(name, "/")

	size := min(len(pParts), len(nParts)-1)

	i := 0
	for i < size && pParts[i] == nParts[i] {
		i++
	}

	if i == 0 {
		return ""
	}
	return path.Join(pParts[:i]...)
}
