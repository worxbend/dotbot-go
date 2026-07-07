package core

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dotbot-go/internal/config"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	sh "dotbot-go/internal/shell"
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
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
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
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
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
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
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

func TestRelinkUpdatesRelativeSymlink(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "dotfiles")
	homeDir := filepath.Join(dir, "home")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "old-vimrc"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "vimrc"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(homeDir, ".vimrc")
	if err := os.Symlink("../dotfiles/old-vimrc", link); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{
				"path":     "dotfiles/vimrc",
				"relative": true,
				"relink":   true,
			},
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
	if got != "../dotfiles/vimrc" {
		t.Fatalf("link target = %q, want relative target", got)
	}
}

func TestForceReplacesExistingDirectoryWithSymlink(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "nvim")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, ".config", "nvim")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "init.lua"), []byte("-- old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			destination: map[string]any{"path": "nvim", "force": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}

	got, err := os.Readlink(destination)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("link target = %q, want %q", got, source)
	}
	if _, err := os.Stat(filepath.Join(destination, "init.lua")); !os.IsNotExist(err) {
		t.Fatalf("old directory contents still visible or unexpected stat error: %v", err)
	}
}

func TestIgnoreMissingCreatesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, ".missing")
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "missing", "ignore-missing": true},
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
	if got != filepath.Join(dir, "missing") {
		t.Fatalf("link target = %q, want missing target under base", got)
	}
}

