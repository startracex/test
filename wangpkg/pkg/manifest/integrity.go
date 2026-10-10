package manifest

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

type Integrity string

func (i Integrity) String() string {
	return string(i)
}

func (i Integrity) Hash() string {
	x := strings.IndexByte(string(i), '-')
	if x == -1 {
		return ""
	}
	return string(i[x+1:])
}

func (i Integrity) Hex() (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(i.Hash())
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decodedBytes), nil
}

func (i Integrity) Algorithm() string {
	x := strings.IndexByte(string(i), '-')
	if x == -1 {
		return ""
	}
	return string(i[:x])
}
