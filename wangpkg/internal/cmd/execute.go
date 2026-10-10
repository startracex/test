package cmd

import (
	"fmt"
	"os"

	"github.com/startracex/wangpkg/pkg/execute"
	"github.com/startracex/wangpkg/pkg/npmconfig"
	"github.com/startracex/wangpkg/pkg/workspace"
	"github.com/urfave/cli/v2"
)

var ExecuteCommand = &cli.Command{
	Name:      "execute",
	Aliases:   []string{"exec", "x"},
	Usage:     "Execute a bin file",
	ArgsUsage: "<bin> [args...]",
	Flags: []cli.Flag{
		flagEnv,
	},
	Action: func(ctx *cli.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args := ctx.Args()
		if args.Len() == 0 {
			return fmt.Errorf("no bin name provided")
		}

		return RunExecute(cwd, ExecuteOptions{
			Args: args.Slice(),
		})
	},
}

type ExecuteOptions struct {
	Args  []string
	Shell string
}

func RunExecute(cwd string, opts ExecuteOptions) error {
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
	r := &execute.Executor{
		Workspace: ws,
		NpmINI:    cfg,
	}

	return r.RunCommand(ws.RecentPathOf(cwd), cwd, opts.Args...)
}
