package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/startracex/wangpkg/pkg/deps"
	"github.com/startracex/wangpkg/pkg/execute"
	"github.com/startracex/wangpkg/pkg/filters"
	"github.com/startracex/wangpkg/pkg/install"
	"github.com/startracex/wangpkg/pkg/lockfile"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/npmconfig"
	"github.com/startracex/wangpkg/pkg/npmregistry"
	"github.com/startracex/wangpkg/pkg/platform"
	"github.com/startracex/wangpkg/pkg/resolve"
	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/startracex/wangpkg/pkg/storage"
	"github.com/startracex/wangpkg/pkg/workspace"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

var InstallCommand = &cli.Command{
	Name:    "install",
	Aliases: []string{"i", "add"},
	Usage:   "Install packages",
	Flags: []cli.Flag{
		flagFilter,
		flagRecursive,
		flagWorkspaceRoot,
		flagDev,
		flagOptional,
		flagPeer,
		flagProd,
		flagOs,
		flagArch,
		flagLibc,
		flagConcurrency,
		flagLocked,
		flagNoPeer,
	},
	Action: func(ctx *cli.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args := ctx.Args().Slice()
		return RunInstall(cwd, InstallOptions{
			Recursive:     ctx.Bool(flagRecursive.Name),
			WorkspaceRoot: ctx.Bool(flagWorkspaceRoot.Name),
			Filters:       ctx.StringSlice(flagFilter.Name),
			Args:          args,
			OS:            ctx.String(flagOs.Name),
			Arch:          ctx.String(flagArch.Name),
			Libc:          ctx.String(flagLibc.Name),
			Concurrency:   ctx.Int(flagConcurrency.Name),
			Locked:        ctx.Bool(flagLocked.Name),
			NoPeer:        ctx.Bool(flagNoPeer.Name),
			Dev:           ctx.Bool(flagDev.Name),
			Optional:      ctx.Bool(flagOptional.Name),
			Peer:          ctx.Bool(flagPeer.Name),
			Prod:          ctx.Bool(flagProd.Name),
		})
	},
}

type InstallOptions struct {
	filters.Options
	Args        []string
	OS          string
	Arch        string
	Libc        string
	Concurrency int
	Locked      bool
	NoPeer      bool
	Dev         bool
	Optional    bool
	Peer        bool
	Prod        bool
}

func RunInstall(cwd string, opts InstallOptions) error {
	ws := workspace.New()
	if err := ws.FindPackages(cwd); err != nil {
		return err
	}

	cfg, err := npmconfig.LoadNpmConfig(npmconfig.LoadOptions{
		Workspace: ws,
		Cwd:       cwd,
		Location:  npmconfig.LocationProject,
	})
	if err != nil {
		return err
	}

	baseLockfile, err := lockfile.Load(ws)
	if err != nil {
		return err
	}

	targetLockfile := lockfile.New(ws)
	if opts.Locked && !baseLockfile.VerifyWorkspace(ws) {
		return fmt.Errorf("lockfile not match to current workspace")
	}

	currentPlatform := platform.New(opts.OS, opts.Arch, opts.Libc)
	for _, p := range ws.Packages() {
		warnings, err := p.Manifest.DevEngines.Check(currentPlatform)
		if err != nil {
			return err
		}
		for _, w := range warnings {
			fmt.Printf("%s: %s\n", p.Rel, w)
		}
	}

	// The packages whose manifests may be modified: in `add` mode only the
	// selected (Nearby) packages, otherwise all workspace packages (no-op).
	pkgs := ws.Packages()
	packageAliases := make(map[*workspace.Package][]string)
	if len(opts.Args) > 0 {
		if opts.Locked {
			return fmt.Errorf("cannot specify packages when using --locked")
		}

		specifiers, err := errSliceMap(opts.Args, specifier.Parse)
		if err != nil {
			return err
		}

		pkgs = filters.Filter(filters.Options{
			Workspace:     ws,
			Filters:       opts.Filters,
			Recursive:     opts.Recursive,
			WorkspaceRoot: opts.WorkspaceRoot,
			Dir:           cwd,
			Nearby:        true,
		})
		mode := deps.NewMode(opts.Dev, opts.Optional, opts.Peer, opts.Prod)
		modeExplicit := opts.Dev || opts.Optional || opts.Peer || opts.Prod
		addedAliases := sliceMap(specifiers, func(s specifier.Specifier) string {
			return s.Alias()
		})
		for _, pkg := range pkgs {
			for _, spec := range specifiers {
				depMode := mode
				if !modeExplicit {
					if existing := pkg.Manifest.DependencyMode(spec.Alias()); existing != 0 {
						depMode = existing
					}
				}
				pkg.SetDependency(spec.Alias(), spec.String(), depMode)
			}
			packageAliases[pkg] = addedAliases
		}
	}

	r := execute.Executor{Workspace: ws, NpmINI: cfg}
	if err := r.RunPre("install", pkgs...); err != nil {
		return err
	}

	reg := npmregistry.New(npmregistry.Options{NpmINI: cfg})
	resolver := resolve.Resolver{
		Registry:       reg,
		DocumentsCache: manifest.NewDocCache(128, 2048),
		Lockfile:       baseLockfile,
		Workspace:      ws,
		Weighted:       semaphore.NewWeighted(int64(max(1, opts.Concurrency))),
		Storage:        storage.New(npmconfig.StoreDir(cfg), reg),
		Locked:         opts.Locked,
	}
	nodes, err := resolver.ResolveWorkspace()
	if err != nil {
		return err
	}
	installer := &install.Installer{
		Storage:        resolver.Storage,
		Lockfile:       targetLockfile,
		Workspace:      ws,
		Weighted:       resolver.Weighted,
		Platform:       currentPlatform,
		Locked:         opts.Locked,
		PackageAliases: packageAliases,
	}

	start := time.Now()
	if _, err := installer.InstallWorkspace(nodes); err != nil {
		return err
	}
	if err := installer.Save(); err != nil {
		return err
	}
	fmt.Printf("Done in %s\n", time.Since(start).Truncate(time.Millisecond))

	if err := r.RunPost("install", pkgs...); err != nil {
		return err
	}

	return nil
}

func errSliceMap[T, R any](s []T, f func(T) (R, error)) ([]R, error) {
	if s == nil {
		return nil, nil
	}
	result := make([]R, 0, len(s))
	for _, s := range s {
		value, err := f(s)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func sliceMap[T, R any](s []T, f func(T) R) []R {
	if s == nil {
		return nil
	}
	result := make([]R, 0, len(s))
	for _, s := range s {
		value := f(s)
		result = append(result, value)
	}
	return result
}
