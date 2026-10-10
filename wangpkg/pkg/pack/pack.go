package pack

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/startracex/wangpkg/pkg/fmtutil"
	"github.com/startracex/wangpkg/pkg/glob"
	"github.com/startracex/wangpkg/pkg/osutil"
	"github.com/startracex/wangpkg/pkg/pathutil"
	"github.com/startracex/wangpkg/pkg/specifier"
	"github.com/startracex/wangpkg/pkg/workspace"
)

var AlwaysInclude = []string{
	"README",
	"LICENSE",
	"LICENCE",
}

var AlwaysExclude = []string{
	`.git`,
	`.npmrc`,
	`node_modules`,
	`package-lock.json`,
	`pnpm-lock.yaml`,
	`yarn.lock`,
	`bun.lock`,
	`deno.lock`,
	`wangpkg.lock`,
}

var UsesIgnoreFileNames = []string{
	".npmignore",
	".gitignore",
}

var DefaultIgnore = []string{
	`*.orig`,
	`.*.swp`,
	`.DS_Store`,
	`._*`,
	`.git`,
	`.hg`,
	`.lock-wscript`,
	`.npmrc`,
	`.svn`,
	`.wafpickle-N`,
	`CVS`,
	`config.gypi`,
	`node_modules`,
	`npm-debug.log`,
	`package-lock.json`,
	`pnpm-lock.yaml`,
	`yarn.lock`,
	`bun.lock`,
	`deno.lock`,
	`wangpkg.lock`,
}

func containsFold(s []string, t string) bool {
	for _, v := range s {
		if strings.EqualFold(v, t) {
			return true
		}
	}
	return false
}

func FindSimilarFiles(dir string, names []string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	found := ""
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		if containsFold(names, fileName) {
			return fileName, nil
		}
		if found == "" && containsFold(names, pathutil.TrimExt(fileName)) {
			found = fileName
		}
	}
	return found, nil
}

func PackName(name string) string {
	reg := &specifier.RegistrySpecifier{Name: name}
	if scope, base := reg.SplitName(); scope != "" {
		name = scope + "-" + base
	}
	return name
}

type Options struct {
	Package *workspace.Package
	Writer  *tar.Writer
}

type Init struct {
	*workspace.Workspace
	once        sync.Once
	LicensePath string
	Prefix      string
}

func (i *Init) Create(pkg *workspace.Package, writer *tar.Writer) (err error) {
	prefix := i.Prefix
	paths, err := IncludeFileMap(pkg)
	hasLicense := false
	for rel, abs := range paths {
		name := filepath.ToSlash(filepath.Join(prefix, rel))
		if abs == pkg.FilePath {
			if err := syncAndWrite(pkg, abs, name, writer); err != nil {
				return err
			}
			continue
		}

		if err := copyAndWrite(abs, name, writer); err != nil {
			return err
		}
		hasLicense = hasLicense || containsFold(AlwaysInclude[1:], pathutil.TrimExt(rel))
	}
	if !hasLicense {
		if filePath := i.LicensePath; filePath != "" {
			fileName := filepath.Base(filePath)
			name := filepath.ToSlash(filepath.Join(prefix, fileName))
			err = copyAndWrite(filePath, name, writer)
		}
	}
	return err
}

func IncludeFileMap(pkg *workspace.Package) (map[string]string, error) {
	mustIncludes, err := MustIncludeFiles(pkg)
	if err != nil {
		return nil, err
	}
	var includes []string
	if len(pkg.Files) > 0 {
		includes, err = PatternsIncludeFiles(pkg.Dir, pkg.Files)
	} else {
		includes, err = CollectFiles(pkg.Dir, CollectOptions{
			FileOnly:        true,
			Absolute:        true,
			IgnoreFileNames: UsesIgnoreFileNames,
			GlobalIgnore:    ignore.CompileIgnoreLines(append(UsesIgnoreFileNames, DefaultIgnore...)...),
		})
	}
	if err != nil {
		return nil, err
	}
	paths := append(mustIncludes, includes...)
	result := make(map[string]string, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(pkg.Dir, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		result[rel] = path
	}
	return result, nil
}

func MustIncludeFiles(pkg *workspace.Package) ([]string, error) {
	result := []string{pkg.FilePath}
	files, err := os.ReadDir(pkg.Dir)
	add := func(name string) {
		result = append(result, filepath.Join(pkg.Dir, name))
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		if containsFold(AlwaysInclude, pathutil.TrimExt(fileName)) {
			add(fileName)
		}
	}
	for _, binFile := range pkg.Bin() {
		add(binFile)
	}
	if main := string(pkg.Main); main != "" {
		add(main)
	}
	return result, err
}

func PatternsIncludeFiles(dir string, files []string) ([]string, error) {
	patterns := make([]string, 0, len(UsesIgnoreFileNames)+len(AlwaysExclude)+len(files))
	for _, s := range UsesIgnoreFileNames {
		patterns = append(patterns, "!**/"+s)
	}
	for _, s := range AlwaysExclude {
		patterns = append(patterns, "!**/"+s)
	}
	patterns = append(patterns, files...)
	return glob.MatchFiles(patterns, glob.Options{
		Dir:             dir,
		GlobStar:        true,
		FileOnly:        true,
		Absolute:        true,
		ExpandDirs:      true,
		CaseInsensitive: osutil.CaseInsensitive(),
	})
}

func syncAndWrite(pkg *workspace.Package, path, name string, writer *tar.Writer) error {
	b, err := fmtutil.FormatJSON(path, pkg.Manifest)
	if err != nil {
		return err
	}
	i, err := os.Stat(pkg.FilePath)
	if err != nil {
		return err
	}
	size := int64(len(b))
	fi := info{
		FileInfo: i,
		name:     &name,
		size:     &size,
	}
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	if err := writer.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = writer.Write(b)
	return err
}

func copyAndWrite(path, name string, writer *tar.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name

	if err := writer.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err = io.Copy(writer, file); err != nil {
		return err
	}
	return err
}
