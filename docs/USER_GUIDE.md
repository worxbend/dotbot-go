# User Guide

This guide explains how to use `dotbot-go` from zero.

## The Basic Idea

Your dotfiles repository contains the real files. Your home directory contains links to those files.

For example:

```text
~/.dotfiles/vimrc      real file in your repo
~/.vimrc              symlink that points to ~/.dotfiles/vimrc
```

This lets you edit files in one Git repository and install them on any machine.

## Step 1: Create A Dotfiles Repository

Example:

```bash
mkdir -p ~/.dotfiles
cd ~/.dotfiles
git init
```

Add a few files:

```bash
touch vimrc
touch tmux.conf
mkdir -p nvim
```

## Step 2: Add `install.conf.yaml`

Create this file:

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

## Step 3: Validate The Config

Run:

```bash
dotbot-go validate -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

Validation checks that the config can be read, that directive payloads have
supported shapes, and how many operations would be planned. It does not create
directories, links, or run shell commands.

## Step 4: Inspect Planned Operations

Run:

```bash
dotbot-go plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

For scripts and automation, use JSON:

```bash
dotbot-go plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --output json
```

Planning prints the operation list without creating directories, links, or
running shell commands.

## Step 5: Preview Changes

Run:

```bash
dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run
```

Read the output. It should say what would be created, linked, cleaned, or run.

## Step 6: Apply Changes

Run:

```bash
dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

## How Paths Work

In a `link` directive:

```yaml
- link:
    ~/.vimrc: vimrc
```

The left side is the destination. This is where the link will appear.

```text
~/.vimrc
```

The right side is the source. It is relative to the base directory.

```text
vimrc
```

If your base directory is `~/.dotfiles`, then `vimrc` means:

```text
~/.dotfiles/vimrc
```

## Link Options

Simple form:

```yaml
- link:
    ~/.tmux.conf: tmux.conf
```

Detailed form:

```yaml
- link:
    ~/.vimrc:
      path: vimrc
      create: true
      relink: true
      backup: true
```

Common options:

| Option | Meaning |
| --- | --- |
| `path` | Source path in your dotfiles repo |
| `create` | Create parent folders for the destination |
| `relink` | Replace an existing symlink |
| `backup` | Rename an existing normal file before linking |
| `force` | Remove an existing destination before linking |
| `relative` | Create relative symlinks |
| `ignore-missing` | Do not fail if the source path does not exist |

Use `backup: true` for normal existing files. Use `force: true` only when deleting the existing path is intentional.

## Create Directories

Simple list:

```yaml
- create:
    - "~/.config"
    - "~/.local/bin"
```

With a mode:

```yaml
- create:
    ~/.ssh:
      mode: 448
```

`448` is decimal for Unix mode `0700`.

## Clean Broken Symlinks

```yaml
- clean:
    - "~"
```

This removes broken symlinks under your home directory. It does not delete normal files.

## Run Shell Commands

Short form:

```yaml
- shell:
    - [git submodule update --init --recursive, Installing submodules]
```

Detailed form:

```yaml
- shell:
    - command: "git submodule update --init --recursive"
      description: "Installing submodules"
      quiet: false
```

Shell commands run from the base directory. On Unix they use `$SHELL -c`. On Windows they use `cmd /C`.

## Run Only Part Of The Config

Run only links:

```bash
dotbot-go -c install.conf.yaml --only link
```

Skip shell commands:

```bash
dotbot-go -c install.conf.yaml --except shell
```

## Recommended First Config

Start small:

```yaml
- defaults:
    link:
      create: true
      relink: true

- link:
    ~/.vimrc: vimrc
```

Preview:

```bash
dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run
```

Apply:

```bash
dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

Then add more files one by one.
