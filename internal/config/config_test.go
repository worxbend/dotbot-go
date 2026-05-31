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
