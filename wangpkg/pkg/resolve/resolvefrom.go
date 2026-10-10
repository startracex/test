package resolve

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/startracex/wangpkg/pkg/lockfile"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/specifier"
)

// resolveFromLockfilePackage restores the manifest of an already installed
// package from the store, falling back to the lockfile metadata when the cache
// is unavailable.
func (r *Resolver) resolveFromLockfilePackage(lockfilePkg *lockfile.Package) (*manifest.Manifest, error) {
	if r.Locked {
		return manifestFromLockfilePackage(lockfilePkg), nil
	}

	if err := r.Weighted.Acquire(context.Background(), 1); err != nil {
		return nil, fmt.Errorf("acquire resolve slot: %w", err)
	}
	defer r.Weighted.Release(1)

	fm, err := r.Storage.LoadFileMap(lockfilePkg.Integrity)
	if err != nil {
		// If package cache is not available, fallback to lockfile metadata
		if lockfilePkg != nil {
			return manifestFromLockfilePackage(lockfilePkg), nil
		}
		return nil, fmt.Errorf("load file map of %s: %w", lockfilePkg.Integrity, err)
	}
	f, ok := fm.Files["package.json"]
	if !ok {
		return nil, fmt.Errorf("package.json missing from the file map of %s", lockfilePkg.Integrity)
	}
	p, err := r.Storage.RealPath(f.Integrity)
	if err != nil {
		return nil, fmt.Errorf("resolve stored path of package.json: %w", err)
	}
	m, err := manifest.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", p, err)
	}
	m.Dist = fm.Dist
	return m, nil
}

// manifestFromLockfilePackage rebuilds a minimal manifest from lockfile data.
func manifestFromLockfilePackage(pkg *lockfile.Package) *manifest.Manifest {
	m := &manifest.Manifest{
		BinRaw:  pkg.Bin,
		Engines: pkg.Engines,
		OS:      pkg.OS,
		CPU:     pkg.CPU,
		Libc:    pkg.Libc,
		Dist:    manifest.Dist{Integrity: pkg.Integrity},
	}
	return m
}

// ResolveRegistrySpecifier fetches the packument of a registry package and
// resolves the requested version from it.
func (r *Resolver) ResolveRegistrySpecifier(requestSpecifier *specifier.RegistrySpecifier) (*manifest.Manifest, *manifest.Document, error) {
	if err := r.Weighted.Acquire(context.Background(), 1); err != nil {
		return nil, nil, fmt.Errorf("acquire resolve slot: %w", err)
	}
	defer r.Weighted.Release(1)

	d, err := r.DocumentsCache.FetchDocument(requestSpecifier, r.Registry.NpmINI(), r.Registry)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch document of %s: %w", requestSpecifier.Name, err)
	}

	resolvedManifest, err := d.Resolve(requestSpecifier.Version)
	if err != nil {
		return nil, d, fmt.Errorf("resolve %s: %w", requestSpecifier, err)
	}
	return resolvedManifest, d, nil
}

// resolveFromDir reads the manifest of a directory, preferring a workspace
// package when one is registered there.
func (r *Resolver) resolveFromDir(dir string) (*manifest.Manifest, error) {
	if p, ok := r.Workspace.PathOf(dir); ok {
		return p.Manifest, nil
	}
	return manifest.ReadFile(filepath.Join(dir, "package.json"))
}
