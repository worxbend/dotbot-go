package config

import (
	"reflect"
	"testing"
)

func TestTOMLNestedTablesPreserveValuesAndDirectiveOrder(t *testing.T) {
	for _, tc := range []struct {
		name, document string
	}{
		{name: "nested table", document: `
[[tasks]]
shell = ["before"]
[tasks.link]
"destination" = { path = "source", create = true }
[tasks.create]
"directory" = { mode = 448 }
[[tasks]]
shell = ["after"]
`},
		{name: "nested array table", document: `
[[tasks]]
[[tasks.shell]]
command = "before"
[tasks.link]
"destination" = { path = "source", create = true }
[tasks.create]
"directory" = { mode = 448 }
[[tasks]]
shell = ["after"]
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := (TOMLParser{}).Parse("test.toml", []byte(tc.document))
			if err != nil {
				t.Fatal(err)
			}
			tasks, err := tasksFromRaw("test.toml", raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) != 2 || !reflect.DeepEqual(actionDirectives(tasks[0].Actions()), []string{"shell", "link", "create"}) {
				t.Fatalf("tasks lost values or order: %#v", tasks)
			}
			links, ok := tasks[0]["link"].(map[string]any)
			if !ok {
				t.Fatalf("link data = %#v", tasks[0]["link"])
			}
			options, ok := links["destination"].(map[string]any)
			if !ok || options["path"] != "source" || options["create"] != true {
				t.Fatalf("nested link options = %#v", links)
			}
			if got := tasks[1]["shell"]; !reflect.DeepEqual(got, []any{"after"}) {
				t.Fatalf("second task = %#v", tasks[1])
			}
		})
	}
}

func TestTOMLIgnoresUnrelatedNestedTaskKeys(t *testing.T) {
	raw, err := (TOMLParser{}).Parse("test.toml", []byte(`
tasks = [{ shell = ["real"] }]
[metadata]
tasks = [{ shell = ["unrelated"] }]
`))
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := tasksFromRaw("test.toml", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || !reflect.DeepEqual(tasks[0]["shell"], []any{"real"}) {
		t.Fatalf("unrelated table changed tasks: %#v", tasks)
	}
}

func TestYAMLMergePreservesOptionsPrecedenceAndOrder(t *testing.T) {
	raw, err := (YAMLParser{}).Parse("test.yaml", []byte(`
- defaults:
    link: &first {create: true, force: false}
- defaults:
    link: &second {create: false, relink: true}
- link:
    destination:
      <<: [*first, *second]
      path: source
      force: true
- <<: &directives {shell: ["before"], create: ["directory"]}
  clean: ["directory"]
`))
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := tasksFromRaw("test.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	options := tasks[2]["link"].(map[string]any)["destination"].(map[string]any)
	want := map[string]any{"create": true, "relink": true, "force": true, "path": "source"}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("merged options = %#v, want %#v", options, want)
	}
	if got := actionDirectives(tasks[3].Actions()); !reflect.DeepEqual(got, []string{"shell", "create", "clean"}) {
		t.Fatalf("merged directive order = %v", got)
	}
}

func TestYAMLRejectsInvalidAliasesAndMerges(t *testing.T) {
	for _, tc := range []struct{ name, document string }{
		{name: "recursive alias", document: "- link: &loop {destination: *loop}"},
		{name: "duplicate key", document: "- link: {destination: one, destination: two}"},
		{name: "invalid merge", document: "- link: {destination: {<<: 42}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := (YAMLParser{}).Parse("test.yaml", []byte(tc.document)); err == nil {
				t.Fatal("accepted invalid YAML")
			}
		})
	}
}
