package manifest

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"sync"

	"github.com/startracex/wangpkg/pkg/orderedobject"
)

type Manifest struct {
	PackageJSON
	Object   *orderedobject.Object
	mu       sync.RWMutex
	FilePath string
}

func (m *Manifest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	value, err := dec.ReadValue()
	if err != nil {
		return err
	}

	o := orderedobject.New()
	if err := json.Unmarshal(value, o); err != nil {
		return err
	}

	p := new(PackageJSON)
	if err := json.Unmarshal(value, p); err != nil {
		return err
	}
	m.PackageJSON = *p
	m.Object = o
	return nil
}

func (m *Manifest) MarshalJSONTo(enc *jsontext.Encoder) error {
	if m.Object != nil {
		return m.Object.MarshalJSONTo(enc)
	}

	if err := json.MarshalEncode(enc, &m.PackageJSON); err != nil {
		return err
	}
	return nil
}

func ReadFile(path string) (*Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := new(Manifest)
	if err := json.Unmarshal(content, m); err != nil {
		return nil, err
	}
	m.FilePath = path
	return m, nil
}
