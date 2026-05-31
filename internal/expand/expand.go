package expand

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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

func Path(path string) string {
	return User(os.ExpandEnv(NormSlash(path)))
}

func Abs(path string) string {
	abs, err := filepath.Abs(Path(path))
	if err != nil {
		return Path(path)
	}
	return abs
}

func NormSlash(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", `\`)
	}
	return path
}
