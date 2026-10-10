package storage

import (
	"encoding/json/v2"
	"os"

	"github.com/startracex/wangpkg/pkg/manifest"
)

type FileMap struct {
	Files     map[string]File `json:"files"`
	Dist      manifest.Dist   `json:"dist"`
	Prefix    string          `json:"prefix"`
	FileCount int             `json:"fileCount"`
}

type File struct {
	Integrity manifest.Integrity `json:"integrity"`
	Mode      int64              `json:"mode"`
	Size      int64              `json:"size"`
}

func newFileMap(pm *PathMap[File], d manifest.Dist) *FileMap {
	i := &FileMap{Dist: d}
	i.FileCount = pm.Len()
	out := make(map[string]File, len(pm.Map()))
	for fullname, v := range pm.Map() {
		out[pm.Relative(fullname)] = v
	}
	i.Files = out
	i.Prefix = pm.Prefix()
	return i
}

func writeFileMap(filePath string, fm *FileMap) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if err := json.MarshalWrite(file, fm); err != nil {
		return err
	}
	return nil
}

func readFileMap(filePath string) (*FileMap, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	fm := new(FileMap)
	if err := json.UnmarshalRead(file, fm); err != nil {
		return nil, err
	}

	return fm, nil
}
