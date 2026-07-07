package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestReadPreservesYAMLTaskDirectiveOrder(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.conf.yaml", `
- shell:
    - [echo before, before]
  create:
    - tmp
  clean:
    - "~"
`, []string{"shell", "create", "clean"})
}

func TestReadPreservesJSONTaskDirectiveOrder(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.json", `[
  {
    "shell": [["echo before", "before"]],
    "create": ["tmp"],
    "clean": ["~"]
  }
]`, []string{"shell", "create", "clean"})
}

func TestReadPreservesJSON5TaskDirectiveOrder(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.json5", `[
  {
    // JSON5 allows comments, unquoted keys, and trailing commas.
    shell: [["echo before", "before"]],
    create: ["tmp"],
    clean: ["~"],
  },
]`, []string{"shell", "create", "clean"})
}

func TestReadPreservesJSON5OrderWithDelimitersInStringsAndComments(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.json5", `[
  {
    // Delimiters in comments must not close the shell value: ] } ,
    shell: [
      ["printf '{still a string}, with comma'", "message, with ] delimiters"],
    ],
    /* Delimiters in block comments must not close the create value: ] } , */
    create: [
      "tmp-{literal},[]",
    ],
    clean: ["~"],
  },
]`, []string{"shell", "create", "clean"})
}

func TestReadPreservesJSON5QuotedKeysWithDelimiters(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.json5", `[
  {
    "shell:with,delimiters}": ["true"],
    create: ["tmp"],
  },
]`, []string{"shell:with,delimiters}", "create"})
}

func TestReadPreservesTOMLTaskDirectiveOrder(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.toml", `
tasks = [
  { shell = [["echo before", "before"]], create = ["tmp"], clean = ["~"] },
]
`, []string{"shell", "create", "clean"})
}

func TestReadPreservesTOMLArrayTableTaskDirectiveOrder(t *testing.T) {
	assertReadPreservesDirectiveOrder(t, "install.toml", `
[[tasks]]
shell = [["echo before", "before"]]
create = ["tmp"]
clean = ["~"]
`, []string{"shell", "create", "clean"})
}

func assertReadPreservesDirectiveOrder(t *testing.T, file, content string, expected []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), file)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tasks, err := Read([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	got := actionDirectives(tasks[0].Actions())
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("action order = %#v, want %#v", got, expected)
	}
}

func TestReadDecodesTOMLStringsInOrderedTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "install.toml")
	if err := os.WriteFile(path, []byte(`
tasks = [
  { shell = [["echo hi", "line\nnext"]] },
]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, err := Read([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	actions := tasks[0].Actions()
	if len(actions) != 1 || actions[0].Directive != "shell" {
		t.Fatalf("actions = %#v, want one shell action", actions)
	}
	items, ok := actions[0].Data.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("shell data = %#v, want one shell item", actions[0].Data)
	}
	command, ok := items[0].([]any)
	if !ok || len(command) != 2 {
		t.Fatalf("shell item = %#v, want command and description", items[0])
	}
	if command[1] != "line\nnext" {
		t.Fatalf("description = %#v, want decoded newline", command[1])
	}
}

func TestExamplesParse(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	paths := []string{
		filepath.Join(root, "examples", "install.conf.yaml"),
		filepath.Join(root, "examples", "install.json"),
		filepath.Join(root, "examples", "install.json5"),
		filepath.Join(root, "examples", "install.toml"),
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			tasks, err := Read([]string{path})
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) == 0 {
				t.Fatal("expected example to contain tasks")
			}
		})
	}
}

func actionDirectives(actions []Action) []string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		out = append(out, action.Directive)
	}
	return out
}

func TestReadRejectsUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"install.ini", "install.conf", "install.hocon"} {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(dir, file)
			if err := os.WriteFile(path, []byte("create=[]\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Read([]string{path})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "unsupported config file format") {
				t.Fatalf("error = %q, want unsupported format", err.Error())
			}
		})
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
