package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func hasGlobChars(path string) bool {
	return strings.ContainsAny(path, "?*[")
}

func createGlobResults(pattern string, exclude []string) ([]string, error) {
	include, err := glob(pattern)
	if err != nil {
		return nil, err
	}
	excluded := map[string]bool{}
	for _, ex := range exclude {
		items, err := glob(ex)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			excluded[item] = true
		}
	}
	out := []string{}
	for _, item := range include {
		if !excluded[item] {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out, nil
}

func glob(pattern string) ([]string, error) {
	matches := []string{}
	if strings.Contains(pattern, "**") {
		root := pattern[:strings.Index(pattern, "**")]
		root = strings.TrimRight(root, string(filepath.Separator))
		if root == "" {
			root = "."
		}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			ok, err := doublestarMatch(pattern, path)
			if err != nil {
				return err
			}
			if ok {
				if !strings.HasSuffix(pattern, string(filepath.Separator)) && d.IsDir() {
					return nil
				}
				matches = append(matches, filepath.Clean(path))
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return matches, nil
	}
	found, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, item := range found {
		matches = append(matches, filepath.Clean(item))
	}
	return matches, nil
}

func doublestarMatch(pattern, path string) (bool, error) {
	if !strings.Contains(pattern, "**") {
		return filepath.Match(pattern, path)
	}
	parts := strings.Split(pattern, "**")
	prefix := filepath.Clean(strings.TrimRight(parts[0], string(filepath.Separator)))
	suffix := ""
	if len(parts) > 1 {
		suffix = strings.TrimLeft(parts[1], string(filepath.Separator))
	}
	if prefix != "." && prefix != "" {
		cleanPath := filepath.Clean(path)
		if cleanPath != prefix && !strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return false, nil
		}
	}
	if suffix == "" {
		return true, nil
	}
	return filepath.Match(suffix, filepath.Base(path))
}

func globLinkItem(pattern, item string) string {
	dir := filepath.Dir(commonPrefix(pattern, item))
	if dir == "." || dir == string(filepath.Separator) || dir == "" {
		return item
	}
	rel := strings.TrimPrefix(item, dir+string(filepath.Separator))
	return rel
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}
