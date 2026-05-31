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
- `plugins`, `--plugin`, and `--plugin-dir` are accepted but Go cannot execute Python plugins. A requested plugin path is reported as unsupported and marks the run failed.
- Built-in plugin disabling is supported, which can cause built-in directives to be unhandled.
- Shell commands run through `$SHELL -c` on Unix and `cmd /C` on Windows, matching the Python intent.
- Shell commands use a default timeout to avoid unbounded process hangs. The timeout is a Go safety improvement and is documented in `README.md`.

## Filesystem Safety

Filesystem operations are centralized in `fsops` and routed through dry-run checks. Destructive operations are limited to cases matching Dotbot semantics:

- `relink` removes only existing symlinks with mismatched targets.
- `force` may remove files/directories at the destination.
- `backup` renames existing non-link destinations before linking.
- `clean` only removes broken symlinks under the base directory unless `force` is enabled.
