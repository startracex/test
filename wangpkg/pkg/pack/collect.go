package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/startracex/wangpkg/pkg/fmtutil"
	"github.com/startracex/wangpkg/pkg/pathutil"
)

type ignoreRule struct {
	matcher *ignore.GitIgnore
	dir     string
}

type CollectOptions struct {
	GlobalIgnore    *ignore.GitIgnore
	IgnoreFileNames []string
	Absolute        bool
	DirOnly         bool
	FileOnly        bool
}

func CollectFiles(rootDir string, opts CollectOptions) ([]string, error) {
	rootDir, err := pathutil.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	var rules []ignoreRule

	if err := filepath.WalkDir(rootDir, func(dir string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if opts.GlobalIgnore != nil && opts.GlobalIgnore.MatchesPath(name) {
			return skip(d)
		}
		if !d.IsDir() {
			return nil
		}

		if isIgnored(dir, rules) {
			return filepath.SkipDir
		}

		var lines []string
		for _, ignoreName := range opts.IgnoreFileNames {
			content, err := os.ReadFile(filepath.Join(dir, ignoreName))
			if err != nil {
				continue
			}
			for _, line := range splitLines(content) {
				lines = append(lines, string(line))
			}
			break
		}
		if len(lines) > 0 {
			matcher := ignore.CompileIgnoreLines(lines...)
			rules = append(rules, ignoreRule{dir: dir, matcher: matcher})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var result []string
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}
		name := d.Name()
		if opts.GlobalIgnore != nil && opts.GlobalIgnore.MatchesPath(name) {
			return skip(d)
		}

		if isIgnored(path, rules) {
			return skip(d)
		}

		if d.IsDir() {
			if opts.FileOnly {
				return nil
			}
			if opts.Absolute {
				result = append(result, path)
			} else {
				rel, _ := filepath.Rel(rootDir, path)
				result = append(result, rel)
			}
			return nil
		}

		if opts.DirOnly {
			return nil
		}
		if opts.Absolute {
			result = append(result, path)
		} else {
			rel, _ := filepath.Rel(rootDir, path)
			result = append(result, rel)
		}
		return nil
	})

	return result, err
}

func isIgnored(path string, rules []ignoreRule) bool {
	for _, rule := range rules {
		if !isDescendant(path, rule.dir) {
			continue
		}
		rel, _ := filepath.Rel(rule.dir, path)
		if rule.matcher.MatchesPath(filepath.ToSlash(rel)) {
			return true
		}
	}
	return false
}

func isDescendant(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

func skip(i interface{ IsDir() bool }) error {
	if i.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

func splitLines(b []byte) [][]byte {
	return bytes.Split(fmtutil.NormalizeToLF(b), []byte{'\n'})
}
