package config

import (
	"fmt"
	"sort"

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
	var currentTable []string

	for parser.NextExpression() {
		expr := parser.Expression()
		switch expr.Kind {
		case tomlunstable.KeyValue:
			keys := tomlKeyParts(expr.Key())
			value, err := tomlNodeValue(&parser, expr.Value())
			if err != nil {
				return nil, err
			}
			if len(currentTable) == 0 && len(keys) == 1 && isTOMLTaskListKey(keys[0]) {
				list, ok := value.([]any)
				if ok {
					taskLists[keys[0]] = list
				}
				currentTaskList = ""
				currentTaskIndex = -1
				continue
			}
			if currentTaskList == "" || currentTaskIndex < 0 || len(currentTable) > 1 {
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
			currentTable = keys
			if len(keys) == 1 && isTOMLTaskListKey(keys[0]) {
				taskLists[keys[0]] = append(taskLists[keys[0]], orderedMap{})
				currentTaskList = keys[0]
				currentTaskIndex = len(taskLists[keys[0]]) - 1
				continue
			}
			currentTaskList, currentTaskIndex = tomlNestedTask(taskLists, keys)
		case tomlunstable.Table:
			currentTable = tomlKeyParts(expr.Key())
			currentTaskList, currentTaskIndex = tomlNestedTask(taskLists, currentTable)
		}
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	return taskLists, nil
}

func tomlNestedTask(taskLists map[string][]any, keys []string) (string, int) {
	if len(keys) < 2 || !isTOMLTaskListKey(keys[0]) || len(taskLists[keys[0]]) == 0 {
		return "", -1
	}
	index := len(taskLists[keys[0]]) - 1
	task, _ := taskLists[keys[0]][index].(orderedMap)
	for _, pair := range task {
		if pair.key == keys[1] {
			return keys[0], index
		}
	}
	taskLists[keys[0]][index] = append(task, orderedPair{key: keys[1]})
	return keys[0], index
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
		decodedTasks, ok := normalizeValue(root[key]).([]any)
		if !ok {
			continue
		}
		orderedTasks := make([]any, 0, len(decodedTasks))
		for i, decoded := range decodedTasks {
			values, ok := decoded.(map[string]any)
			if !ok {
				orderedTasks = append(orderedTasks, decoded)
				continue
			}
			ordered := orderedMap{}
			seen := map[string]bool{}
			if i < len(tasks) {
				pairs, _ := tasks[i].(orderedMap)
				for _, pair := range pairs {
					if value, exists := values[pair.key]; exists && !seen[pair.key] {
						ordered = append(ordered, orderedPair{key: pair.key, value: value})
						seen[pair.key] = true
					}
				}
			}
			// Preserve all decoded values even if an ordering expression is unfamiliar.
			remaining := []string{}
			for name := range values {
				if !seen[name] {
					remaining = append(remaining, name)
				}
			}
			sort.Strings(remaining)
			for _, name := range remaining {
				ordered = append(ordered, orderedPair{key: name, value: values[name]})
			}
			orderedTasks = append(orderedTasks, ordered)
		}
		root[key] = orderedTasks
	}
}

func tomlNodeValue(parser *tomlunstable.Parser, node *tomlunstable.Node) (any, error) {
	if node == nil || !node.Valid() {
		return nil, fmt.Errorf("invalid toml value")
	}

	switch node.Kind {
	case tomlunstable.Array:
		out := []any{}
		children := node.Children()
		for children.Next() {
			value, err := tomlNodeValue(parser, children.Node())
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
			value, err := tomlNodeValue(parser, child.Value())
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
		return tomlScalarValue(parser, node)
	default:
		return nil, fmt.Errorf("unsupported toml value kind %s", node.Kind)
	}
}

func tomlScalarValue(parser *tomlunstable.Parser, node *tomlunstable.Node) (any, error) {
	var raw map[string]any
	doc := []byte("value = ")
	if node.Kind == tomlunstable.String {
		doc = append(doc, parser.Raw(node.Raw)...)
	} else {
		doc = append(doc, node.Data...)
	}
	doc = append(doc, '\n')
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
