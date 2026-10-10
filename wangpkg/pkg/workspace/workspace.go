package workspace

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/startracex/wangpkg/pkg/deps"
	"github.com/startracex/wangpkg/pkg/glob"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/osutil"
	"github.com/startracex/wangpkg/pkg/pathutil"
	"github.com/startracex/wangpkg/pkg/specifier"
)

var ErrNoWorkspaceRoot = errors.New("no workspace root")

type Package struct {
	*manifest.Manifest
	Dir    string
	Rel    string
	IsRoot bool
}

type Workspace struct {
	nameMap  map[string]*Package
	pathMap  map[string]*Package
	root     *Package
	packages []*Package
}

func New() *Workspace {
	return &Workspace{
		nameMap: make(map[string]*Package),
		pathMap: make(map[string]*Package),
	}
}

func (w *Workspace) Packages() []*Package {
	return w.packages
}

func (w *Workspace) Root() *Package {
	return w.root
}

func (w *Workspace) PathOf(path string) (*Package, bool) {
	if filepath.IsAbs(path) {
		path = filepath.Clean(path)
		pkg, ok := w.pathMap[path]
		return pkg, ok
	}
	path = filepath.Join(w.Root().Dir, path)
	pkg, ok := w.pathMap[path]
	return pkg, ok
}

func (w *Workspace) RecentPathOf(path string) *Package {
	pkg, ok := w.PathOf(path)
	if ok {
		return pkg
	}
	root := w.Root()
	for dir := range pathutil.UpDir(path, filepath.Dir) {
		if _, err := filepath.Rel(root.Dir, dir); err != nil {
			break
		}
		if pkg, ok := w.PathOf(dir); ok {
			return pkg
		}
	}
	return root
}

func (w *Workspace) NameOf(name string) (*Package, bool) {
	pkg, ok := w.nameMap[name]
	return pkg, ok
}

func (w *Workspace) AddPackage(pkg *Package) {
	if pkg.IsRoot {
		w.root = pkg
	}
	w.packages = append(w.packages, pkg)
	w.pathMap[pkg.Dir] = pkg
	w.nameMap[pkg.Name] = pkg
}

func (w *Workspace) FindRoot(dir string) error {
	if w.root != nil {
		return nil
	}
	dir, err := pathutil.Abs(dir)
	if err != nil {
		return err
	}
	var near *Package
	for dir := range pathutil.UpDir(dir, filepath.Dir) {
		m, err := manifestFromDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		pkg := &Package{
			Manifest: m,
			Dir:      dir,
			Rel:      ".",
			IsRoot:   true,
		}
		if near == nil {
			near = pkg
		}
		if len(m.Workspaces) > 0 {
			nearRel, _ := filepath.Rel(pkg.Dir, near.Dir)
			if glob.MatchPatterns(
				m.Workspaces,
				filepath.ToSlash(nearRel),
				true,
				osutil.CaseInsensitive(),
			) {
				w.AddPackage(pkg)
				return nil
			}
		}
	}
	if near == nil {
		return ErrNoWorkspaceRoot
	}
	w.AddPackage(near)
	return nil
}

func (w *Workspace) FindPackages(dir string) error {
	if err := w.FindRoot(dir); err != nil {
		return err
	}
	patterns := w.Root().Workspaces
	if len(patterns) == 0 {
		return nil
	}
	patterns = append([]string{"!**/node_modules/**", "!**/.git/**"}, patterns...)
	dirs, err := glob.MatchFiles(patterns, glob.Options{
		Dir:             w.Root().Dir,
		GlobStar:        true,
		Absolute:        true,
		DirOnly:         true,
		CaseInsensitive: osutil.CaseInsensitive(),
	})
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := w.AddDir(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (w *Workspace) AddDir(dir string) error {
	m, err := manifestFromDir(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(w.Root().Dir, dir)
	if err != nil {
		return err
	}
	rel = pathutil.PrettyPath(rel)
	w.AddPackage(&Package{
		Manifest: m,
		Dir:      dir,
		Rel:      rel,
	})

	return nil
}

func (w *Workspace) SyncedPackage(pkg *Package) (*Package, error) {
	p := &pkg.PackageJSON
	pp := *p
	pp.Dependencies = maps.Clone(p.Dependencies)
	pp.DevDependencies = maps.Clone(p.DevDependencies)
	pp.PeerDependencies = maps.Clone(p.PeerDependencies)
	pp.OptionalDependencies = maps.Clone(p.OptionalDependencies)
	result := &Package{
		Dir:    pkg.Dir,
		Rel:    pkg.Rel,
		IsRoot: pkg.IsRoot,
		Manifest: &manifest.Manifest{
			FilePath:    pkg.Manifest.FilePath,
			PackageJSON: pp,
			Object:      pkg.Object.Clone(),
		},
	}
	for alias, mode := range pkg.DependenciesMeta(deps.ModeProd | deps.ModeOptional | deps.ModeDev | deps.ModePeer) {
		spec, err := specifier.Parse(fmt.Sprintf("%s@%s", alias, mode.Version))
		if err != nil {
			return nil, err
		}
		switch spec := spec.(type) {
		case *specifier.WorkspaceSpecifier:
			if alias != spec.Alias() {
				return nil, fmt.Errorf("can not convert specifier: different alias %s and %s", alias, spec)
			}
			target, ok := w.NameOf(spec.Name)
			if !ok {
				return nil, fmt.Errorf("unknown workspace %s", spec.Name)
			}
			var version string
			switch spec.Version {
			case "^", "~":
				version = spec.Version + target.Version
			case "*":
				version = target.Version
			default:
				version = spec.Version
			}
			result.SetDependency(alias, version, mode.Mode)
		}
	}
	return result, nil
}

func (w *Workspace) Graph(pkgs []*Package) *deps.Graph[*Package] {
	graph := deps.NewGraph[*Package]()
	for _, pkg := range pkgs {
		graph.Add(pkg)
		for depAlias, depMeta := range pkg.DependenciesMeta(deps.ModeDev | deps.ModeOptional | deps.ModePeer | deps.ModeProd) {
			spec, err := specifier.Parse(fmt.Sprintf("%s@%s", depAlias, depMeta.Version))
			if err != nil {
				continue
			}
			switch v := spec.(type) {
			case *specifier.LinkSpecifier:
				if targPkg, ok := w.PathOf(v.Path); ok {
					graph.AddEdge(targPkg, pkg)
				}
			case *specifier.WorkspaceSpecifier:
				if targPkg, ok := w.NameOf(depAlias); ok {
					graph.AddEdge(targPkg, pkg)
				}
			}
		}
	}
	return graph
}

func manifestFromDir(dir string) (*manifest.Manifest, error) {
	return manifest.ReadFile(filepath.Join(dir, "package.json"))
}
