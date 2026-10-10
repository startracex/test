package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTrimExt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple extension", in: "main.go", want: "main"},
		{name: "multiple dots", in: "archive.tar.gz", want: "archive.tar"},
		{name: "no extension", in: "README", want: "README"},
		{name: "empty input", in: "", want: ""},
		{name: "trailing dot", in: "file.", want: "file"},
		{name: "with directory", in: filepath.Join("internal", "cmd", "run.go"), want: filepath.Join("internal", "cmd", "run")},
		{name: "directory with dot", in: filepath.Join("1.2.3", "index.js"), want: filepath.Join("1.2.3", "index")},
		{name: "directory without extension", in: filepath.Join("src", "lib"), want: filepath.Join("src", "lib")},
		{name: "dot file", in: ".gitignore", want: ".gitignore"},
		{name: "dot file with directory", in: filepath.Join("config", ".npmrc"), want: filepath.Join("config", ".npmrc")},
		{name: "dot file without extension text", in: ".git", want: ".git"},
		{name: "single char extension only", in: ".a", want: ".a"},
		{name: "dot file with nested directory", in: filepath.Join("a", "b", ".editorconfig"), want: filepath.Join("a", "b", ".editorconfig")},
		{name: "dot file with real extension", in: ".npmrc.bak", want: ".npmrc"},
		{name: "dot file with two dots", in: ".a.b", want: ".a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimExt(tt.in); got != tt.want {
				t.Errorf("TrimExt(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsPathLike(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "absolute path", in: "/abs", want: true},
		{name: "relative with dot slash", in: "./foo", want: true},
		{name: "relative with double dot", in: "../bar", want: true},
		{name: "deep dot path", in: "./a/b/c/d", want: true},
		{name: "deep dotdot path", in: "../x/y/z", want: true},
		{name: "three dots", in: "...", want: false},
		{name: "two dots then char", in: "..a", want: false},
		{name: "one dot then char", in: ".a", want: false},
		{name: "windows style backslash dot", in: ".\\foo", want: true},
		{name: "plain package name", in: "foo", want: false},
		{name: "scoped package name", in: "@scope/pkg", want: false},
		{name: "package with version", in: "foo@1.2.3", want: false},
		{name: "empty", in: "", want: false},
		{name: "single dot slash", in: "./", want: true},
		{name: "double dot slash", in: "../", want: true},
		{name: "single dot", in: ".", want: true},
		{name: "double dot", in: "..", want: true},
		{name: "single slash", in: "/", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPathLike(tt.in); got != tt.want {
				t.Errorf("IsPathLike(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPrettyPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	abs := filepath.Join(cwd, "abs", "path")
	absSlash := strings.ReplaceAll(filepath.Clean(abs), "\\", "/")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "bare name gets a dot slash prefix", in: "foo", want: "./foo"},
		{name: "leading dot slash is normalized", in: "./foo", want: "./foo"},
		{name: "nested relative path", in: "foo/bar", want: "./foo/bar"},
		{name: "parent relative path stays as is", in: "../bar", want: "../bar"},
		{name: "current directory", in: ".", want: "."},
		{name: "empty becomes dot", in: "", want: "."},
		{name: "absolute path stays as is", in: abs, want: absSlash},
		{name: "redundant separators are cleaned", in: "foo//bar", want: "./foo/bar"},
		{name: "backslash separators are normalized", in: `foo\bar`, want: "./foo/bar"},
		{name: "deep parent path", in: "../../foo", want: "../../foo"},
		{name: "absolute path", in: "/foo", want: "/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrettyPath(tt.in); got != tt.want {
				t.Errorf("PrettyPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	t.Run("absolute input is only cleaned", func(t *testing.T) {
		in := filepath.Join(cwd, "a", "b", "..", "c")
		want := filepath.Clean(in)
		got, err := Abs(in)
		if err != nil {
			t.Fatalf("Abs(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Errorf("Abs(%q) = %q, want %q", in, got, want)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("Abs(%q) = %q is not absolute", in, got)
		}
	})

	t.Run("relative input is resolved against the working directory", func(t *testing.T) {
		want, err := filepath.Abs("foo/bar")
		if err != nil {
			t.Fatalf("filepath.Abs failed: %v", err)
		}
		got, err := Abs("foo/bar")
		if err != nil {
			t.Fatalf("Abs returned error: %v", err)
		}
		if got != want {
			t.Errorf("Abs(\"foo/bar\") = %q, want %q", got, want)
		}
	})

	t.Run("empty input resolves to the working directory", func(t *testing.T) {
		want, err := filepath.Abs("")
		if err != nil {
			t.Fatalf("filepath.Abs failed: %v", err)
		}
		got, err := Abs("")
		if err != nil {
			t.Fatalf("Abs returned error: %v", err)
		}
		if got != want {
			t.Errorf("Abs(\"\") = %q, want %q", got, want)
		}
	})
}

func TestEquals(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "identical", a: "a/b/c", b: "a/b/c", want: true},
		{name: "different", a: "a/b/c", b: "a/b/d", want: false},
		{name: "empty", a: "", b: "", want: true},
		{name: "case difference", a: "A/B", b: "a/b", want: runtime.GOOS == "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equals(tt.a, tt.b); got != tt.want {
				t.Errorf("Equals(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestUpDir(t *testing.T) {
	t.Run("walks up until the directory stops changing", func(t *testing.T) {
		root := filepath.Join("a", "b", "c")
		want := []string{
			filepath.Join("a", "b", "c"),
			filepath.Join("a", "b"),
			"a",
			".",
		}

		var got []string
		for dir := range UpDir(root, filepath.Dir) {
			got = append(got, dir)
		}

		if len(got) != len(want) {
			t.Fatalf("UpDir yielded %d entries (%v), want %d (%v)", len(got), got, len(want), want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("UpDir entry %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("stops immediately when the directory function is stationary", func(t *testing.T) {
		var got []string
		for dir := range UpDir("root", func(s string) string { return s }) {
			got = append(got, dir)
		}
		if len(got) != 1 || got[0] != "root" {
			t.Fatalf("expected a single entry [root], got %v", got)
		}
	})

	t.Run("honors early termination", func(t *testing.T) {
		root := filepath.Join("a", "b", "c")
		var got []string
		for dir := range UpDir(root, filepath.Dir) {
			got = append(got, dir)
			if len(got) == 2 {
				break
			}
		}
		want := []string{filepath.Join("a", "b", "c"), filepath.Join("a", "b")}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("yields nothing when the consumer stops right away", func(t *testing.T) {
		count := 0
		for range UpDir("root", filepath.Dir) {
			count++
			break
		}
		if count != 1 {
			t.Fatalf("expected exactly one yield, got %d", count)
		}
	})
}
