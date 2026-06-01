package app

import (
	"bytes"
	"context"
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
	if got := stdout.String(); got != "Dotbot-Go version 0.1.0\n" {
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
