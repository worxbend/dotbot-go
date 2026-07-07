package config

import "testing"

func TestActionsSortsWithoutExplicitOrder(t *testing.T) {
	task := Task{
		"shell":  []any{"a"},
		"create": []any{"b"},
		"link":   map[string]any{},
	}
	actions := task.Actions()
	got := make([]string, len(actions))
	for i, action := range actions {
		got[i] = action.Directive
	}
	want := []string{"create", "link", "shell"}
	if len(got) != len(want) {
		t.Fatalf("directives = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("directives = %v, want %v", got, want)
		}
	}
}

func TestNewTaskDedupesAndDropsEmptyDirectives(t *testing.T) {
	task := NewTask(
		Action{Directive: "", Data: "ignored"},
		Action{Directive: "link", Data: "first"},
		Action{Directive: "link", Data: "second"},
		Action{Directive: "shell", Data: "s"},
	)
	actions := task.Actions()
	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2: %#v", len(actions), actions)
	}
	if actions[0].Directive != "link" || actions[1].Directive != "shell" {
		t.Fatalf("directive order = %v, want [link shell]", actions)
	}
	if actions[0].Data != "second" {
		t.Fatalf("link data = %v, want last value 'second'", actions[0].Data)
	}
}

func TestNewTaskEmptyReturnsNoActions(t *testing.T) {
	if actions := NewTask().Actions(); len(actions) != 0 {
		t.Fatalf("len(actions) = %d, want 0", len(actions))
	}
}

func TestNormalizeExtension(t *testing.T) {
	cases := map[string]string{
		"yaml":  ".yaml",
		".YML":  ".yml",
		".JSON": ".json",
		"":      "",
	}
	for in, want := range cases {
		if got := normalizeExtension(in); got != want {
			t.Errorf("normalizeExtension(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeValueConvertsMapAnyAny(t *testing.T) {
	in := map[any]any{
		1:   "x",
		"k": map[any]any{"n": 2},
	}
	out, ok := normalizeValue(in).(map[string]any)
	if !ok {
		t.Fatalf("normalizeValue did not return map[string]any: %T", normalizeValue(in))
	}
	if out["1"] != "x" {
		t.Fatalf("out[\"1\"] = %v, want x", out["1"])
	}
	nested, ok := out["k"].(map[string]any)
	if !ok {
		t.Fatalf("nested value is not map[string]any: %T", out["k"])
	}
	if nested["n"] != 2 {
		t.Fatalf("nested[\"n\"] = %v, want 2", nested["n"])
	}
}

func TestNormalizeValueConvertsSliceOfMaps(t *testing.T) {
	in := []map[string]any{{"a": 1}}
	out, ok := normalizeValue(in).([]any)
	if !ok {
		t.Fatalf("normalizeValue did not return []any: %T", normalizeValue(in))
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	item, ok := out[0].(map[string]any)
	if !ok {
		t.Fatalf("item is not map[string]any: %T", out[0])
	}
	if item["a"] != 1 {
		t.Fatalf("item[\"a\"] = %v, want 1", item["a"])
	}
}

func TestPlainValueFlattensOrderedMap(t *testing.T) {
	in := orderedMap{
		{key: "a", value: orderedMap{{key: "b", value: 2}}},
	}
	out, ok := plainValue(in).(map[string]any)
	if !ok {
		t.Fatalf("plainValue did not return map[string]any: %T", plainValue(in))
	}
	nested, ok := out["a"].(map[string]any)
	if !ok {
		t.Fatalf("nested value is not map[string]any: %T", out["a"])
	}
	if nested["b"] != 2 {
		t.Fatalf("nested[\"b\"] = %v, want 2", nested["b"])
	}
}
