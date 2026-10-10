package manifest

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"sync"

	"github.com/startracex/wangpkg/pkg/orderedobject"
)

// Manifest is a package.json: PackageJSON is its typed view, Object keeps the
// original key order so the file can be written back unchanged.
type Manifest struct {
	PackageJSON
	Object   *orderedobject.Object
	mu       sync.RWMutex
	FilePath string
}

// UnmarshalJSONFrom decodes a package.json into both representations.
func (m *Manifest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	value, err := dec.ReadValue()
	if err != nil {
		return fmt.Errorf("read manifest value: %w", err)
	}

	o := orderedobject.New()
	if err := json.Unmarshal(value, o); err != nil {
		return fmt.Errorf("decode ordered object: %w", err)
	}

	p := new(PackageJSON)
	if err := json.Unmarshal(value, p); err != nil {
		return fmt.Errorf("decode package.json: %w", err)
	}
	m.PackageJSON = *p
	m.Object = o
	return nil
}

// MarshalJSONTo writes the ordered object when present, so key order is kept,
// and falls back to the typed view otherwise.
func (m *Manifest) MarshalJSONTo(enc *jsontext.Encoder) error {
	if m.Object != nil {
		return m.Object.MarshalJSONTo(enc)
	}

	if err := json.MarshalEncode(enc, &m.PackageJSON); err != nil {
		return fmt.Errorf("encode package.json: %w", err)
	}
	return nil
}

// ReadFile reads and parses the package.json at path.
//
// The read error is returned untouched: callers detect a missing manifest with
// os.IsNotExist, which does not unwrap.
func ReadFile(path string) (*Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := new(Manifest)
	if err := json.Unmarshal(content, m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	m.FilePath = path
	return m, nil
}
