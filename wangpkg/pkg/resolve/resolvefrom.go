package resolve

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/startracex/wangpkg/pkg/lockfile"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/specifier"
)

func (r *Resolver) resolveFromLockfilePackage(lockfilePkg *lockfile.Package) (*manifest.Manifest, error) {
	if r.Locked {
		return manifestFromLockfilePackage(lockfilePkg), nil
	}

	if err := r.Weighted.Acquire(context.Background(), 1); err != nil {
		return nil, err
	}
	defer r.Weighted.Release(1)

	fm, err := r.Storage.LoadFileMap(lockfilePkg.Integrity)
	if err != nil {
		// If package cache is not available, fallback to lockfile metadata
		if lockfilePkg != nil {
			return manifestFromLockfilePackage(lockfilePkg), nil
		}
		return nil, err
	}
	f, ok := fm.Files["package.json"]
	if !ok {
		return nil, fmt.Errorf("file not exist")
	}
	p, err := r.Storage.RealPath(f.Integrity)
	if err != nil {
		return nil, err
	}
	m, err := manifest.ReadFile(p)
	if err != nil {
		return nil, err
	}
	m.Dist = fm.Dist
	return m, nil
}

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

func (r *Resolver) ResolveRegistrySpecifier(requestSpecifier *specifier.RegistrySpecifier) (*manifest.Manifest, *manifest.Document, error) {
	if err := r.Weighted.Acquire(context.Background(), 1); err != nil {
		return nil, nil, err
	}
	defer r.Weighted.Release(1)

	d, err := r.DocumentsCache.FetchDocument(requestSpecifier, r.Registry.NpmINI(), r.Registry)
	if err != nil {
		return nil, nil, err
	}

	resolvedManifest, err := d.Resolve(requestSpecifier.Version)
	if err != nil {
		return nil, d, err
	}
	return resolvedManifest, d, nil
}

func (r *Resolver) resolveFromDir(dir string) (*manifest.Manifest, error) {
	if p, ok := r.Workspace.PathOf(dir); ok {
		return p.Manifest, nil
	}
	return manifest.ReadFile(filepath.Join(dir, "package.json"))
}
