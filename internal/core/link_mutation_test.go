package core

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dotbot-go/internal/config"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
)

func TestHardlinkCreatesHardlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("set number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "type": "hardlink"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	linkInfo, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("created a symlink, want a hardlink")
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(linkInfo, targetInfo) {
		t.Fatalf("link %q is not the same file as %q", link, target)
	}
}

func TestHardlinkExistingSameFileSucceeds(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("set number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.Link(target, link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "type": "hardlink"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if !strings.Contains(out.String(), "Link exists") {
		t.Fatalf("missing 'Link exists' output: %s", out.String())
	}
}

func TestHardlinkOverExistingSymlinkWarns(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("set number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "type": "hardlink"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected failure replacing symlink with hardlink: %s", out.String())
	}
	if !strings.Contains(out.String(), "is a symbolic link, not a hard link") {
		t.Fatalf("missing symlink/hardlink mismatch warning: %s", out.String())
	}
}

func TestIncorrectSymlinkWarns(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(dir, "wrong")
	if err := os.WriteFile(wrong, []byte("wrong\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.Symlink(wrong, link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{link: "vimrc"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected failure for mismatched existing symlink: %s", out.String())
	}
	if !strings.Contains(out.String(), "Incorrect link") {
		t.Fatalf("missing 'Incorrect link' warning: %s", out.String())
	}
}

func TestInvalidSymlinkWarns(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{link: "vimrc"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected failure for dangling existing symlink: %s", out.String())
	}
	if !strings.Contains(out.String(), "Invalid link") {
		t.Fatalf("missing 'Invalid link' warning: %s", out.String())
	}
}

func TestBackupDryRunLeavesOriginal(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.WriteFile(link, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{DryRun: true})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "backup": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if !strings.Contains(out.String(), "Would backup") {
		t.Fatalf("missing dry-run backup output: %s", out.String())
	}
	content, err := os.ReadFile(link)
	if err != nil {
		t.Fatalf("original file removed during dry run: %v", err)
	}
	if string(content) != "existing\n" {
		t.Fatalf("original content changed during dry run: %q", content)
	}
}

func TestBackupFailureReportsWarning(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.WriteFile(link, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	logger := log.New(&out)
	logger.SetLevel(log.Debug)
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Logger:        logger,
		Handlers:      BuiltIns(),
		FS:            renameFailFS{OSFS: fsops.OSFS{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "backup": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected backup failure: %s", out.String())
	}
	if !strings.Contains(out.String(), "Failed to backup file") {
		t.Fatalf("missing backup failure warning: %s", out.String())
	}
}

func TestForceReplacesExistingRegularFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.WriteFile(link, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "force": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if !strings.Contains(out.String(), "Removing") {
		t.Fatalf("missing removal output: %s", out.String())
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("force did not replace regular file with symlink: %v", err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func TestSameFileGuardSkipsDestructiveAction(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	// A hardlink makes link and target the same underlying file.
	if err := os.Link(target, link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "force": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected same-file guard to fail the link: %s", out.String())
	}
	if !strings.Contains(out.String(), "appears to be the same file") {
		t.Fatalf("missing same-file warning: %s", out.String())
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target was destroyed by same-file path: %v", err)
	}
}

func TestRelinkDryRunKeepsExistingSymlink(t *testing.T) {
	dir := t.TempDir()
	oldTarget := filepath.Join(dir, "old-vimrc")
	if err := os.WriteFile(oldTarget, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	newTarget := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(newTarget, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	if err := os.Symlink(oldTarget, link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{DryRun: true})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "relink": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if !strings.Contains(out.String(), "Would remove") {
		t.Fatalf("missing dry-run removal output: %s", out.String())
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != oldTarget {
		t.Fatalf("dry run mutated link: target = %q, want %q", got, oldTarget)
	}
}

func TestLinkIfConditionSkips(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".vimrc")
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "if": "exit 1"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if !strings.Contains(out.String(), "Skipping") {
		t.Fatalf("missing skip output: %s", out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("link created despite failing 'if' condition: %v", err)
	}
}

func TestCleanMapFormForceRemovesLinkOutsideBase(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "dotfiles")
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".vimrc")
	if err := os.Symlink(filepath.Join(dir, "elsewhere", "vimrc"), link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, base, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"clean": map[string]any{
			home: map[string]any{"force": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("force clean kept out-of-base broken link or stat failed: %v", err)
	}
}

func TestCleanMapFormRecursiveRemovesNestedBrokenLink(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "dotfiles")
	sub := filepath.Join(dir, "home", "sub")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(sub, ".vimrc")
	if err := os.Symlink(filepath.Join(base, "vimrc"), link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, base, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"clean": map[string]any{
			filepath.Join(dir, "home"): map[string]any{"recursive": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("recursive clean kept nested broken link or stat failed: %v", err)
	}
}

type renameFailFS struct {
	fsops.OSFS
}

func (renameFailFS) Rename(oldpath, newpath string) error {
	return errors.New("rename failed")
}
