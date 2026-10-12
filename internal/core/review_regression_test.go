package core

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dotbot-go/internal/config"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/shell"
)

func TestBackupFailureKeepsOriginalWithForce(t *testing.T) {
	for _, failure := range []struct {
		name string
		fs   fsops.FS
	}{
		{name: "rename", fs: renameFailFS{}},
		{name: "reservation", fs: backupReservationFailFS{}},
	} {
		t.Run(failure.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source")
			destination := filepath.Join(dir, "destination")
			writeReviewFile(t, source, "new")
			writeReviewFile(t, destination, "original")
			d, err := NewDispatcher(DispatcherConfig{BaseDirectory: dir, FS: failure.fs})
			if err != nil {
				t.Fatal(err)
			}
			success, err := d.Dispatch(context.Background(), []config.Task{{"link": map[string]any{
				destination: map[string]any{"path": source, "backup": true, "force": true},
			}}})
			if err != nil || success {
				t.Fatalf("success = %v, error = %v; expected failed backup", success, err)
			}
			got, err := os.ReadFile(destination)
			if err != nil || string(got) != "original" {
				t.Fatalf("original = %q, error = %v", got, err)
			}
			info, err := os.Lstat(destination)
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("original replaced: %v", err)
			}
		})
	}
}

type backupReservationFailFS struct{ fsops.OSFS }

func (backupReservationFailFS) MkdirTemp(string, string) (string, error) {
	return "", errors.New("backup reservation failed")
}

func TestBackupsPreserveEveryVersionAtSameTimestamp(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "file"
		if directory {
			name = "directory"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source")
			destination := filepath.Join(dir, "destination")
			writeReviewFile(t, source, "new")
			clock := func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
			legacy := destination + ".dotbot-backup.20260102-030405"
			writeReviewFile(t, legacy, "previous backup")
			d, err := NewDispatcher(DispatcherConfig{BaseDirectory: dir, Clock: clock})
			if err != nil {
				t.Fatal(err)
			}
			for _, content := range []string{"first", "second"} {
				if directory {
					if err := os.Mkdir(destination, 0o700); err != nil {
						t.Fatal(err)
					}
					writeReviewFile(t, filepath.Join(destination, "content"), content)
				} else {
					writeReviewFile(t, destination, content)
				}
				success, err := d.Dispatch(context.Background(), []config.Task{{"link": map[string]any{
					destination: map[string]any{"path": source, "backup": true},
				}}})
				if err != nil || !success {
					t.Fatalf("success = %v, error = %v", success, err)
				}
				if err := os.Remove(destination); err != nil {
					t.Fatal(err)
				}
			}
			backups, err := filepath.Glob(legacy + "-*")
			if err != nil || len(backups) != 2 {
				t.Fatalf("backups = %v, error = %v", backups, err)
			}
			versions := map[string]bool{}
			for _, backup := range backups {
				path := filepath.Join(backup, "destination")
				if directory {
					path = filepath.Join(path, "content")
				}
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				versions[string(got)] = true
			}
			if !versions["first"] || !versions["second"] {
				t.Fatalf("backup versions = %v", versions)
			}
			got, err := os.ReadFile(legacy)
			if err != nil || string(got) != "previous backup" {
				t.Fatalf("legacy backup = %q, error = %v", got, err)
			}
		})
	}
}

func TestConditionalLinkDryRunNeverExecutesShell(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer
	d := newTestDispatcher(t, dir, &output, Options{DryRun: true})
	runner := &recordingShellRunner{}
	d.ctx.Shell = runner
	destination := filepath.Join(dir, "destination")
	success, err := d.Dispatch(context.Background(), []config.Task{{"link": map[string]any{
		destination: map[string]any{"path": "missing", "if": "touch side-effect", "force": true},
	}}})
	if err != nil || !success || len(runner.commands) != 0 {
		t.Fatalf("success = %v, error = %v, commands = %v", success, err, runner.commands)
	}
	if !strings.Contains(output.String(), "conditional link preview skipped") {
		t.Fatalf("missing conditional preview: %s", output.String())
	}
}

func TestLinkResolvesAbsoluteAndExpandedSources(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	writeReviewFile(t, source, "source")
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("DOTBOT_TEST_SOURCE", source)
	for _, tc := range []struct{ name, path string }{
		{name: "absolute", path: source},
		{name: "home", path: "~/source"},
		{name: "environment", path: "$DOTBOT_TEST_SOURCE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDispatcher(DispatcherConfig{BaseDirectory: dir})
			if err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(dir, tc.name)
			success, err := d.Dispatch(context.Background(), []config.Task{{"link": map[string]any{destination: tc.path}}})
			if err != nil || !success {
				t.Fatalf("success = %v, error = %v", success, err)
			}
			got, err := os.Readlink(destination)
			if err != nil || got != source {
				t.Fatalf("link = %q, error = %v", got, err)
			}
		})
	}
}

