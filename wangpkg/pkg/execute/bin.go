package execute

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/startracex/wangpkg/pkg/constants"
	"github.com/startracex/wangpkg/pkg/osutil"
)

func getBinPath(s string) string {
	return filepath.Join(s, constants.ModulesDirName, constants.BinDirName)
}

func WriteBin(pkgDir, distPath string, binMap map[string]string) error {
	binDir := getBinPath(pkgDir)
	if err := os.MkdirAll(binDir, 0o777); err != nil {
		return err
	}
	for binName, binPath := range binMap {
		executePath, err := filepath.Rel(binDir, filepath.Join(distPath, binPath))
		if err != nil {
			return err
		}

		writePath := filepath.Join(binDir, binName)
		if err := writeScripts(writePath, executePath); err != nil {
			return err
		}
	}
	return nil
}

func RemoveBin(pkgDir, binName string) error {
	binDir := getBinPath(pkgDir)
	execPath := filepath.Join(binDir, binName)
	if err := removeScripts(execPath); err != nil {
		return err
	}
	return nil
}

func removeScripts(name string) error {
	for _, name := range []string{name, name + ".ps1", name + ".cmd"} {
		if err := osutil.RemoveIfExist(name); err != nil {
			return err
		}
	}
	return nil
}

func writeScripts(name, exec string) error {
	exec = filepath.ToSlash(exec)
	for name, data := range map[string][]byte{
		name:          formatSH(exec),
		name + ".ps1": formatPWSH(exec),
		name + ".cmd": formatCMD(exec),
	} {
		if err := os.WriteFile(name, data, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func formatCMD(execraw string) []byte {
	exec := strings.ReplaceAll(execraw, "/", "\\")
	buf := new(bytes.Buffer)

	buf.WriteString(`@IF EXIST "%~dp0\node.exe" (
  "%~dp0\node.exe"  "%~dp0\` + exec + `" %*
) ELSE (`)

	if ext := strings.ToUpper(filepath.Ext(execraw)); ext != "" {
		buf.WriteString(`
  @SET PATHEXT=%PATHEXT:;` + ext + `;=;%`)
	}

	buf.WriteString(`
  node  "%~dp0\` + exec + `" %*
)
`)
	return buf.Bytes()
}

func formatPWSH(exec string) []byte {
	return fmt.Appendf(nil, `#!/usr/bin/env pwsh
$basedir=Split-Path $MyInvocation.MyCommand.Definition -Parent
if ($MyInvocation.ExpectingInput) {
	$input | & node  "$basedir/%s" $args
} else {
	& node "$basedir/%s" $args
}
exit $LASTEXITCODE
`, exec, exec)
}

func formatSH(exec string) []byte {
	return fmt.Appendf(nil, `#!/bin/sh
basedir=$(dirname "$(echo "$0" | sed -e 's,\\,/,g')")
"case `+"`uname`"+` in
    *CYGWIN*|*MINGW*|*MSYS*)
        if command -v cygpath > /dev/null 2>&1; then
			basedir=`+"`cygpath -w \"$basedir\"`"+`
        fi
    ;;
esac

exec node  "$basedir/%s" "$@"
`, exec)
}
