# dotbot-go

`dotbot-go` is a small command-line tool that helps you set up a computer from a dotfiles repository.

You write one config file that says:

- create these folders
- link these files into my home directory
- remove broken old symlinks
- run these setup commands

Then you run `dotbot-go`, preview the changes, and apply them.

This project is a Go implementation of the core [Dotbot](https://github.com/anishathalye/dotbot) workflow. It keeps the familiar config style while shipping as a single binary.

## When To Use It

Use `dotbot-go` when you keep configuration files in a Git repository and want to install them on a new machine or keep multiple machines in sync.

Common examples:

- link `~/.vimrc` to `vimrc` in your dotfiles repo
- link `~/.tmux.conf` to `tmux.conf`
- create folders like `~/.config/nvim` or `~/.local/bin`
- run setup commands such as `git submodule update --init --recursive`

## Quick Start

### 1. Build The Binary

From this repository:

```bash
just build
```

This creates:

```text
bin/dotbot
```

If you do not have `just`, you can build with Go directly:

```bash
mkdir -p bin
go build -buildvcs=false -o bin/dotbot ./cmd/dotbot-go
```

### 2. Create A Config File

Create `install.conf.yaml` in your dotfiles repository:

```yaml
- defaults:
    link:
      create: true
      relink: true

- clean:
    - "~"

- create:
    - "~/.config"
    - "~/.local/bin"

- link:
    ~/.vimrc: vimrc
    ~/.tmux.conf: tmux.conf
    ~/.config/nvim:
      path: nvim

- shell:
    - [git submodule update --init --recursive, Installing submodules]
```

In plain English, this means:

- use sensible defaults for links
- clean broken symlinks in your home directory
- create a couple of folders
- link files from the repo into your home directory
- update Git submodules

### 3. Preview First

Always run a dry run first:

```bash
./bin/dotbot -c install.conf.yaml --dry-run
```

Dry run prints what would happen without changing your filesystem.

### 4. Apply The Config

When the preview looks correct:

```bash
./bin/dotbot -c install.conf.yaml
```

## Install From A Dotfiles Repo

A typical dotfiles repository looks like this:

```text
dotfiles/
├── install.conf.yaml
├── vimrc
├── tmux.conf
└── nvim/
```

Run `dotbot-go` from that repository:

```bash
cd ~/.dotfiles
dotbot-go -c install.conf.yaml
```

Or point to the repository explicitly:

```bash
dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

The `-d` option sets the base directory. Relative paths in your config, such as `vimrc` or `nvim`, are resolved from that directory.

## Supported Config Formats

`dotbot-go` supports these file formats:

| Format | Extensions | Best For |
| --- | --- | --- |
| YAML | `.yaml`, `.yml` | Most dotfiles users |
| JSON | `.json` | Strict machine-generated config |
| JSON5 | `.json5` | JSON with comments and trailing commas |
| TOML | `.toml` | Users who prefer TOML syntax |
| HOCON | `.conf`, `.hocon` | HOCON-based config workflows |

Complete examples:

- [YAML example](examples/install.conf.yaml)
- [JSON example](examples/install.json)
- [JSON5 example](examples/install.json5)
- [TOML example](examples/install.toml)
- [HOCON example](examples/install.hocon)

YAML, JSON, and JSON5 can use a top-level task list. TOML and HOCON should put the task list under `tasks`.

More detail: [Config Formats Guide](docs/CONFIG_FORMATS.md)

## Directives

Directives are the actions in your config file. They run in the order you write them.

| Directive | What It Does |
| --- | --- |
| `defaults` | Sets default options for later directives |
| `clean` | Removes broken symlinks from selected directories |
| `create` | Creates directories |
| `link` | Creates symlinks or hardlinks |
| `shell` | Runs shell commands |

More detail: [User Guide](docs/USER_GUIDE.md)

## Common Commands

Preview without changing files:

```bash
dotbot-go -c install.conf.yaml --dry-run
```

Run only link tasks:

```bash
dotbot-go -c install.conf.yaml --only link
```

Skip shell commands:

```bash
dotbot-go -c install.conf.yaml --except shell
```

Show more output:

```bash
dotbot-go -vv -c install.conf.yaml
```

Disable color:

```bash
dotbot-go -c install.conf.yaml --no-color
```

Force color when piping output:

```bash
dotbot-go -c install.conf.yaml --force-color
```

Stop after the first failed directive:

```bash
dotbot-go -c install.conf.yaml --exit-on-failure
```

## Safe Linking Rules

`dotbot-go` will not overwrite a normal file or directory by default.

If a destination already exists:

- use `backup: true` to rename the existing file before linking
- use `force: true` only when you intentionally want to remove the existing path
- use `relink: true` to replace an existing symlink

Example:

```yaml
- link:
    ~/.vimrc:
      path: vimrc
      backup: true
```

## Troubleshooting

If something looks wrong, start here:

```bash
dotbot-go -c install.conf.yaml --dry-run -vv
```

Common problems:

| Problem | What To Check |
| --- | --- |
| `unsupported config file format` | File extension must be one of the supported extensions |
| `configuration file must be a list of tasks` | YAML/JSON/JSON5 should be a list, TOML/HOCON should use `tasks` |
| Link target does not exist | Make sure the source file exists in your dotfiles repo |
| Existing file blocks a link | Add `backup: true`, move the file manually, or intentionally use `force: true` |
| Shell command fails | Run with `-vv` to see command output |

More detail: [Troubleshooting Guide](docs/TROUBLESHOOTING.md)

## For Developers

Useful commands:

```bash
just --list
just verify
just vulncheck
```

Without `just`:

```bash
gofmt -w .
go mod tidy
golangci-lint run ./...
go test ./...
go test -race ./...
go vet ./...
GOTOOLCHAIN=go1.26.4+auto go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go build -buildvcs=false -o bin/dotbot ./cmd/dotbot-go
```

## Releases

GitHub Actions tests and builds the project on pushes and pull requests. To publish a release from the GitHub UI, run the `Test, Build, and Release` workflow manually, enter the version, and choose `release` or `prerelease`. The workflow creates the `v` tag, tests and builds that tag, and publishes the GitHub Release with Linux and macOS archives plus checksums.

Tags pushed directly still publish releases. When you push a tag that starts with `v`, for example `v0.2.1`, the workflow publishes a stable GitHub Release. Tags with prerelease suffixes, such as `v0.2.1-rc.1`, are published as prereleases.

Example:

```bash
git tag v0.2.1
git push origin v0.2.1
```
