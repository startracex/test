package osutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	junction "github.com/nyaosorg/go-windows-junction"
	"github.com/startracex/wangpkg/pkg/pathutil"
)

func CaseInsensitive() bool {
	return runtime.GOOS == "windows"
}

func RemoveIfExist(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func RemoveDirIfEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		return os.Remove(path)
	}
	return nil
}

func WriteFileIfNotEqual(path string, content []byte, perm os.FileMode) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(content, old) {
		return nil
	}
	return os.WriteFile(path, content, perm)
}

func WriteFileIfNotExist(path string, content []byte, perm os.FileMode) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, content, perm)
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	err = CopyFile(src, dst)
	if err != nil {
		return err
	}
	return os.Remove(src)
}

func Symlink(oldname, newname string) error {
	newAbs, err := pathutil.Abs(newname)
	if err != nil {
		return err
	}
	oldAbs, err := pathutil.Abs(oldname)
	if err != nil {
		return err
	}

	if pathutil.Equals(oldAbs, newAbs) {
		return fmt.Errorf("cannot create symlink from %s to itself", newname)
	}

	if _, err := os.Lstat(newAbs); err == nil {
		if link, readErr := os.Readlink(newAbs); readErr == nil {
			var linkAbs string
			if filepath.IsAbs(link) {
				linkAbs = filepath.Clean(link)
			} else {
				linkAbs = filepath.Join(filepath.Dir(newAbs), link)
			}
			if pathutil.Equals(oldAbs, linkAbs) {
				return nil
			}
			if err := os.Remove(newAbs); err != nil {
				return err
			}
		} else if !os.IsNotExist(readErr) {
			if err := os.Remove(newAbs); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	parentDir := filepath.Dir(newAbs)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}

	return junction.Create(oldAbs, newAbs)
}
