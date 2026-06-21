# Dependency Decisions

Role: Dependency Strategist

## Principles

- Prefer the Go standard library for filesystem, process, path, glob, JSON, logging, and testing behavior.
- Use third-party modules only for behavior that is part of Dotbot's user-facing surface and impractical to reproduce completely.
- Keep dependencies small, mature, and justified.

## Runtime Dependencies

### `github.com/spf13/cobra`

Purpose: CLI command and flag parsing.

Justification: The requested migration explicitly asks the CLI Engineer to implement Cobra commands. Cobra also gives stable help text, repeatable flags, validation hooks, and conventional exit handling.

### `gopkg.in/yaml.v3`

Purpose: YAML configuration parsing.

Justification: Dotbot configuration is primarily YAML. Go's standard library has JSON support but no YAML parser. `yaml.v3` preserves generic maps/sequences well enough for Dotbot's flexible directive schema and can decode into `interface{}` structures for parity.

Transitive dependency:

- `gopkg.in/check.v1` may appear indirectly through YAML's module graph. It is not used by application code.

### `github.com/titanous/json5`

Purpose: JSON5 configuration parsing for commented JSON-like dotfiles configs.

Justification: Go's standard `encoding/json` package intentionally accepts strict JSON only. JSON5 is a user-facing config format here, so a focused decoder keeps comment/trailing-comma support isolated to `internal/config`.

### `github.com/pelletier/go-toml/v2`

Purpose: TOML configuration parsing.

Justification: Go's standard library has no TOML decoder. `go-toml/v2` is a maintained TOML library with a simple `Unmarshal` API, and the parser is isolated behind the config parser registry.

## Standard Library Usage

- `encoding/json`: JSON config support.
- `os`, `io/fs`, `path/filepath`: filesystem operations.
- `os/exec`: shell execution.
- `context`: command timeout and cancellation boundary.
- `time`: backup suffixes and shell timeout.
- `runtime`: platform-sensitive behavior.
- `testing`, `bytes`, `strings`: tests and test doubles.

## Rejected Dependencies

- A plugin framework dependency: not needed for the built-in directives, and Python plugin loading cannot be faithfully implemented in-process in Go.
- A color library: ANSI color output is small enough to implement locally.
- A glob library: `filepath.Glob` plus small recursive `**` handling covers the migration surface without another module.
- A logging framework: Dotbot output levels are simple and behavior-specific.
- A HOCON parser: HOCON support was removed because the previous parser exposed map-backed objects, which prevented deterministic source-order execution across supported formats.
