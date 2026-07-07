package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestHasGlobChars(t *testing.T) {
	cases := map[string]bool{
		"a*b":   true,
		"a?b":   true,
		"a[bc]": true,
		"plain": false,
		"":      false,
	}
	for in, want := range cases {
		if got := hasGlobChars(in); got != want {
			t.Errorf("hasGlobChars(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	cases := []struct {
		a, b, want string
	}{
		{"abcdef", "abcxyz", "abc"},
		{"abc", "abc", "abc"},
		{"", "abc", ""},
		{"abc", "ab", "ab"},
		{"xyz", "abc", ""},
	}
	for _, tc := range cases {
		if got := commonPrefix(tc.a, tc.b); got != tc.want {
			t.Errorf("commonPrefix(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestGlobLinkItem(t *testing.T) {
	cases := []struct {
		pattern, item, want string
	}{
		{filepath.Join("dotfiles", "*.conf"), filepath.Join("dotfiles", "a.conf"), "a.conf"},
		{"*.conf", "a.conf", "a.conf"},
		{filepath.Join("dotfiles", "**", "x"), filepath.Join("dotfiles", "sub", "x"), filepath.Join("sub", "x")},
	}
	for _, tc := range cases {
		if got := globLinkItem(tc.pattern, tc.item); got != tc.want {
			t.Errorf("globLinkItem(%q, %q) = %q, want %q", tc.pattern, tc.item, got, tc.want)
		}
	}
}

func TestDoublestarMatch(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"plain match", "*.conf", "a.conf", true},
		{"plain mismatch", "*.conf", "a.txt", false},
		{"doublestar dir match", filepath.Join("dir", "**"), filepath.Join("dir", "a"), true},
		{"doublestar wrong prefix", filepath.Join("dir", "**"), filepath.Join("other", "a"), false},
		{"doublestar suffix match", filepath.Join("dir", "**", "*.conf"), filepath.Join("dir", "sub", "a.conf"), true},
		{"doublestar suffix mismatch", filepath.Join("dir", "**", "*.conf"), filepath.Join("dir", "sub", "a.txt"), false},
		{"leading doublestar", filepath.Join("**", "*.conf"), filepath.Join("a", "b.conf"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := doublestarMatch(tc.pattern, tc.path)
			if err != nil {
				t.Fatalf("doublestarMatch(%q, %q) error: %v", tc.pattern, tc.path, err)
			}
			if got != tc.want {
				t.Fatalf("doublestarMatch(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}

func TestCreateGlobResults(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path string) {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "a.conf"))
	write(filepath.Join(root, "b.conf"))
	write(filepath.Join(sub, "c.conf"))

	t.Run("flat glob", func(t *testing.T) {
		got, err := createGlobResults(filepath.Join(root, "*.conf"), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{filepath.Join(root, "a.conf"), filepath.Join(root, "b.conf")}
		assertSameSet(t, got, want)
	})

	t.Run("flat glob with exclude", func(t *testing.T) {
		got, err := createGlobResults(filepath.Join(root, "*.conf"), []string{filepath.Join(root, "b.conf")})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{filepath.Join(root, "a.conf")}
		assertSameSet(t, got, want)
	})

	t.Run("doublestar glob", func(t *testing.T) {
		got, err := createGlobResults(filepath.Join(root, "**", "*.conf"), nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			filepath.Join(root, "a.conf"),
			filepath.Join(root, "b.conf"),
			filepath.Join(sub, "c.conf"),
		}
		assertSameSet(t, got, want)
	})

	t.Run("malformed include pattern errors", func(t *testing.T) {
		if _, err := createGlobResults("[", nil); err == nil {
			t.Fatal("expected error for malformed include pattern")
		}
	})

	t.Run("malformed exclude pattern errors", func(t *testing.T) {
		if _, err := createGlobResults(filepath.Join(root, "*.conf"), []string{"["}); err == nil {
			t.Fatal("expected error for malformed exclude pattern")
		}
	})
}

func assertSameSet(t *testing.T, got, want []string) {
	t.Helper()
	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	if strings.Join(gotCopy, "|") != strings.Join(wantCopy, "|") {
		t.Fatalf("set mismatch:\n got: %v\nwant: %v", gotCopy, wantCopy)
	}
}
