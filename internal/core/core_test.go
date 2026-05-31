package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dotbot-go/internal/config"
	"dotbot-go/internal/log"
)

func TestCreateAndLinkSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "tmux.conf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("set -g mouse on\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "home", ".tmux.conf")
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch([]config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "dotfiles/tmux.conf", "create": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func TestExistingRegularFileBlocksSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zshrc"), []byte("source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(link, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch([]config.Task{
		{"link": map[string]any{link: "zshrc"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatal("expected dispatch failure")
	}
	if !strings.Contains(out.String(), "already exists but is a regular file or directory") {
		t.Fatalf("missing regular-file warning: %s", out.String())
	}
}

func TestDryRunDoesNotCreateLink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vimrc"), []byte("set number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{DryRun: true})
	success, err := dispatcher.Dispatch([]config.Task{
		{"link": map[string]any{link: "vimrc"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("dry run created path or unexpected stat error: %v", err)
	}
	if !strings.Contains(out.String(), "Would create symlink") {
		t.Fatalf("missing dry-run output: %s", out.String())
	}
}

func TestShellDirectiveFailure(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch([]config.Task{
		{"shell": []any{"exit 7"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatal("expected shell failure")
	}
	if !strings.Contains(out.String(), "Command [exit 7] failed") {
		t.Fatalf("missing shell warning: %s", out.String())
	}
}

func newTestDispatcher(t *testing.T, dir string, out *bytes.Buffer, opts Options) *Dispatcher {
	t.Helper()
	logger := log.New(out)
	logger.SetLevel(log.Debug)
	dispatcher, err := NewDispatcher(dir, opts, logger, BuiltIns())
	if err != nil {
		t.Fatal(err)
	}
	return dispatcher
}
