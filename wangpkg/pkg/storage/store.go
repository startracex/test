package storage

import (
	"archive/tar"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/startracex/wangpkg/pkg/concurrent"
	"github.com/startracex/wangpkg/pkg/manifest"
	"github.com/startracex/wangpkg/pkg/npmregistry"
)

type Storage struct {
	results  concurrent.SharedResult[*FileMap]
	Registry *npmregistry.Client
	Dir      string
}

func New(dir string, registry *npmregistry.Client) *Storage {
	return &Storage{Dir: dir, Registry: registry}
}

func (s *Storage) getIndexPath(hash string) string {
	return filepath.Join(s.Dir, "index", chunkPath(hash, []int{2, 2}))
}

func (s *Storage) getChunkPath(hash string) string {
	return filepath.Join(s.Dir, "chunk", chunkPath(hash, []int{2, 2}))
}

func (s *Storage) getTempPath() string {
	return filepath.Join(s.Dir, "tmp")
}

func (s *Storage) FetchArtifact(d manifest.Dist) (*FileMap, error) {
	return s.results.Do(d.Integrity.String(), func() (*FileMap, error) {
		ihex, err := d.Integrity.Hex()
		if err != nil {
			return nil, err
		}
		indexFilePath := s.getIndexPath(ihex)
		fm, err := readFileMap(indexFilePath)
		if err == nil {
			return fm, nil
		}

		resp, err := s.Registry.Tarball(d.Tarball)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		tmpDir := s.getTempPath()
		if err := os.MkdirAll(tmpDir, 0o755); err != nil {
			return nil, err
		}

		pm := new(PathMap[File])
		if err := verifiedReadTar(d.Integrity, resp.Body, func(tarHeader *tar.Header, tarReader *tar.Reader) error {
			name := tarHeader.Name
			if tarHeader.Typeflag != tar.TypeReg {
				return skip
			}
			tmpFile, err := os.CreateTemp(tmpDir, "tmp-*")
			if err != nil {
				return err
			}

			tmpName := tmpFile.Name()
			renameDone := false
			closetmpDone := false
			defer func() {
				if !closetmpDone {
					_ = tmpFile.Close()
				}
				if !renameDone {
					_ = os.Remove(tmpName)
				}
			}()

			fh := sha512.New()
			if _, err := io.Copy(io.MultiWriter(tmpFile, fh), tarReader); err != nil {
				return err
			}
			hashsum := fh.Sum(nil)
			hashsumhex := hex.EncodeToString(hashsum)
			hashbase64 := base64.StdEncoding.EncodeToString(hashsum)
			pm.Set(name, File{
				Size:      tarHeader.Size,
				Mode:      tarHeader.Mode,
				Integrity: manifest.Integrity("sha512-" + hashbase64),
			})

			if err := syncAndClose(tmpFile); err != nil {
				return err
			}
			closetmpDone = true

			chunkFilePath := s.getChunkPath(hashsumhex)
			if err := os.MkdirAll(filepath.Dir(chunkFilePath), 0o755); err != nil {
				return err
			}
			if err := os.Rename(tmpName, chunkFilePath); err != nil {
				if !errors.Is(err, os.ErrExist) {
					// may other go routine already wrote the file
					return err
				}
			}
			renameDone = true

			return nil
		}); err != nil {
			return nil, err
		}

		fm = newFileMap(pm, d)
		if err := os.MkdirAll(filepath.Dir(indexFilePath), 0o755); err != nil {
			return nil, err
		}
		if err := writeFileMap(indexFilePath, fm); err != nil {
			return nil, err
		}
		return fm, nil
	})
}

func (s *Storage) RealPath(i manifest.Integrity) (string, error) {
	ihex, err := i.Hex()
	if err != nil {
		return "", err
	}
	return s.getChunkPath(ihex), nil
}

func (s *Storage) LoadFileMap(i manifest.Integrity) (*FileMap, error) {
	ihex, err := i.Hex()
	if err != nil {
		return nil, err
	}
	filePath := s.getIndexPath(ihex)
	return readFileMap(filePath)
}