func TestGlobPrefixAndExcludeCreatesExpectedLinks(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	sourceDir := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.conf", "b.conf", "skip.conf"} {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	linkDir := filepath.Join(dir, "links")

	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			linkDir: map[string]any{
				"path":    filepath.Join("dotfiles", "*.conf"),
				"glob":    true,
				"prefix":  "cfg-",
				"exclude": []any{filepath.Join("dotfiles", "skip.conf")},
				"create":  true,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}

	for _, name := range []string{"cfg-a.conf", "cfg-b.conf"} {
		if _, err := os.Readlink(filepath.Join(linkDir, name)); err != nil {
			t.Fatalf("missing globbed link %s: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(linkDir, "cfg-skip.conf")); !os.IsNotExist(err) {
		t.Fatalf("excluded glob link exists or unexpected stat error: %v", err)
	}
}

func TestShellDirectiveFailure(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
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

func TestShellDirectiveMergesDefaultsAndCommandOptions(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	logger := log.New(&out)
	logger.SetLevel(log.Debug)
	runner := &recordingShellRunner{}
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Options: Options{
			ShellTimeout: 3 * time.Second,
		},
		Logger:   logger,
		Handlers: BuiltIns(),
		Shell:    runner,
	})
	if err != nil {
		t.Fatal(err)
	}

	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"defaults": map[string]any{
			"shell": map[string]any{
				"stdin":  true,
				"stdout": false,
				"stderr": true,
				"quiet":  true,
			},
		}},
		{"shell": []any{
			map[string]any{
				"command":     "echo hi",
				"description": "say hi",
				"stdin":       false,
				"stdout":      true,
				"quiet":       false,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %#v, want one command", runner.commands)
	}
	if len(runner.options) != 1 {
		t.Fatalf("options = %#v, want one command options", runner.options)
	}
	if runner.commands[0] != "echo hi" {
		t.Fatalf("command = %q, want echo hi", runner.commands[0])
	}
	got := runner.options[0]
	wantCWD, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.CWD != wantCWD {
		t.Fatalf("CWD = %q, want %q", got.CWD, wantCWD)
	}
	if got.EnableStdin {
		t.Fatal("EnableStdin = true, want false from command override")
	}
	if !got.EnableStdout {
		t.Fatal("EnableStdout = false, want true from command override")
	}
	if !got.EnableStderr {
		t.Fatal("EnableStderr = false, want true from defaults")
	}
	if got.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want 3s", got.Timeout)
	}
	if !strings.Contains(out.String(), "say hi [echo hi]") {
		t.Fatalf("missing shell action log: %s", out.String())
	}
}

func TestPluginsDirectiveIsUnhandled(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"plugins": []any{"example.py"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatal("expected plugins directive to fail")
	}
	if !strings.Contains(out.String(), "Action plugins not handled") {
		t.Fatalf("missing unhandled directive output: %s", out.String())
	}
}

func TestValidateRejectsUnhandledDirective(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	err := dispatcher.Validate([]config.Task{
		{"plugins": []any{"example.py"}},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "action plugins not handled") {
		t.Fatalf("err = %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("validation wrote output: %s", out.String())
	}
}

func TestValidateRejectsMalformedCreateDirective(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	err := dispatcher.Validate([]config.Task{
		{"create": []any{42}},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "create directive item must be a string") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateRejectsInvalidCreateMode(t *testing.T) {
	cases := []struct {
		name  string
		tasks []config.Task
	}{
		{
			name: "default mode",
			tasks: []config.Task{
				{"defaults": map[string]any{
					"create": map[string]any{"mode": "invalid"},
				}},
				{"create": []any{"tmp"}},
			},
		},
		{
			name: "entry mode",
			tasks: []config.Task{
				{"create": map[string]any{
					"tmp": map[string]any{"mode": "invalid"},
				}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var out bytes.Buffer
			dispatcher := newTestDispatcher(t, dir, &out, Options{})
			err := dispatcher.Validate(tc.tasks)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "mode") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestValidateRejectsInvalidLinkType(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	err := dispatcher.Validate([]config.Task{
		{"link": map[string]any{
			filepath.Join(dir, ".vimrc"): map[string]any{
				"path": "vimrc",
				"type": "copy",
			},
		}},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "link type is not recognized") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateRejectsMalformedShellDirective(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	err := dispatcher.Validate([]config.Task{
		{"shell": []any{
			map[string]any{"description": "missing command"},
		}},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "shell directive item must include a command") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateHonorsOnlyFilter(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{Only: []string{"link"}})
	err := dispatcher.Validate([]config.Task{
		{"plugins": []any{"example.py"}},
	})
	if err != nil {
		t.Fatalf("validation failed for skipped action: %v", err)
	}
}

func TestPlanBuildsOperationList(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	plan, err := dispatcher.Plan([]config.Task{
		{"defaults": map[string]any{
			"link": map[string]any{"relative": true},
		}},
		{"create": []any{"~/.config", "~/.local/bin"}},
		{"link": map[string]any{
			filepath.Join(dir, ".vimrc"): "vimrc",
			filepath.Join(dir, ".config", "nvim"): map[string]any{
				"path": "nvim",
			},
		}},
		{"clean": []any{"~"}},
		{"shell": []any{
			"true",
			[]any{"echo hi", "say hi"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 7 {
		t.Fatalf("len(plan.Operations) = %d, want 7: %#v", len(plan.Operations), plan.Operations)
	}
	if !hasOperation(plan, Operation{Directive: "create", Target: "~/.config"}) {
		t.Fatalf("missing create operation: %#v", plan.Operations)
	}
	if !hasOperation(plan, Operation{Directive: "link", Target: filepath.Join(dir, ".vimrc"), Detail: "vimrc"}) {
		t.Fatalf("missing link operation: %#v", plan.Operations)
	}
	if !hasOperation(plan, Operation{Directive: "shell", Target: "echo hi", Detail: "say hi"}) {
		t.Fatalf("missing shell operation: %#v", plan.Operations)
	}
}

func TestPlanHonorsOnlyFilter(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{Only: []string{"shell"}})
	plan, err := dispatcher.Plan([]config.Task{
		{"create": []any{"~/.config"}},
		{"shell": []any{"true"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("len(plan.Operations) = %d, want 1: %#v", len(plan.Operations), plan.Operations)
	}
	if !hasOperation(plan, Operation{Directive: "shell", Target: "true"}) {
		t.Fatalf("missing shell operation: %#v", plan.Operations)
	}
}

func TestPlanSortsMapTargets(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	plan, err := dispatcher.Plan([]config.Task{
		{"create": map[string]any{
			"z-dir": nil,
			"a-dir": nil,
		}},
		{"clean": map[string]any{
			"z-clean": nil,
			"a-clean": nil,
		}},
		{"link": map[string]any{
			"z-link": "z-target",
			"a-link": "a-target",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []Operation{
		{Directive: "create", Target: "a-dir"},
		{Directive: "create", Target: "z-dir"},
		{Directive: "clean", Target: "a-clean"},
		{Directive: "clean", Target: "z-clean"},
		{Directive: "link", Target: "a-link", Detail: "a-target"},
		{Directive: "link", Target: "z-link", Detail: "z-target"},
	}
	if len(plan.Operations) != len(expected) {
		t.Fatalf("len(plan.Operations) = %d, want %d: %#v", len(plan.Operations), len(expected), plan.Operations)
	}
	for i, operation := range expected {
		if plan.Operations[i] != operation {
			t.Fatalf("operation[%d] = %#v, want %#v", i, plan.Operations[i], operation)
		}
	}
}

func TestPlanPreservesTaskActionOrder(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	plan, err := dispatcher.Plan([]config.Task{
		config.NewTask(
			config.Action{
				Directive: "shell",
				Data:      []any{"true"},
			},
			config.Action{
				Directive: "create",
				Data:      []any{"~/.config"},
			},
			config.Action{
				Directive: "clean",
				Data:      []any{"~"},
			},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"shell", "create", "clean"}
	if len(plan.Operations) != len(expected) {
		t.Fatalf("operation count = %d, want %d: %#v", len(plan.Operations), len(expected), plan.Operations)
	}
	for i, directive := range expected {
		if plan.Operations[i].Directive != directive {
			t.Fatalf("operation[%d].Directive = %q, want %q: %#v", i, plan.Operations[i].Directive, directive, plan.Operations)
		}
	}
}

func TestDispatchPreservesTaskActionOrder(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var got []string
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Logger:        log.New(&out),
		Handlers: []Handler{
			recordingHandler{directives: &got},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		config.NewTask(
			config.Action{Directive: "second", Data: nil},
			config.Action{Directive: "first", Data: nil},
			config.Action{Directive: "third", Data: nil},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	expected := []string{"second", "first", "third"}
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("dispatch order = %#v, want %#v", got, expected)
	}
}

func TestBackupUsesInjectedClock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "vimrc")
	if err := os.WriteFile(target, []byte("set number\n"), 0o600); err != nil {
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
		Clock: func() time.Time {
			return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		},
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
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	backupPath := link + ".dotbot-backup.20260102-030405"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("missing backup %q: %v", backupPath, err)
	}
}

func TestCleanRemovesBrokenSymlinkPointingIntoBase(t *testing.T) {
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
	if err := os.Symlink(filepath.Join(base, "vimrc"), link); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, base, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"clean": []any{home}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("clean kept broken base link or stat failed: %v", err)
	}
}

func TestCleanKeepsBrokenSymlinkOutsideBase(t *testing.T) {
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
		{"clean": []any{home}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !success {
		t.Fatalf("dispatch failed: %s", out.String())
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("clean removed outside-base link: %v", err)
	}
}

func TestCreateReportsChmodFailure(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	logger := log.New(&out)
	logger.SetLevel(log.Debug)
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Logger:        logger,
		Handlers:      BuiltIns(),
		FS:            chmodFailFS{OSFS: fsops.OSFS{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"create": []any{filepath.Join(dir, "ssh")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected chmod failure: %s", out.String())
	}
}

func TestRelinkReportsReadlinkFailure(t *testing.T) {
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
	logger := log.New(&out)
	logger.SetLevel(log.Debug)
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Logger:        logger,
		Handlers:      BuiltIns(),
		FS:            readlinkFailFS{OSFS: fsops.OSFS{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			link: map[string]any{"path": "vimrc", "relink": true},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected readlink failure: %s", out.String())
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != oldTarget {
		t.Fatalf("link target = %q, want original target %q", got, oldTarget)
	}
}

func TestRecursiveGlobReportsMalformedPattern(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "dotfiles")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vimrc"), []byte("set number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	dispatcher := newTestDispatcher(t, dir, &out, Options{})
	success, err := dispatcher.Dispatch(context.Background(), []config.Task{
		{"link": map[string]any{
			filepath.Join(dir, "home"): map[string]any{
				"path": filepath.Join(root, "**", "["),
				"glob": true,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if success {
		t.Fatalf("expected malformed glob failure: %s", out.String())
	}
	if !strings.Contains(out.String(), "Unable to expand glob") {
		t.Fatalf("missing glob warning: %s", out.String())
	}
}

type chmodFailFS struct {
	fsops.OSFS
}

func (fs chmodFailFS) Chmod(path string, mode os.FileMode) error {
	return errors.New("chmod failed")
}

type readlinkFailFS struct {
	fsops.OSFS
}

func (fs readlinkFailFS) Readlink(path string) (string, error) {
	return "", errors.New("readlink failed")
}

type recordingHandler struct {
	directives *[]string
}

type recordingShellRunner struct {
	commands []string
	options  []sh.Options
}

func (r *recordingShellRunner) Run(ctx context.Context, command string, opts sh.Options) int {
	r.commands = append(r.commands, command)
	r.options = append(r.options, opts)
	return 0
}

func (h recordingHandler) CanHandle(directive string) bool {
	return directive != "defaults"
}

func (recordingHandler) SupportsDryRun() bool {
	return true
}

func (h recordingHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	*h.directives = append(*h.directives, directive)
	return true, nil
}

func newTestDispatcher(t *testing.T, dir string, out *bytes.Buffer, opts Options) *Dispatcher {
	t.Helper()
	logger := log.New(out)
	logger.SetLevel(log.Debug)
	dispatcher, err := NewDispatcher(DispatcherConfig{
		BaseDirectory: dir,
		Options:       opts,
		Logger:        logger,
		Handlers:      BuiltIns(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return dispatcher
}

func hasOperation(plan Plan, expected Operation) bool {
	for _, operation := range plan.Operations {
		if operation == expected {
			return true
		}
	}
	return false
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
