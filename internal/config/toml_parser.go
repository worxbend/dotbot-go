package config

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
	tomlunstable "github.com/pelletier/go-toml/v2/unstable"
)

// TOMLParser parses TOML configuration files and preserves task directive order.
type TOMLParser struct{}

// Parse decodes TOML into normalized raw values.
func (TOMLParser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	orderedTasks, err := tomlOrderedTaskLists(data)
	if err != nil {
		return nil, err
	}
	applyTOMLTaskOrder(raw, orderedTasks)
	return raw, nil
}

func tomlOrderedTaskLists(data []byte) (map[string][]any, error) {
	parser := tomlunstable.Parser{}
	parser.Reset(data)

	taskLists := map[string][]any{}
	currentTaskList := ""
	currentTaskIndex := -1

	for parser.NextExpression() {
		expr := parser.Expression()
		switch expr.Kind {
		case tomlunstable.KeyValue:
			keys := tomlKeyParts(expr.Key())
			value, err := tomlNodeValue(expr.Value())
			if err != nil {
				return nil, err
			}
			if len(keys) == 1 && isTOMLTaskListKey(keys[0]) {
				list, ok := value.([]any)
				if ok {
					taskLists[keys[0]] = list
				}
				currentTaskList = ""
				currentTaskIndex = -1
				continue
			}
			if currentTaskList == "" || currentTaskIndex < 0 {
				continue
			}
			task, ok := taskLists[currentTaskList][currentTaskIndex].(orderedMap)
			if !ok {
				task = orderedMap{}
			}
			task = insertTOMLKeyValue(task, keys, value)
			taskLists[currentTaskList][currentTaskIndex] = task
		case tomlunstable.ArrayTable:
			keys := tomlKeyParts(expr.Key())
			if len(keys) == 1 && isTOMLTaskListKey(keys[0]) {
				taskLists[keys[0]] = append(taskLists[keys[0]], orderedMap{})
				currentTaskList = keys[0]
				currentTaskIndex = len(taskLists[keys[0]]) - 1
				continue
			}
			currentTaskList = ""
			currentTaskIndex = -1
		case tomlunstable.Table:
			currentTaskList = ""
			currentTaskIndex = -1
		}
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	return taskLists, nil
}

func applyTOMLTaskOrder(raw any, taskLists map[string][]any) {
	if len(taskLists) == 0 {
		return
	}
	root, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for _, key := range []string{"tasks", "task"} {
		tasks, ok := taskLists[key]
		if !ok {
			continue
		}
		if _, exists := root[key]; exists {
			root[key] = tasks
		}
	}
}

func tomlNodeValue(node *tomlunstable.Node) (any, error) {
	if node == nil || !node.Valid() {
		return nil, fmt.Errorf("invalid toml value")
	}

	switch node.Kind {
	case tomlunstable.Array:
		out := []any{}
		children := node.Children()
		for children.Next() {
			value, err := tomlNodeValue(children.Node())
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	case tomlunstable.InlineTable:
		out := orderedMap{}
		children := node.Children()
		for children.Next() {
			child := children.Node()
			value, err := tomlNodeValue(child.Value())
			if err != nil {
				return nil, err
			}
			out = insertTOMLKeyValue(out, tomlKeyParts(child.Key()), value)
		}
		return out, nil
	case tomlunstable.String,
		tomlunstable.Bool,
		tomlunstable.Integer,
		tomlunstable.Float,
		tomlunstable.DateTime,
		tomlunstable.LocalDate,
		tomlunstable.LocalTime,
		tomlunstable.LocalDateTime:
		return tomlScalarValue(node)
	default:
		return nil, fmt.Errorf("unsupported toml value kind %s", node.Kind)
	}
}

func tomlScalarValue(node *tomlunstable.Node) (any, error) {
	if node.Kind == tomlunstable.String {
		return string(node.Data), nil
	}
	var raw map[string]any
	doc := []byte("value = " + string(node.Data) + "\n")
	if err := toml.Unmarshal(doc, &raw); err != nil {
		return nil, err
	}
	return raw["value"], nil
}

func tomlKeyParts(keys tomlunstable.Iterator) []string {
	var parts []string
	for keys.Next() {
		parts = append(parts, string(keys.Node().Data))
	}
	return parts
}

func insertTOMLKeyValue(m orderedMap, keys []string, value any) orderedMap {
	if len(keys) == 0 {
		return m
	}
	if len(keys) == 1 {
		return append(m, orderedPair{key: keys[0], value: value})
	}

	for i := range m {
		if m[i].key != keys[0] {
			continue
		}
		child, _ := m[i].value.(orderedMap)
		m[i].value = insertTOMLKeyValue(child, keys[1:], value)
		return m
	}

	child := insertTOMLKeyValue(orderedMap{}, keys[1:], value)
	return append(m, orderedPair{key: keys[0], value: child})
}

func isTOMLTaskListKey(key string) bool {
	return key == "tasks" || key == "task"
}

var _ Parser = TOMLParser{}
