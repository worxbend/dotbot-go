# Filesystem and Shell Engineer Report

## Implemented

- `internal/fsops.FS` interface with an OS-backed implementation.
- Centralized symlink, hardlink, rename, remove, mkdir, stat, and realpath calls.
- Dry-run checks before filesystem mutation.
- `internal/shell.Runner` interface with an OS command runner.
- Shell execution through `$SHELL -c` on Unix and `cmd /C` on Windows.
- `context.Context` and a 10-minute timeout per shell command or `link.if` condition. The CLI and config provide no timeout override.
- Dry run does not execute shell commands or evaluate `link.if` conditions. It reports each conditional link's condition and destination, then skips its filesystem preview.

## Safety Notes

- `relink` removes mismatched symlinks.
- `force` is required before existing regular files or directories are removed.
- `backup` reserves a private, unique `<destination>.dotbot-backup.<timestamp>-<random>` directory and moves the original file or directory into it under its basename. Existing backups are preserved; reservation or move failures leave the original destination in place.
- `clean` removes broken symlinks in selected directories whose targets are inside the base directory unless forced. Cleanup is nonrecursive by default and never traverses directory symlinks.

## Testability

The core accepts filesystem and shell abstractions through context, and integration tests currently use temporary directories with real OS filesystem behavior.
