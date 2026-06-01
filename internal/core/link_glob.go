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

func createGlobResults(ctx *Context, pattern string, exclude []string) []string {
	include := glob(ctx, pattern)
	excluded := map[string]bool{}
	for _, ex := range exclude {
		for _, item := range glob(ctx, ex) {
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
	return out
}

func glob(ctx *Context, pattern string) []string {
	matches := []string{}
	if strings.Contains(pattern, "**") {
		root := pattern[:strings.Index(pattern, "**")]
		root = strings.TrimRight(root, string(filepath.Separator))
		if root == "" {
			root = "."
		}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			ok, _ := doublestarMatch(pattern, path)
			if ok {
				if !strings.HasSuffix(pattern, string(filepath.Separator)) && d.IsDir() {
					return nil
				}
				matches = append(matches, filepath.Clean(path))
			}
			return nil
		}); err != nil {
			ctx.Log.Debug(err.Error())
		}
		return matches
	}
	found, err := filepath.Glob(pattern)
	if err != nil {
		ctx.Log.Debug(err.Error())
		return nil
	}
	for _, item := range found {
		matches = append(matches, filepath.Clean(item))
	}
	return matches
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
