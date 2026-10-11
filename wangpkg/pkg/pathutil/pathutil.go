package pathutil

import (
	"iter"
	"path/filepath"
	"runtime"
	"strings"
)

func TrimExt(path string) string {
	ext := filepath.Ext(path)
	if base := filepath.Base(path); len(base) > 1 && ext == base {
		return path
	}
	return strings.TrimSuffix(path, ext)
}

func IsPathLike(s string) bool {
	if s == "" {
		return false
	}

	dot := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' || c == '\\' {
			return dot <= 2
		}
		if c == '.' && dot < 2 {
			dot++
			continue
		}
		return false
	}
	return dot == 1 || dot == 2 || filepath.IsAbs(s)
}

func PrettyPath(s string) string {
	s = filepath.Clean(s)
	s = strings.ReplaceAll(s, "\\", "/")
	if s[0] == '/' || s == "." || strings.HasPrefix(s, "..") || filepath.IsAbs(s) {
		return s
	}
	return "./" + s
}

func Abs(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Abs(path)
}

func Equals(path1, path2 string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(path1, path2)
	}
	return path1 == path2
}

func UpDir(root string, dirFunc func(string) string) iter.Seq[string] {
	return func(yield func(string) bool) {
		currentDir := root

		for {
			if !yield(currentDir) {
				return
			}

			parentDir := dirFunc(currentDir)
			if parentDir == currentDir {
				break
			}
			currentDir = parentDir
		}
	}
}
