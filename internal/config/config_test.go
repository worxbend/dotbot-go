package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConcatenatesYAMLAndJSON(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "install.conf.yaml")
	jsonPath := filepath.Join(dir, "extra.json")
	if err := os.WriteFile(yamlPath, []byte("- create:\n  - tmp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte(`[{"shell":["true"]}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	tasks, err := Read([]string{yamlPath, jsonPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if _, ok := tasks[0]["create"]; !ok {
		t.Fatalf("first task missing create: %#v", tasks[0])
	}
	if _, ok := tasks[1]["shell"]; !ok {
		t.Fatalf("second task missing shell: %#v", tasks[1])
	}
}

func TestReadRejectsNonListConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("create: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read([]string{path}); err == nil {
		t.Fatal("expected error")
	}
}

func TestReadSupportsConfigFormats(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		file    string
		content string
		wantKey string
	}{
		{
			name:    "yaml",
			file:    "install.yaml",
			content: "- create:\n  - tmp\n",
			wantKey: "create",
		},
		{
			name:    "json",
			file:    "install.json",
			content: `[{"shell":["true"]}]`,
			wantKey: "shell",
		},
		{
			name: "json5",
			file: "install.json5",
			content: `[
				// comments are accepted
				{shell: ["true",],},
			]`,
			wantKey: "shell",
		},
		{
			name: "toml",
			file: "install.toml",
			content: `
				tasks = [
				  { create = ["tmp"] },
				]
			`,
			wantKey: "create",
		},
		{
			name: "hocon",
			file: "install.conf",
			content: `
				tasks = [
				  { shell = ["true"] }
				]
			`,
			wantKey: "shell",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.file)
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			tasks, err := Read([]string{path})
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) != 1 {
				t.Fatalf("len(tasks) = %d, want 1", len(tasks))
			}
			if _, ok := tasks[0][tt.wantKey]; !ok {
				t.Fatalf("task missing %q: %#v", tt.wantKey, tasks[0])
			}
		})
	}
}

func TestReadRejectsUnsupportedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "install.ini")
	if err := os.WriteFile(path, []byte("create=[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read([]string{path}); err == nil {
		t.Fatal("expected error")
	}
}

func TestReaderUsesInjectedRegistry(t *testing.T) {
	registry := NewRegistry()
	registry.Register([]string{".test"}, ParserFunc(func(path string, data []byte) (any, error) {
		return []any{map[string]any{"create": []any{"tmp"}}}, nil
	}))
	reader := NewReader(registry)
	reader.readFile = func(path string) ([]byte, error) {
		return []byte("custom"), nil
	}

	tasks, err := reader.Read([]string{"install.test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	if _, ok := tasks[0]["create"]; !ok {
		t.Fatalf("task missing create: %#v", tasks[0])
	}
}
