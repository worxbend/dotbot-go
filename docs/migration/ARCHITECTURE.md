# Architecture

Role: Go Architect

## Target Layout

```text
dotbot-go/
  cmd/dotbot-go/main.go
  internal/cli
  internal/config
  internal/core
  internal/expand
  internal/fsops
  internal/log
  internal/shell
  docs/migration
```

## Package Ownership

- `cmd/dotbot-go`: executable entry point only. No business logic belongs here.
- `internal/cli`: Cobra command, flags, validation, version output, and process-level orchestration.
- `internal/config`: reads YAML/JSON config files into ordered task data.
- `internal/core`: dispatcher, context, directive handlers, and behavior-compatible business logic.
- `internal/expand`: shell-like path expansion helpers for `~` and environment variables.
- `internal/fsops`: filesystem abstraction and OS-backed implementation for test doubles.
- `internal/log`: Dotbot-like leveled logger with optional color.
- `internal/shell`: shell command runner with `context.Context`, timeout support, and output controls.

## Core Interfaces

- `core.Handler`: directive handler interface with `CanHandle`, `Handle`, and dry-run support.
- `fsops.FS`: filesystem interface used by core handlers.
- `shell.Runner`: command execution abstraction used by `shell` and conditional `link.if`.

These interfaces keep side effects replaceable in tests and prevent business logic from moving into `main.go`.

## Error Strategy

The Python implementation usually logs per-item failures and returns aggregate success. The Go port keeps that model:

- Validation/config read errors return explicit errors.
- Directive handlers return `(bool, error)`.
- Per-item operational failures are logged and return `false`.
- Unexpected programming or parsing errors are returned as errors and never swallowed.
- `--exit-on-failure` stops after the first failed directive.

## Compatibility Decisions

- YAML and JSON task arrays are supported.
- Multiple config files are concatenated in CLI order.
- Base directory defaults to the first config file's directory.
- Built-ins supported: `defaults`, `link`, `create`, `shell`, `clean`.
- Python plugin loading is not supported. `plugins`, `--plugin`, `--plugin-dir`, and built-in plugin disabling are intentionally omitted.
- Shell commands run through `$SHELL -c` on Unix and `cmd /C` on Windows, matching the Python intent.
- Shell commands and `link.if` conditions have a 10-minute timeout per command, documented in `README.md` and the user guide. The CLI and config provide no timeout override.
- Dry run does not execute shell commands or evaluate `link.if` conditions. It reports each conditional link's condition and destination, then skips its filesystem preview; the outcome remains unknown until an actual run.

## Filesystem Safety

Filesystem operations are centralized in `fsops` and routed through dry-run checks. Destructive operations are limited to cases matching Dotbot semantics:

- `relink` removes only existing symlinks with mismatched targets.
- `force` may remove files/directories at the destination.
- `backup` reserves a private, unique `<destination>.dotbot-backup.<timestamp>-<random>` directory and moves the original file or directory into it under its basename. Existing backups are preserved; reservation or move failures leave the original destination in place.
- `clean` removes broken symlinks in selected directories only when their targets are inside the base directory, unless `force` is enabled. It checks immediate entries unless `recursive` is enabled and does not traverse directory symlinks.
