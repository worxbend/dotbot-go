# User Guide

This guide explains how to use `dotbot-go` from zero. Script installation, release archives, and source builds provide `dotbot`. Snap provides `dotbot-go`; use that name instead in the commands below.

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
dotbot validate -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

Validation checks that the config can be read and that directive payloads have
supported shapes. It also builds the plan to surface planning errors, but it
does not create directories, links, or run shell commands.

## Step 4: Inspect Planned Operations

Run:

```bash
dotbot plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

For scripts and automation, use JSON:

```bash
dotbot plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --output json
```

Planning prints the operation list without creating directories, links, or
running shell commands.

## Step 5: Preview Changes

Run:

```bash
dotbot -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run
```

Read the output. It should say what would be created, linked, cleaned, or run. Dry run does not execute shell commands, including `link.if` conditions. For links with `link.if`, it reports the condition and destination, then skips their filesystem preview. The outcome remains unknown until an actual run.

## Step 6: Apply Changes

Run:

```bash
dotbot -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
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
| `if` | Run a shell command and link only if it exits successfully; dry run does not evaluate it |

Use `backup: true` for normal existing files. Use `force: true` only when deleting the existing path is intentional.

With `backup: true`, the original file or directory moves into a new private directory named `<destination>.dotbot-backup.<timestamp>-<random>/<basename>`. For example, a backup of `~/.vimrc` is stored as `.vimrc` inside a sibling `.vimrc.dotbot-backup.*` directory. Existing backups are preserved. If reserving the directory or moving the original fails, the original destination remains in place. To restore it, remove the replacement link and move the saved file or directory back to the original destination.

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

This checks immediate entries in your home directory and removes only broken symlinks whose targets are inside the base directory. It does not delete normal files.

Enable `recursive` to include subdirectories. Directory symlinks are not traversed:

```yaml
- clean:
    "~":
      recursive: true
```

Set `force: true` in the same options map to also remove broken symlinks pointing outside the base directory. Both options default to `false`.

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

Shell commands run from the base directory. On Unix they use `$SHELL -c`, falling back to `/bin/sh` when `$SHELL` is empty. On Windows they use `cmd /C`.

Each shell command and `link.if` condition has a 10-minute timeout. Commands that exceed it fail. Neither CLI flags nor config options can override this limit.

## Run Only Part Of The Config

Run only links:

```bash
dotbot -c install.conf.yaml --only link
```

Skip shell commands:

```bash
dotbot -c install.conf.yaml --except shell
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
dotbot -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run
```

Apply:

```bash
dotbot -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml
```

Then add more files one by one.
