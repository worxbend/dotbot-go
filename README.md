# dotbot-go

`dotbot-go` is a friendly Go implementation of the core Dotbot workflow for bootstrapping dotfiles. It reads a small YAML or JSON config, creates directories, links files into place, cleans broken symlinks, and runs optional setup commands.

The goal is to keep the familiar Dotbot behavior while producing a single Go binary you can place in your dotfiles repository or install locally.

## Supported Directives

- `defaults`
- `link`
- `create`
- `shell`
- `clean`

Python plugin loading is accepted at the CLI/config surface but is not executed by this Go port. Shell commands run through `$SHELL -c` on Unix and `cmd /C` on Windows. To avoid unbounded hangs, shell commands use a default execution timeout.

## Build

Please use `just` for the common project commands:

```bash
just build
```

That creates:

```text
bin/dotbot
```

To run formatting, module cleanup, tests, vet, and the build together:

```bash
just verify
```

To create Linux release binaries locally:

```bash
just package-linux
```

That writes Linux `amd64` and `arm64` archives plus checksums into `dist/`.

## Usage

Run against a config file:

```bash
./bin/dotbot -c install.conf.yaml
```

Preview changes without touching the filesystem:

```bash
./bin/dotbot -c install.conf.yaml --dry-run
```

Run from a specific dotfiles directory:

```bash
./bin/dotbot -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

Run only one directive:

```bash
./bin/dotbot -c install.conf.yaml --only link
```

Skip a directive:

```bash
./bin/dotbot -c install.conf.yaml --except shell
```

Show more detail:

```bash
./bin/dotbot -vv -c install.conf.yaml
```

Color is enabled automatically when output is connected to a terminal. Use `--force-color` to keep color in redirected output, or `--no-color` to disable ANSI output.

## Example Config

A complete example is available at [examples/install.conf.yaml](examples/install.conf.yaml).

Here is a small version:

```yaml
- defaults:
    link:
      create: true
      relink: true

- clean:
    - "~"

- create:
    - "~/.vim/undo-history"

- link:
    ~/.tmux.conf: tmux.conf
    ~/.vimrc:
      path: vimrc
      backup: true
    ~/.config/starship.toml:
      path: starship.toml

- shell:
    - [git submodule update --init --recursive, Installing submodules]
```

If a destination already exists as a normal file or directory, `dotbot-go` will not overwrite it by default. Use `backup: true` when you want the existing path renamed before linking, or `force: true` only when removing the existing path is intentional.

## Development

```bash
just --list
just verify
```

## Releases

GitHub Actions tests and builds the project on pushes and pull requests. When you push a tag that starts with `v`, for example `v0.1.0`, the workflow publishes a GitHub Release with:

- `dotbot-linux-amd64.tar.gz`
- `dotbot-linux-amd64.tar.gz.sha256`
- `dotbot-linux-arm64.tar.gz`
- `dotbot-linux-arm64.tar.gz.sha256`

Example:

```bash
git tag v0.1.0
git push origin v0.1.0
```
