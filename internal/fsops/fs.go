package fsops

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileInfo is the filesystem metadata type returned by FS.
type FileInfo = os.FileInfo

// FS is the filesystem surface used by directive handlers.
//
// Implementations should match the behavior of the corresponding os and filepath
// operations closely enough that handlers can be tested against fakes.
type FS interface {
	Abs(string) (string, error)
	Chmod(string, os.FileMode) error
	Exists(string) bool
	Lexists(string) bool
	IsDir(string) bool
	IsSymlink(string) bool
	ListDir(string) ([]string, error)
	MkdirAll(string, os.FileMode) error
	// MkdirTemp reserves a new private directory with an exclusive generated name.
	MkdirTemp(string, string) (string, error)
	// Glob and WalkDir expose traversal through the same adapter as mutation.
	Glob(string) ([]string, error)
	WalkDir(string, fs.WalkDirFunc) error
	Readlink(string) (string, error)
	Realpath(string) (string, error)
	Remove(string) error
	RemoveAll(string) error
	Rename(string, string) error
	SameFile(string, string) (bool, error)
	Stat(string) (FileInfo, error)
	Symlink(string, string) error
	Link(string, string) error
}

// OSFS implements FS with the local operating system filesystem.
type OSFS struct{}

func (OSFS) Abs(path string) (string, error)           { return filepath.Abs(path) }
func (OSFS) Chmod(path string, mode os.FileMode) error { return os.Chmod(path, mode) }
func (OSFS) Exists(path string) bool                   { _, err := os.Stat(path); return err == nil }
func (OSFS) Lexists(path string) bool                  { _, err := os.Lstat(path); return err == nil }
func (OSFS) IsDir(path string) bool                    { info, err := os.Stat(path); return err == nil && info.IsDir() }
func (OSFS) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
func (OSFS) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}
func (OSFS) MkdirAll(path string, mode os.FileMode) error  { return os.MkdirAll(path, mode) }
func (OSFS) MkdirTemp(dir, pattern string) (string, error) { return os.MkdirTemp(dir, pattern) }
func (OSFS) Glob(pattern string) ([]string, error)         { return filepath.Glob(pattern) }
func (OSFS) WalkDir(root string, fn fs.WalkDirFunc) error  { return filepath.WalkDir(root, fn) }
func (OSFS) Readlink(path string) (string, error)          { return os.Readlink(path) }
func (OSFS) Realpath(path string) (string, error)          { return filepath.EvalSymlinks(path) }
func (OSFS) Remove(path string) error                      { return os.Remove(path) }
func (OSFS) RemoveAll(path string) error                   { return os.RemoveAll(path) }
func (OSFS) Rename(oldpath, newpath string) error          { return os.Rename(oldpath, newpath) }
func (OSFS) Stat(path string) (FileInfo, error)            { return os.Stat(path) }
func (OSFS) Symlink(oldname, newname string) error         { return os.Symlink(oldname, newname) }
func (OSFS) Link(oldname, newname string) error            { return os.Link(oldname, newname) }

func (OSFS) SameFile(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ai, bi), nil
}
