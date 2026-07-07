# Migration Analysis

Role: Repository Analyst

## Source Project

The source repository is the Python Dotbot project in `dotbot/`. It is a command line tool that reads YAML or JSON task lists and applies dotfile bootstrap directives.

## Python Structure

- `src/dotbot/cli.py`: argument parsing, color/verbosity setup, plugin loading, config reading, dispatcher construction, and process exit codes.
- `src/dotbot/config.py`: YAML/JSON config reader. Multiple config files are concatenated into one task list.
- `src/dotbot/dispatcher.py`: task dispatcher. It applies `defaults`, loads config-declared plugins, and routes each directive to plugins.
- `src/dotbot/context.py`: base directory, defaults, options, and plugin metadata.
- `src/dotbot/plugin.py`: plugin interface.
- `src/dotbot/plugins/link.py`: symlink/hardlink behavior, glob handling, backups, force/relink, target existence checks, relative links, and dry-run.
- `src/dotbot/plugins/create.py`: directory creation with optional mode.
- `src/dotbot/plugins/shell.py`: shell command execution with optional stdin/stdout/stderr and quiet messages.
- `src/dotbot/plugins/clean.py`: broken symlink cleanup.
- `src/dotbot/messenger/`: singleton logger with levels and ANSI color.
- `src/dotbot/util/`: shell execution, path normalization, dynamic Python plugin loading.
- `bin/dotbot`: shim entry point for repository/submodule usage.
- `tools/git-submodule/` and `tools/hg-subrepo/`: installer shims.

## Entry Points

- Python package script: `dotbot = dotbot.cli:main`.
- Repository shim: `bin/dotbot`.

## CLI Surface

The Python CLI supports:

- `-Q`, `--super-quiet`: deprecated quiet mode.
- `-q`, `--quiet`: suppress most output.
- `-v`, `--verbose`: repeatable; `-vv` also enables shell stdout/stderr.
- `-d`, `--base-directory`: base directory for config-relative targets.
- `-c`, `--config-file`: one or more config files.
- `-p`, `--plugin`: Python plugin path.
- `--disable-built-in-plugins`
- `--plugin-dir`: deprecated plugin directory.
- `--only`: run only selected directives.
- `--except`: skip selected directives.
- `-n`, `--dry-run`
- `--force-color`
- `--no-color`
- `--version`
- `-x`, `--exit-on-failure`

## Config Format

Configuration files are YAML or JSON arrays. Each item is a task object whose keys are directives. Task order is preserved, while directive order inside a map is not a documented contract.

Built-in directives:

- `defaults`
- `link`
- `create`
- `shell`
- `clean`
- `plugins`

## Dependencies

Runtime Python dependency:

- `PyYAML>=6.0.1,<7`

Development/test dependencies are managed by Hatch and pytest tooling in `pyproject.toml`.

## Tests

The Python suite includes tests for CLI, config, link, clean, create, shell, plugin behavior, no-op configs, and the shim. The test fixture wraps filesystem-mutating functions to keep tests inside a temp root.

Baseline verification observed during migration:

```text
PYTHONPATH=src python3 -m pytest
148 passed, 2 skipped
```

## Side Effects

Dotbot intentionally performs side effects:

- Creates directories.
- Creates symlinks and hardlinks.
- Removes links or files/directories when `relink` or `force` is enabled.
- Renames existing files/directories when `backup` is enabled.
- Executes arbitrary shell commands.
- Loads arbitrary Python plugin files.
- Cleans broken symlinks.
- Changes working directory to the base directory before dispatch.

The Go migration preserves supported built-in directive behavior. Python plugin loading is intentionally omitted from the Go port.
