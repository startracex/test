package cli

import (
	"runtime"

	"github.com/startracex/wangpkg/pkg/npmconfig"
	"github.com/startracex/wangpkg/pkg/platform"
	"github.com/urfave/cli/v2"
)

var (
	flagEnv = &cli.StringSliceFlag{
		Name:  "env",
		Usage: "environment variables.",
	}
	flagRecursive = &cli.BoolFlag{
		Name:  "recursive",
		Usage: "recursive filter workspace packages.",
	}
	flagWorkspaceRoot = &cli.BoolFlag{
		Name:  "workspace-root",
		Usage: "workspace root directory.",
	}
	flagFilter = &cli.StringSliceFlag{
		Name:  "filter",
		Usage: "filter the packages in the workspace.",
	}
	flagOutput = &cli.StringFlag{
		Name:  "output",
		Usage: "output directory",
	}
	flagDev = &cli.BoolFlag{
		Name:    "dev",
		Aliases: []string{"D"},
		Usage:   "save package as devDependencies.",
	}
	flagOptional = &cli.BoolFlag{
		Name:    "optional",
		Aliases: []string{"O"},
		Usage:   "save package as optionalDependencies.",
	}
	flagPeer = &cli.BoolFlag{
		Name:  "peer",
		Usage: "save package as peerDependencies.",
	}
	flagProd = &cli.BoolFlag{
		Name:    "prod",
		Aliases: []string{"P"},
		Usage:   "save package as dependency.",
	}
	flagOs = &cli.StringFlag{
		Name:    "os",
		Usage:   "os to install package.",
		Aliases: []string{"platform"},
		Value:   platform.Platform{}.V8OS(),
	}
	flagArch = &cli.StringFlag{
		Name:    "arch",
		Usage:   "arch to install package.",
		Aliases: []string{"cpu"},
		Value:   platform.Platform{}.V8Arch(),
	}
	flagLibc = &cli.StringFlag{
		Name:  "libc",
		Usage: "libc to install package.",
		Value: platform.DetectLibc(),
	}
	flagConcurrency = &cli.IntFlag{
		Name:  "concurrency",
		Usage: "max concurrency to install package.",
		Value: runtime.NumCPU() * 2,
	}
	flagNoPeer = &cli.BoolFlag{
		Name:  "no-peer",
		Usage: "no peer dependencies.",
	}
	flagLocked = &cli.BoolFlag{
		Name:  "locked",
		Usage: "ensure lockfile is locked",
	}
	flagSize = &cli.IntFlag{
		Name:  "size",
		Usage: "size to search packages.",
		Value: 10,
	}
	flagLocationFlag_Project = &cli.StringFlag{
		Name:  "location",
		Usage: "location of the npmrc file.",
		Value: npmconfig.LocationProject,
	}
	flagLocationFlag_User = &cli.StringFlag{
		Name:  "location",
		Usage: "location of the npmrc file.",
		Value: npmconfig.LocationUser,
	}
)
