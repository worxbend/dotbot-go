package expand

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// User expands a leading "~" to the current user's home directory.
func User(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(path, `~\`)) {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// Path expands environment variables, normalizes path separators, and expands a
// leading "~".
func Path(path string) string {
	return User(os.ExpandEnv(NormSlash(path)))
}

// Abs returns an absolute path after applying Path expansion.
func Abs(path string) string {
	abs, err := filepath.Abs(Path(path))
	if err != nil {
		return Path(path)
	}
	return abs
}

// NormSlash converts forward slashes to platform separators on Windows.
func NormSlash(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", `\`)
	}
	return path
}
