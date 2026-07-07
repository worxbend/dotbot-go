# Troubleshooting Guide

Start every debugging session by inspecting the plan, then run a verbose dry run:

```bash
dotbot-go plan -c install.conf.yaml
```

```bash
dotbot-go -c install.conf.yaml --dry-run -vv
```

These commands show what `dotbot-go` plans to do without changing your filesystem.

## `unsupported config file format`

The file extension is not supported.

Supported extensions:

- `.yaml`
- `.yml`
- `.json`
- `.json5`
- `.toml`

Fix:

```bash
mv install.yaml.txt install.yaml
```

## `configuration file must be a list of tasks`

Your config shape is wrong.

For YAML, JSON, and JSON5, the top level should usually be a list:

```yaml
- link:
    ~/.vimrc: vimrc
```

For TOML, put the list under `tasks`:

```toml
tasks = [
  { link = { "~/.vimrc" = "vimrc" } },
]
```

## Link Target Does Not Exist

Example output:

```text
Nonexistent target ~/.vimrc -> vimrc
```

This usually means the source file is missing from your dotfiles repository.

Check:

```bash
ls -la ~/.dotfiles/vimrc
```

If the file is somewhere else, update the `path`:

```yaml
- link:
    ~/.vimrc:
      path: editors/vimrc
```

## Existing File Blocks A Link

`dotbot-go` does not overwrite normal files by default.

If you want to keep the existing file, use:

```yaml
- link:
    ~/.vimrc:
      path: vimrc
      backup: true
```

If you intentionally want to remove the existing destination, use:

```yaml
- link:
    ~/.vimrc:
      path: vimrc
      force: true
```

Use `force` carefully.

## Shell Command Fails

Run with `-vv`:

```bash
dotbot-go -c install.conf.yaml -vv
```

At verbosity level `-vv`, shell command stdout and stderr are shown.

You can also skip shell commands while testing links:

```bash
dotbot-go -c install.conf.yaml --except shell --dry-run
```

## Colors Look Wrong

Disable color:

```bash
dotbot-go -c install.conf.yaml --no-color
```

Force color when output is redirected:

```bash
dotbot-go -c install.conf.yaml --force-color
```

## I Am Afraid To Run It

Use this workflow:

```bash
dotbot-go -c install.conf.yaml --dry-run
dotbot-go -c install.conf.yaml --only create
dotbot-go -c install.conf.yaml --only link --dry-run
dotbot-go -c install.conf.yaml --only link
```

This applies the config in smaller steps.
