package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Task is one configuration task, keyed by directive name.
//
// Tasks created by ordered parsers carry hidden directive-order metadata so
// planning and dispatch preserve the order written by the user.
type Task map[string]any

// Action is one directive entry from a configuration task.
type Action struct {
	// Directive names the built-in directive, such as link, create, shell, or clean.
	Directive string
	// Data is the directive payload after parser-specific values have been normalized.
	Data any
}

const taskActionOrderKey = "\x00dotbot-go:action-order"

type taskActionOrder []string

type orderedPair struct {
	key   string
	value any
}

type orderedMap []orderedPair

// NewTask builds a task with explicit directive order.
func NewTask(actions ...Action) Task {
	task := Task{}
	order := make(taskActionOrder, 0, len(actions))
	seen := map[string]bool{}
	for _, action := range actions {
		if action.Directive == "" {
			continue
		}
		if !seen[action.Directive] {
			order = append(order, action.Directive)
			seen[action.Directive] = true
		}
		task[action.Directive] = action.Data
	}
	if len(order) > 0 {
		task[taskActionOrderKey] = order
	}
	return task
}

// Actions returns task directives in source order when available.
func (t Task) Actions() []Action {
	if order, ok := t[taskActionOrderKey].(taskActionOrder); ok {
		actions := make([]Action, 0, len(order))
		for _, directive := range order {
			data, ok := t[directive]
			if !ok {
				continue
			}
			actions = append(actions, Action{Directive: directive, Data: data})
		}
		return actions
	}

	keys := make([]string, 0, len(t))
	for key := range t {
		if key == taskActionOrderKey {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	actions := make([]Action, 0, len(keys))
	for _, key := range keys {
		actions = append(actions, Action{Directive: key, Data: t[key]})
	}
	return actions
}

// Parser converts one config file into the raw representation consumed by Reader.
type Parser interface {
	// Parse decodes data from path while preserving directive order when possible.
	Parse(path string, data []byte) (any, error)
}

// ParserFunc adapts a function to the Parser interface.
type ParserFunc func(path string, data []byte) (any, error)

// Parse calls f(path, data).
func (f ParserFunc) Parse(path string, data []byte) (any, error) {
	return f(path, data)
}

// Registry maps file extensions to parsers.
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates an empty parser registry.
func NewRegistry() *Registry {
	return &Registry{parsers: map[string]Parser{}}
}

// DefaultRegistry creates a registry for YAML, JSON, JSON5, and TOML configs.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register([]string{".yaml", ".yml"}, YAMLParser{})
	r.Register([]string{".json"}, JSONParser{})
	r.Register([]string{".json5"}, JSON5Parser{})
	r.Register([]string{".toml"}, TOMLParser{})
	return r
}

// Register associates each extension with parser.
func (r *Registry) Register(extensions []string, parser Parser) {
	for _, ext := range extensions {
		r.parsers[normalizeExtension(ext)] = parser
	}
}

// ParserFor returns the parser registered for path's extension.
func (r *Registry) ParserFor(path string) (Parser, bool) {
	parser, ok := r.parsers[normalizeExtension(filepath.Ext(path))]
	return parser, ok
}

// Reader reads config files through a registry.
type Reader struct {
	registry *Registry
	readFile func(string) ([]byte, error)
}

// NewReader creates a Reader, using DefaultRegistry when registry is nil.
func NewReader(registry *Registry) *Reader {
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Reader{
		registry: registry,
		readFile: os.ReadFile,
	}
}

// Read reads all paths with the default registry.
func Read(paths []string) ([]Task, error) {
	return NewReader(nil).Read(paths)
}

// Read reads all paths in order and concatenates their tasks.
func (r *Reader) Read(paths []string) ([]Task, error) {
	tasks := []Task{}
	for _, path := range paths {
		fileTasks, err := r.readOne(path)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, fileTasks...)
	}
	return tasks, nil
}

func (r *Reader) readOne(path string) ([]Task, error) {
	data, err := r.readFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %q: %w", path, err)
	}
	parser, ok := r.registry.ParserFor(path)
	if !ok {
		return nil, fmt.Errorf("unsupported config file format %q", filepath.Ext(path))
	}
	raw, err := parser.Parse(path, data)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %q: %w", path, err)
	}
	return tasksFromRaw(path, raw)
}

func tasksFromRaw(path string, raw any) ([]Task, error) {
	raw = normalizeValue(raw)
	if raw == nil {
		return []Task{}, nil
	}
	list, ok := raw.([]any)
	if !ok {
		if m, isMap := raw.(orderedMap); isMap {
			list, ok = taskListFromOrderedMap(m)
		} else if m, isMap := raw.(map[string]any); isMap {
			list, ok = taskListFromMap(m)
		}
	}
	if !ok {
		return nil, fmt.Errorf("configuration file %q must be a list of tasks", path)
	}
	tasks := make([]Task, 0, len(list))
	for _, item := range list {
		item = normalizeValue(item)
		switch m := item.(type) {
		case orderedMap:
			tasks = append(tasks, taskFromOrderedMap(m))
		case map[string]any:
			tasks = append(tasks, Task(m))
		default:
			return nil, fmt.Errorf("configuration task in %q must be a mapping", path)
		}
	}
	return tasks, nil
}

func taskFromOrderedMap(m orderedMap) Task {
	actions := make([]Action, 0, len(m))
	for _, pair := range m {
		actions = append(actions, Action{
			Directive: pair.key,
			Data:      plainValue(pair.value),
		})
	}
	return NewTask(actions...)
}

func taskListFromOrderedMap(m orderedMap) ([]any, bool) {
	for _, key := range []string{"tasks", "task"} {
		if value, ok := orderedMapValue(m, key); ok {
			if list, ok := normalizeValue(value).([]any); ok {
				return list, true
			}
		}
	}
	return nil, false
}

func orderedMapValue(m orderedMap, key string) (any, bool) {
	for _, pair := range m {
		if pair.key == key {
			return pair.value, true
		}
	}
	return nil, false
}

func taskListFromMap(m map[string]any) ([]any, bool) {
	for _, key := range []string{"tasks", "task"} {
		if list, ok := normalizeValue(m[key]).([]any); ok {
			return list, true
		}
	}
	return nil, false
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case orderedMap:
		out := make(orderedMap, 0, len(t))
		for _, pair := range t {
			out = append(out, orderedPair{
				key:   pair.key,
				value: normalizeValue(pair.value),
			})
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, normalizeValue(item))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, normalizeValue(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for key, value := range t {
			out[key] = normalizeValue(value)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for key, value := range t {
			out[fmt.Sprint(key)] = normalizeValue(value)
		}
		return out
	default:
		return v
	}
}

func plainValue(v any) any {
	switch t := v.(type) {
	case orderedMap:
		out := make(map[string]any, len(t))
		for _, pair := range t {
			out[pair.key] = plainValue(pair.value)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, plainValue(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for key, value := range t {
			out[key] = plainValue(value)
		}
		return out
	default:
		return v
	}
}

func normalizeExtension(ext string) string {
	if ext == "" {
		return ext
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return strings.ToLower(ext)
}
