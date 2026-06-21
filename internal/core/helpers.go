package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
)

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func asList(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []string:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = item
		}
		return out, true
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Slice {
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = rv.Index(i).Interface()
			}
			return out, true
		}
		return nil, false
	}
}

func asString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case fmt.Stringer:
		return t.String(), true
	default:
		return "", false
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func boolValue(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return fallback
}

func stringValue(m map[string]any, key string, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := asString(v); ok {
			return s
		}
	}
	return fallback
}

func stringSlice(v any) []string {
	list, ok := asList(v)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := asString(item); ok {
			out = append(out, s)
		}
	}
	return out
}

func isStringList(v any) bool {
	list, ok := asList(v)
	if !ok {
		return false
	}
	for _, item := range list {
		if _, ok := asString(item); !ok {
			return false
		}
	}
	return true
}

func defaultTarget(linkName string, target any) string {
	if target == nil {
		base := filepath.Base(linkName)
		if len(base) > 0 && base[0] == '.' {
			return base[1:]
		}
		return base
	}
	if s, ok := asString(target); ok {
		return s
	}
	return ""
}

func parseMode(v any, fallback os.FileMode) os.FileMode {
	if s, ok := asString(v); ok {
		i, err := strconv.ParseUint(s, 8, 32)
		if err == nil {
			return os.FileMode(i)
		}
	}
	return fileMode(v, fallback)
}
