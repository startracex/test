package npmconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/startracex/wangpkg/pkg/constants"
	"github.com/startracex/wangpkg/pkg/npmini"
	"github.com/startracex/wangpkg/pkg/workspace"
)

const (
	configFileName  = ".npmrc"
	LocationProject = "project"
	LocationUser    = "user"
	LocationGlobal  = "global"
)

var (
	ErrInvalidLocation = errors.New("invalid location")

	internalKeys = []string{"store-dir"}
)

type NpmConfig struct {
	*npmini.NpmINI
	paths []string
}

type LoadOptions struct {
	Workspace  *workspace.Workspace
	Cwd        string
	Location   string
	SkipNoRoot bool
}

// NewNpmConfig merges the given npmrc files in order; later files override
// earlier ones. Missing files are treated as empty.
func NewNpmConfig(paths ...string) (*npmini.NpmINI, error) {
	c := &NpmConfig{paths: paths}
	if err := c.Reload(); err != nil {
		return nil, err
	}
	return c.NpmINI, nil
}

// LoadNpmConfig reads and merges the npmrc files for the given location.
func LoadNpmConfig(opts LoadOptions) (*npmini.NpmINI, error) {
	paths, err := resolvePaths(opts)
	if err != nil {
		return nil, err
	}
	return NewNpmConfig(paths...)
}

// LocationPath returns the file path that applies to the given location and key.
func LocationPath(opts LoadOptions, key string) (string, error) {
	switch opts.Location {
	case LocationProject:
		if isInternalKey(key) {
			return globalPath(), nil
		}
		if err := opts.Workspace.FindRoot(opts.Cwd); err != nil {
			if opts.SkipNoRoot {
				return filepath.Join(opts.Cwd, configFileName), nil
			}
			return "", err
		}
		return filepath.Join(opts.Workspace.Root().Dir, configFileName), nil
	case LocationUser:
		if isInternalKey(key) {
			return globalPath(), nil
		}
		return userPath(), nil
	case LocationGlobal:
		return globalPath(), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidLocation, opts.Location)
	}
}

// LoadNpmConfigFile loads a single npmrc file into a fresh NpmINI
func LoadNpmConfigFile(path string) (*npmini.NpmINI, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return npmini.New(), nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	ini, err := npmini.Load(file)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return ini, nil
}

// Reload re-reads the applied files and re-merges them, replacing the current
// snapshot. The previous snapshot stays in place if reading fails.
func (c *NpmConfig) Reload() error {
	out := npmini.New()
	for _, p := range c.paths {
		f, err := LoadNpmConfigFile(p)
		if err != nil {
			return fmt.Errorf("reload npm config: %w", err)
		}
		out.Merge(f)
	}
	c.NpmINI = out
	return nil
}

func StoreDir(n *npmini.NpmINI) string {
	dir, ok := n.Get("store-dir")
	if !ok {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, constants.UserConfigDirName, "store")
	}
	return dir
}

func isInternalKey(key string) bool {
	return slices.Contains(internalKeys, key)
}

func resolvePaths(opts LoadOptions) ([]string, error) {
	switch opts.Location {
	case LocationProject:
		if err := opts.Workspace.FindRoot(opts.Cwd); err != nil {
			if opts.SkipNoRoot {
				return []string{globalPath(), userPath(), filepath.Join(opts.Cwd, configFileName)}, nil
			}
			if errors.Is(err, workspace.ErrNoWorkspaceRoot) {
				// No workspace root, so there is no project npmrc to merge.
				return []string{globalPath(), userPath()}, nil
			}
			return nil, err
		}
		return []string{globalPath(), userPath(), filepath.Join(opts.Workspace.Root().Dir, configFileName)}, nil
	case LocationUser:
		return []string{globalPath(), userPath()}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidLocation, opts.Location)
	}
}

func globalPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, constants.UserConfigDirName, configFileName)
}

func userPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configFileName)
}
