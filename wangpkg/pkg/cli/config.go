package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/startracex/wangpkg/pkg/npmconfig"
	"github.com/startracex/wangpkg/pkg/workspace"
	"github.com/urfave/cli/v2"
)

var ConfigCommand = &cli.Command{
	Name:      "config",
	Usage:     "Manage configuration",
	ArgsUsage: "config <command>",
	Subcommands: []*cli.Command{
		{
			Name:  "list",
			Usage: "List config",
			Flags: []cli.Flag{
				flagLocationFlag_Project,
			},
			Action: func(ctx *cli.Context) error {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				return RunConfigList(cwd, ConfigOption{
					Location: ctx.String(flagLocationFlag_Project.Name),
				})
			},
		},
		{
			Name:      "get",
			Usage:     "Get a config value",
			ArgsUsage: "<key>",
			Flags: []cli.Flag{
				flagLocationFlag_Project,
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				if args.Len() != 1 {
					return fmt.Errorf("usage: config get <key>")
				}
				return RunConfigGet(cwd, ConfigOption{
					Location: ctx.String(flagLocationFlag_Project.Name),
					Key:      args.Get(0),
				})
			},
		},
		{
			Name:      "set",
			Usage:     "Set a config value",
			ArgsUsage: "<key> <value>",
			Flags: []cli.Flag{
				flagLocationFlag_User,
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				if args.Len() != 2 {
					return fmt.Errorf("usage: config set <key> <value>")
				}
				return RunConfigSet(cwd, ConfigOption{
					Key:      args.Get(0),
					Value:    args.Get(1),
					Location: ctx.String(flagLocationFlag_User.Name),
				})
			},
		},
		{
			Name:      "delete",
			Usage:     "Delete a config key",
			ArgsUsage: "<key>",
			Flags: []cli.Flag{
				flagLocationFlag_User,
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				if args.Len() != 1 {
					return fmt.Errorf("usage: config delete <key>")
				}
				return RunConfigDelete(cwd, ConfigOption{
					Key:      args.Get(0),
					Location: ctx.String(flagLocationFlag_User.Name),
				})
			},
		},
	},
}

type ConfigOption struct {
	Location string
	Key      string
	Value    string
}

func RunConfigList(cwd string, opts ConfigOption) error {
	cfg, err := npmconfig.LoadNpmConfig(npmconfig.LoadOptions{
		Workspace: workspace.New(),
		Cwd:       cwd,
		Location:  opts.Location,
	})
	if err != nil {
		return err
	}
	fmt.Print(cfg.String())
	return nil
}

func RunConfigGet(cwd string, opts ConfigOption) error {
	cfg, err := npmconfig.LoadNpmConfig(npmconfig.LoadOptions{
		Workspace: workspace.New(),
		Cwd:       cwd,
		Location:  opts.Location,
	})
	if err != nil {
		return err
	}
	if value, ok := cfg.Get(opts.Key); ok {
		fmt.Println(value)
	}
	return nil
}

func RunConfigDelete(cwd string, opts ConfigOption) error {
	path, err := npmconfig.LocationPath(npmconfig.LoadOptions{
		Workspace: workspace.New(),
		Cwd:       cwd,
		Location:  opts.Location,
	}, opts.Key)
	if err != nil {
		return err
	}
	ni, err := npmconfig.LoadNpmConfigFile(path)
	if err != nil {
		return err
	}
	if !ni.Delete(opts.Key) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, ni.Bytes(), 0o644)
}

func RunConfigSet(cwd string, opts ConfigOption) error {
	key := opts.Key
	value := opts.Value
	path, err := npmconfig.LocationPath(npmconfig.LoadOptions{
		Workspace: workspace.New(),
		Cwd:       cwd,
		Location:  opts.Location,
	}, opts.Key)
	if err != nil {
		return err
	}
	ni, err := npmconfig.LoadNpmConfigFile(path)
	if err != nil {
		return err
	}
	if !ni.Set(key, value) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, ni.Bytes(), 0o644)
}
