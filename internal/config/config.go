package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Task map[string]any

type Parser interface {
	Parse(path string, data []byte) (any, error)
}

type ParserFunc func(path string, data []byte) (any, error)

func (f ParserFunc) Parse(path string, data []byte) (any, error) {
	return f(path, data)
}

type Registry struct {
	parsers map[string]Parser
}

func NewRegistry() *Registry {
	return &Registry{parsers: map[string]Parser{}}
}

func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register([]string{".yaml", ".yml"}, YAMLParser{})
	r.Register([]string{".json"}, JSONParser{})
	r.Register([]string{".json5"}, JSON5Parser{})
	r.Register([]string{".toml"}, TOMLParser{})
	r.Register([]string{".conf", ".hocon"}, HOCONParser{})
	return r
}

func (r *Registry) Register(extensions []string, parser Parser) {
	for _, ext := range extensions {
		r.parsers[normalizeExtension(ext)] = parser
	}
}

func (r *Registry) ParserFor(path string) (Parser, bool) {
	parser, ok := r.parsers[normalizeExtension(filepath.Ext(path))]
	return parser, ok
}

type Reader struct {
	registry *Registry
	readFile func(string) ([]byte, error)
}

func NewReader(registry *Registry) *Reader {
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Reader{
		registry: registry,
		readFile: os.ReadFile,
	}
}

func Read(paths []string) ([]Task, error) {
	return NewReader(nil).Read(paths)
}

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
		if m, isMap := raw.(map[string]any); isMap {
			list, ok = taskListFromMap(m)
		}
	}
	if !ok {
		return nil, fmt.Errorf("configuration file %q must be a list of tasks", path)
	}
	tasks := make([]Task, 0, len(list))
	for _, item := range list {
		m, ok := normalizeValue(item).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("configuration task in %q must be a mapping", path)
		}
		tasks = append(tasks, Task(m))
	}
	return tasks, nil
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

func normalizeExtension(ext string) string {
	if ext == "" {
		return ext
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return strings.ToLower(ext)
}
