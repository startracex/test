package npmconfig

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewNpmConfigMergesInOrder(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.npmrc")
	second := filepath.Join(dir, "second.npmrc")
	writeFile(t, first, "registry=https://first.example.com\nfoo=1\n")
	writeFile(t, second, "registry=https://second.example.com\n")

	ini, err := NewNpmConfig(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ini.Get("registry"); got != "https://second.example.com" {
		t.Errorf("registry = %q, later file should override", got)
	}
	if got, ok := ini.Get("foo"); !ok || got != "1" {
		t.Errorf("foo = %q, %v, key unique to earlier file must survive", got, ok)
	}
}

func TestNewNpmConfigMissingFilesAreEmpty(t *testing.T) {
	dir := t.TempDir()
	ini, err := NewNpmConfig(filepath.Join(dir, "does-not-exist.npmrc"))
	if err != nil {
		t.Fatalf("missing file should be treated as empty: %v", err)
	}
	if _, ok := ini.Get("anything"); ok {
		t.Error("expected empty config")
	}
}

func TestReloadPicksUpChanges(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".npmrc")
	writeFile(t, file, "registry=https://old.example.com\n")

	c := &NpmConfig{paths: []string{file}}
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Get("registry"); got != "https://old.example.com" {
		t.Fatalf("initial registry = %q", got)
	}

	writeFile(t, file, "registry=https://new.example.com\n")
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Get("registry"); got != "https://new.example.com" {
		t.Errorf("after reload registry = %q, want new", got)
	}
}

func TestLocationPathInvalidLocation(t *testing.T) {
	if _, err := LocationPath(LoadOptions{Location: "bogus"}, "registry"); !errors.Is(err, ErrInvalidLocation) {
		t.Errorf("invalid location error = %v, want ErrInvalidLocation", err)
	}
}

func TestLocationPathGlobal(t *testing.T) {
	got, err := LocationPath(LoadOptions{Location: LocationGlobal}, "anything")
	if err != nil {
		t.Fatal(err)
	}
	if got != globalPath() {
		t.Errorf("global path = %q, want %q", got, globalPath())
	}
}
