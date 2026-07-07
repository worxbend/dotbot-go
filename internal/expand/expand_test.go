package expand

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "bare tilde", in: "~", want: home},
		{name: "tilde slash", in: "~/foo", want: filepath.Join(home, "foo")},
		{name: "tilde nested", in: "~/foo/bar", want: filepath.Join(home, "foo", "bar")},
		{name: "absolute unchanged", in: "/etc/hosts", want: "/etc/hosts"},
		{name: "relative unchanged", in: "foo/bar", want: "foo/bar"},
		{name: "named tilde unchanged", in: "~user/foo", want: "~user/foo"},
		{name: "empty unchanged", in: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := User(tc.in); got != tc.want {
				t.Fatalf("User(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUserFallsBackWhenHomeMissing(t *testing.T) {
	// os.UserHomeDir reads $HOME on unix; with it empty, expansion is a no-op.
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not the home source on windows")
	}
	t.Setenv("HOME", "")
	if got := User("~/foo"); got != "~/foo" {
		t.Fatalf("User(%q) = %q, want input unchanged when HOME is empty", "~/foo", got)
	}
}

func TestPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EXPAND_TEST_VAR", "value")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "env only", in: "$EXPAND_TEST_VAR/x", want: filepath.Join("value", "x")},
		{name: "tilde and env", in: "~/$EXPAND_TEST_VAR", want: filepath.Join(home, "value")},
		{name: "plain", in: "plain/path", want: filepath.Join("plain", "path")},
		{name: "undefined env", in: "$EXPAND_TEST_MISSING/x", want: string(filepath.Separator) + "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Path(tc.in); got != tc.want {
				t.Fatalf("Path(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Run("relative becomes absolute", func(t *testing.T) {
		got := Abs("foo")
		if !filepath.IsAbs(got) {
			t.Fatalf("Abs(%q) = %q, want absolute path", "foo", got)
		}
		if filepath.Base(got) != "foo" {
			t.Fatalf("Abs(%q) = %q, want it to end in foo", "foo", got)
		}
	})

	t.Run("tilde resolves to home", func(t *testing.T) {
		// home is already absolute (t.TempDir), so Abs should return it unchanged.
		if got := Abs("~"); got != home {
			t.Fatalf("Abs(%q) = %q, want %q", "~", got, home)
		}
	})
}

func TestNormSlash(t *testing.T) {
	if runtime.GOOS == "windows" {
		if got := NormSlash("a/b/c"); got != `a\b\c` {
			t.Fatalf("NormSlash(%q) = %q, want backslash-separated", "a/b/c", got)
		}
		return
	}
	if got := NormSlash("a/b/c"); got != "a/b/c" {
		t.Fatalf("NormSlash(%q) = %q, want unchanged on non-windows", "a/b/c", got)
	}
}
