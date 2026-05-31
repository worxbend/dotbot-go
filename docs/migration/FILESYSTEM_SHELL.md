# Filesystem and Shell Engineer Report

## Implemented

- `internal/fsops.FS` interface with an OS-backed implementation.
- Centralized symlink, hardlink, rename, remove, mkdir, stat, and realpath calls.
- Dry-run checks before filesystem mutation.
- `internal/shell.Runner` interface with an OS command runner.
- Shell execution through `$SHELL -c` on Unix and `cmd /C` on Windows.
- `context.Context` and timeout support for shell commands.

## Safety Notes

- `relink` removes mismatched symlinks.
- `force` is required before existing regular files or directories are removed.
- `backup` renames existing non-link paths before linking.
- `clean` removes broken symlinks under the base directory unless forced.

## Testability

The core accepts filesystem and shell abstractions through context, and integration tests currently use temporary directories with real OS filesystem behavior.
