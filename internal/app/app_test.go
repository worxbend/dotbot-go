package app

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"dotbot-go/internal/config"
)

func TestRunShowsVersion(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{ShowVersion: true}, &stdout, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Dotbot-Go version 0.2.1\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunShowsOverriddenVersion(t *testing.T) {
	originalVersion := Version
	Version = "9.8.7-test"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout bytes.Buffer
	err := Run(context.Background(), Options{ShowVersion: true}, &stdout, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Dotbot-Go version 9.8.7-test\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunRejectsConflictingColorFlags(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ForceColor: true,
		NoColor:    true,
	}, &stdout, Dependencies{})
	if err != ErrExit {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	if !strings.Contains(stdout.String(), "`--force-color` and `--no-color` cannot both be provided") {
		t.Fatalf("missing color conflict output: %q", stdout.String())
	}
}

func TestRunRejectsMissingConfig(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{}, &stdout, Dependencies{})
	if err != ErrExit {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	if !strings.Contains(stdout.String(), "No configuration file specified") {
		t.Fatalf("missing config output: %q", stdout.String())
	}
}

func TestBaseDirectoryRejectsMissingConfigFileFallback(t *testing.T) {
	if _, err := baseDirectory(Options{}); err == nil {
		t.Fatal("expected baseDirectory to reject empty config files without base directory")
	}
}

func TestRunUsesInjectedDependencies(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	var chdirPath string
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			if len(paths) != 1 || paths[0] != configPath {
				t.Fatalf("paths = %#v", paths)
			}
			return []config.Task{}, nil
		},
		Chdir: func(path string) error {
			chdirPath = path
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if chdirPath != dir {
		t.Fatalf("chdir path = %q, want %q", chdirPath, dir)
	}
}

func TestRunValidateDoesNotChangeDirectory(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Validate:    true,
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{
				{"create": []any{filepath.Join(dir, "created-by-apply")}},
			}, nil
		},
		Chdir: func(path string) error {
			t.Fatalf("validate changed directory to %q", path)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Configuration is valid") {
		t.Fatalf("missing validation output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "1 planned operation(s)") {
		t.Fatalf("missing operation count: %q", stdout.String())
	}
}

func TestRunValidateRejectsUnhandledDirective(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Validate:    true,
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{{"plugins": []any{"example.py"}}}, nil
		},
	})
	if err != ErrExit {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	if !strings.Contains(stdout.String(), "action plugins not handled") {
		t.Fatalf("missing validation error: %q", stdout.String())
	}
}

func TestRunPlanTextPrintsOperationsWithoutChangingDirectory(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Plan:        true,
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{
				{"create": []any{"~/.config"}},
				{"link": map[string]any{filepath.Join(dir, ".vimrc"): "vimrc"}},
				{"shell": []any{[]any{"echo hi", "say hi"}}},
			}, nil
		},
		Chdir: func(path string) error {
			t.Fatalf("plan changed directory to %q", path)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, expected := range []string{
		"Plan: 3 operation(s), 3 task(s), 1 config file(s), base " + dir,
		"create  ~/.config (0777)",
		"link    " + filepath.Join(dir, ".vimrc") + " -> vimrc",
		"shell   echo hi [say hi]",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q in plan output:\n%s", expected, got)
		}
	}
}

func TestRunPlanJSONPrintsStructuredOperations(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Plan:        true,
		Output:      "json",
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{
				{"create": []any{"~/.config"}},
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		TaskCount       int    `json:"task_count"`
		ConfigFileCount int    `json:"config_file_count"`
		OperationCount  int    `json:"operation_count"`
		Base            string `json:"base"`
		Operations      []struct {
			Directive string `json:"directive"`
			Target    string `json:"target"`
			Detail    string `json:"detail"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode plan json: %v\n%s", err, stdout.String())
	}
	if got.TaskCount != 1 || got.ConfigFileCount != 1 || got.OperationCount != 1 || got.Base != dir {
		t.Fatalf("unexpected plan summary: %#v", got)
	}
	if len(got.Operations) != 1 {
		t.Fatalf("operation count = %d, want 1", len(got.Operations))
	}
	if got.Operations[0].Directive != "create" || got.Operations[0].Target != "~/.config" || got.Operations[0].Detail != "0777" {
		t.Fatalf("operation = %#v", got.Operations[0])
	}
}

func TestRunPlanJSONIsNotPrefixedByEmptyConfigWarning(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Plan:        true,
		Output:      "json",
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(stdout.String(), "warn") {
		t.Fatalf("json output was prefixed by warning: %q", stdout.String())
	}
	var got struct {
		OperationCount int `json:"operation_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode plan json: %v\n%s", err, stdout.String())
	}
	if got.OperationCount != 0 {
		t.Fatalf("operation_count = %d, want 0", got.OperationCount)
	}
}

func TestRunPlanRejectsUnsupportedOutputFormat(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "install.conf.yaml")
	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigFiles: []string{configPath},
		Plan:        true,
		Output:      "xml",
	}, &stdout, Dependencies{
		ConfigReader: func(paths []string) ([]config.Task, error) {
			return []config.Task{{"create": []any{"~/.config"}}}, nil
		},
	})
	if err != ErrExit {
		t.Fatalf("err = %v, want ErrExit", err)
	}
	if !strings.Contains(stdout.String(), `unsupported output format "xml"`) {
		t.Fatalf("missing output format error: %q", stdout.String())
	}
}