func TestLinkRejectsWrongOptionTypes(t *testing.T) {
	d, err := NewDispatcher(DispatcherConfig{BaseDirectory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"relative", "canonicalize", "canonicalize-path", "force", "relink", "create", "glob", "backup", "ignore-missing", "prefix", "if", "exclude", "type", "path"} {
		t.Run(key, func(t *testing.T) {
			for _, defaults := range []bool{false, true} {
				values := map[string]any{key: 42}
				task := config.Task{"link": map[string]any{"destination": values}}
				if defaults {
					task = config.Task{"defaults": map[string]any{"link": values}}
				}
				if err := d.Validate([]config.Task{task}); err == nil {
					t.Fatalf("accepted invalid %s, defaults = %v", key, defaults)
				}
				success, _ := d.Dispatch(context.Background(), []config.Task{task})
				if success {
					t.Fatalf("applied invalid %s, defaults = %v", key, defaults)
				}
			}
		})
	}
}

type defaultsProbe struct {
	defaults []map[string]any
	plans    int
}

func (*defaultsProbe) CanHandle(directive string) bool { return directive == "probe" }
func (*defaultsProbe) SupportsDryRun() bool            { return true }
func (h *defaultsProbe) Handle(ctx *Context, _ string, _ any) (bool, error) {
	h.defaults = append(h.defaults, ctx.Defaults)
	return true, nil
}
func (h *defaultsProbe) Plan(_ *Context, directive string, _ any) ([]Operation, error) {
	h.plans++
	return []Operation{{Directive: directive}}, nil
}

func TestDispatcherInvocationsKeepDefaultsLocal(t *testing.T) {
	probe := &defaultsProbe{}
	d, err := NewDispatcher(DispatcherConfig{BaseDirectory: t.TempDir(), Handlers: []Handler{probe}})
	if err != nil {
		t.Fatal(err)
	}
	tasks := []config.Task{{"probe": nil}, {"defaults": map[string]any{"link": map[string]any{"force": true}}}}
	if _, err := d.Plan(tasks); err != nil {
		t.Fatal(err)
	}
	for _, invocation := range [][]config.Task{tasks, {{"probe": nil}}} {
		if success, err := d.Dispatch(context.Background(), invocation); err != nil || !success {
			t.Fatalf("success = %v, error = %v", success, err)
		}
	}
	for _, defaults := range probe.defaults {
		if len(defaults) != 0 {
			t.Fatalf("defaults leaked between invocations: %v", defaults)
		}
	}
}

func TestDispatcherUsesSameHandlerForPlanAndApply(t *testing.T) {
	first, second := &defaultsProbe{}, &defaultsProbe{}
	d, err := NewDispatcher(DispatcherConfig{BaseDirectory: t.TempDir(), Handlers: []Handler{first, second}})
	if err != nil {
		t.Fatal(err)
	}
	tasks := []config.Task{{"probe": nil}}
	if _, err := d.Plan(tasks); err != nil {
		t.Fatal(err)
	}
	if success, err := d.Dispatch(context.Background(), tasks); err != nil || !success {
		t.Fatalf("success = %v, error = %v", success, err)
	}
	if first.plans != 1 || len(first.defaults) != 1 || second.plans != 0 || len(second.defaults) != 0 {
		t.Fatalf("handlers: first = %+v, second = %+v", first, second)
	}
}

type traversalFS struct{ fsops.OSFS }

func (traversalFS) Glob(string) ([]string, error) { return []string{"virtual/a.conf"}, nil }
func (traversalFS) WalkDir(_ string, visit fs.WalkDirFunc) error {
	return visit("virtual/sub/b.conf", reviewDirEntry{}, nil)
}
func (traversalFS) Realpath(path string) (string, error) {
	if path == "alias" {
		return "virtual", nil
	}
	return "", os.ErrNotExist
}

type reviewDirEntry struct{}

func (reviewDirEntry) Name() string               { return "b.conf" }
func (reviewDirEntry) IsDir() bool                { return false }
func (reviewDirEntry) Type() fs.FileMode          { return 0 }
func (reviewDirEntry) Info() (fs.FileInfo, error) { return nil, os.ErrNotExist }

func TestTraversalAndCanonicalizationUseInjectedFilesystem(t *testing.T) {
	adapter := traversalFS{}
	for _, pattern := range []string{"virtual/*.conf", "virtual/**/*.conf"} {
		t.Run(pattern, func(t *testing.T) {
			matches, err := createGlobResults(adapter, pattern, nil)
			if err != nil || len(matches) != 1 {
				t.Fatalf("matches = %v, error = %v", matches, err)
			}
		})
	}
	if !inDirectory(adapter, filepath.Join("virtual", "missing"), "alias") {
		t.Fatal("cleanup did not canonicalize through injected filesystem")
	}
}

type interruptRunner struct{ commands []string }

func (r *interruptRunner) Run(_ context.Context, command string, _ shell.Options) int {
	r.commands = append(r.commands, command)
	return 130
}

func TestInterruptedCommandsAbortRemainingTasks(t *testing.T) {
	for _, conditional := range []bool{false, true} {
		name := "shell"
		if conditional {
			name = "condition"
		}
		t.Run(name, func(t *testing.T) {
			runner := &interruptRunner{}
			dir := t.TempDir()
			d, err := NewDispatcher(DispatcherConfig{BaseDirectory: dir, Shell: runner})
			if err != nil {
				t.Fatal(err)
			}
			task := config.Task{"shell": []any{"interrupted", "must not run"}}
			if conditional {
				task = config.Task{"link": map[string]any{
					filepath.Join(dir, "link"): map[string]any{"path": "source", "if": "interrupted"},
				}}
			}
			tasks := []config.Task{task, {"shell": []any{"later directive must not run"}}}
			success, err := d.Dispatch(context.Background(), tasks)
			if success || !errors.Is(err, context.Canceled) || len(runner.commands) != 1 {
				t.Fatalf("success = %v, error = %v, commands = %v", success, err, runner.commands)
			}
		})
	}
}

func writeReviewFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
