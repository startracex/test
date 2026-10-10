package storage

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/startracex/wangpkg/pkg/manifest"
)

func newHash(a string) (hash.Hash, error) {
	switch a {
	case "sha512":
		return sha512.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha384":
		return sha512.New384(), nil
	}
	return nil, fmt.Errorf("algorithm not supported")
}

func syncAndClose(f *os.File) error {
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}

func chunkSlice(input string, segments []int) []string {
	if len(segments) == 0 {
		return []string{input}
	}
	result := make([]string, 0, len(segments))
	runes := []rune(input)
	pos := 0
	for _, segment := range segments {
		if segment < 0 {
			panic("negative segments")
		}

		end := min(pos+segment, len(runes))
		result = append(result, string(runes[pos:end]))
		pos = end
		if pos >= len(runes) {
			break
		}
	}
	if pos < len(runes) {
		result = append(result, string(runes[pos:]))
	}
	return result
}

func chunkPath(hash string, segments []int) string {
	return strings.Join(chunkSlice(hash, segments), string(filepath.Separator))
}

var skip = errors.New("skip")

func readTar(r io.Reader, fn func(*tar.Header, *tar.Reader) error) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if err := fn(header, tr); err != nil {
			if errors.Is(err, skip) {
				// copy to discard to consume tar reader
				if _, copyErr := io.Copy(io.Discard, tr); copyErr != nil {
					return copyErr
				}
				continue
			}
			return err
		}
	}
	return nil
}

func verifiedReadTar(i manifest.Integrity, r io.Reader, fn func(tarHeader *tar.Header, tarReader *tar.Reader) error) error {
	h, err := newHash(i.Algorithm())
	if err != nil {
		return err
	}
	tee := io.TeeReader(r, h)
	if err := readTar(tee, fn); err != nil {
		return err
	}
	actual := base64.StdEncoding.EncodeToString(h.Sum(nil))
	expect := i.Hash()
	if actual != expect {
		return fmt.Errorf("integrity mismatch: the actual hash %q does not match the expected hash %q", actual, expect)
	}
	return nil
}
