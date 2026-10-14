package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:      "create",
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
		return RunDlxCreate(cwd, args.Get(0), args.Slice()[1:])
	},
}

func RunDlxCreate(cwd string, pkg string, args []string) error {
	spec, err := specifier.Parse(pkg)
	if err != nil {
		return err
	}
	regSpec, ok := spec.(*specifier.RegistrySpecifier)
	if !ok {
		return fmt.Errorf("only registry specifier is supported")
	}
	if strings.HasPrefix(regSpec.Name, "@") {
		before, after, ok := strings.Cut(regSpec.Name, "/")
		if ok {
			regSpec.Name = before + "/create-" + after
		} else {
			regSpec.Name = before + "/create"
		}
	} else {
		regSpec.Name = "create-" + regSpec.Name
	}
	return runDlx(cwd, regSpec, args)
}
