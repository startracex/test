package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/startracex/wangpkg/pkg/constants"
	"github.com/startracex/wangpkg/pkg/deps"
	"github.com/startracex/wangpkg/pkg/execute"
	"github.com/startracex/wangpkg/pkg/install"
	"github.com/startracex/wangpkg/pkg/lockfile"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/npmconfig"
	"github.com/startracex/wangpkg/pkg/npmregistry"
	"github.com/startracex/wangpkg/pkg/orderedobject"
	"github.com/startracex/wangpkg/pkg/resolve"
	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/startracex/wangpkg/pkg/storage"
	"github.com/startracex/wangpkg/pkg/workspace"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

var DlxCommand = &cli.Command{
	Name:      "dlx",
	Usage:     "Download and run a bin defined in a package",
	ArgsUsage: "[args...]",
	Action: func(ctx *cli.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args := ctx.Args()
		if args.Len() == 0 {
			return fmt.Errorf(`no package provided`)
		}
		return RunDlx(cwd, args.First(), args.Slice()[1:])
	},
}

func runDlx(cwd string, regSpec *specifier.RegistrySpecifier, args []string) error {
	_, baseName := regSpec.SplitName()
	cfg, err := npmconfig.LoadNpmConfig(npmconfig.LoadOptions{
		Workspace:  workspace.New(),
		Cwd:        cwd,
		Location:   npmconfig.LocationProject,
		SkipNoRoot: true,
	})
	if err != nil {
		return err
	}
	reg := npmregistry.New(npmregistry.Options{NpmINI: cfg})

	d, err := reg.Document(regSpec)
	if err != nil {
		return err
	}
	m, err := d.Resolve(regSpec.Version)
	if err != nil {
		return err
	}
	if m.Bin()[baseName] == "" {
		return fmt.Errorf("no script in %s", regSpec.String())
	}
	dlxID, err := m.Dist.Integrity.Hex()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dlxDir := filepath.Join(homeDir, constants.UserConfigDirName, "store/dlx")
	hasID, err := cleanDlxDir(dlxDir, dlxID)
	if err != nil {
		return err
	}

	ws := workspace.New()
	ws.AddPackage(&workspace.Package{
		Manifest: &manifest.Manifest{
			Object: orderedobject.New(),
		},
		Dir:    filepath.Join(dlxDir, dlxID),
		Rel:    ".",
		IsRoot: true,
	})
	ws.Root().SetDependency(regSpec.Alias(), regSpec.String(), deps.ModeProd)

	if !hasID {
		concurrency := runtime.NumCPU() * 2
		baseLockfile := lockfile.New(nil)
		targetLockfile := lockfile.New(ws)
		resolver := resolve.Resolver{
			Registry:       reg,
			DocumentsCache: manifest.NewDocCache(128, 2048),
			Lockfile:       baseLockfile,
			Workspace:      ws,
			Weighted:       semaphore.NewWeighted(int64(concurrency)),
			Storage:        storage.New(npmconfig.StoreDir(cfg), reg),
		}

		nodes, err := resolver.ResolveWorkspace()
		if err != nil {
			return err
		}

		installer := install.Installer{
			Lockfile:  targetLockfile,
			Workspace: resolver.Workspace,
			Weighted:  resolver.Weighted,
			Storage:   resolver.Storage,
		}

		if _, err := installer.InstallWorkspace(nodes); err != nil {
			return err
		}
	}

	r := &execute.Executor{
		Workspace: ws,
		NpmINI:    cfg,
	}

	return r.RunCommand(ws.Root(), cwd, append([]string{baseName}, args...)...)
}

func RunDlx(cwd string, pkg string, args []string) error {
	spec, err := specifier.Parse(pkg)
	if err != nil {
		return err
	}
	regSpec, ok := spec.(*specifier.RegistrySpecifier)
	if !ok {
		return fmt.Errorf("only registry specifier is supported")
	}
	return runDlx(cwd, regSpec, args)
}

func cleanDlxDir(dlxDir, dlxID string) (bool, error) {
	hasID := false
	if entries, err := os.ReadDir(dlxDir); err == nil {
		cutoffTime := time.Now().Add(-48 * time.Hour)
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() == dlxID {
				hasID = true
				continue
			}
			info, err := entry.Info()
			if err != nil || info.ModTime().Before(cutoffTime) {
				go func() {
					_ = os.RemoveAll(filepath.Join(dlxDir, entry.Name()))
				}()
			}
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(dlxDir, 0o755); err != nil {
			return false, err
		}
	} else {
		return false, err
	}
	return hasID, nil
}
