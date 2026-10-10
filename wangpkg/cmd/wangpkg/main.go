package main

import (
	"fmt"
	"os"

	"github.com/startracex/wangpkg/internal/cmd"
	"github.com/startracex/wangpkg/pkg/constants"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    constants.AppName,
		Version: constants.Version,
		Commands: []*cli.Command{
			cmd.InitCommand,
			cmd.ListCommand,
			cmd.ConfigCommand,
			cmd.PackCommand,
			cmd.DlxCommand,
			cmd.CreateCommand,
			cmd.RunCommand,
			cmd.ExecuteCommand,
			cmd.RemoveCommand,
			cmd.InstallCommand,
			cmd.SearchCommand,
			cmd.InfoCommand,
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
